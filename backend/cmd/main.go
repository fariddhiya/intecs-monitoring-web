package main

import (
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/intecs/iot-monitoring/backend/internal/alert"
	"github.com/intecs/iot-monitoring/backend/internal/api"
	"github.com/intecs/iot-monitoring/backend/internal/database"
	"github.com/intecs/iot-monitoring/backend/internal/device"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := getEnvOrDefault("DATABASE_URL", "postgresql://intecs:intecs123@localhost:5432/intecs?sslmode=disable")
	mqttURL := getEnvOrDefault("MQTT_BROKER_URL", "tcp://localhost:1883")
	mqttUser := getEnvOrDefault("MQTT_USERNAME", "intecs")
	mqttPass := getEnvOrDefault("MQTT_PASSWORD", "intecs123")
	highTempThreshold := getEnvOrDefault("HIGH_TEMP_THRESHOLD", "90")

	db, err := database.New(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	devMgr := device.NewManager(db)
	alertMgr := alert.NewManager(db, highTempThreshold)

	opts := mqtt.NewClientOptions().
		AddBroker(mqttURL).
		SetClientID("intecs-backend").
		SetUsername(mqttUser).
		SetPassword(mqttPass).
		SetKeepAlive(30).
		SetAutoReconnect(true).
		SetOnConnectHandler(func(client mqtt.Client) {
			log.Println("MQTT connected successfully")
			sub := "intecs/site/+/device/+/telemetry"
			token := client.Subscribe(sub, 1, func(c mqtt.Client, msg mqtt.Message) {
				devMgr.HandleTelemetry(msg)
				alertMgr.CheckAlerts(msg)
			})
			token.Wait()
			log.Printf("Subscribed to: %s", sub)
		}).
		SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Printf("MQTT connection lost: %v. Will reconnect...", err)
		})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.WaitTimeout(10 * time.Second)
	if token.Error != nil {
		log.Printf("Warning: MQTT connection failed initially, will retry automatically: %v", token.Error)
	}

	router := api.NewRouter(db, devMgr, alertMgr)

	log.Printf("REST API starting on :8080")
	if err := api.Start(router, ":8080"); err != nil {
		log.Fatal(err)
	}
}

func getEnvOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
