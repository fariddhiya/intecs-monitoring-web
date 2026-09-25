package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/intecs/iot-monitoring/backend/internal/alert"
	"github.com/intecs/iot-monitoring/backend/internal/api"
	"github.com/intecs/iot-monitoring/backend/internal/database"
	"github.com/intecs/iot-monitoring/backend/internal/device"
	ws "github.com/intecs/iot-monitoring/backend/internal/websocket"
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

	router, hub := api.NewRouter(db, devMgr)
	alertMgr := alert.NewManager(db, hub, highTempThreshold)

	onMessage := func(c mqtt.Client, msg mqtt.Message) {
		devMgr.HandleTelemetry(msg)
		alertMgr.CheckAlerts(msg)

		payload := parsePayload(msg.Payload())
		if payload == nil {
			return
		}

		deviceID, _ := payload["device_id"].(string)
		timestamp, _ := payload["timestamp"].(string)

		hub.Broadcast(ws.MessageTypeTelemetry, map[string]interface{}{
			"device_id":         deviceID,
			"timestamp":         timestamp,
			"fuel_percent":      safeNum(payload["fuel_percentage"]),
			"fuel_level":        safeNum(payload["fuel_level"]),
			"temperature":       safeNum(payload["temperature"]),
			"flow_rate":         safeNum(payload["flow_rate"]),
			"equipment_status":  payload["equipment_status"],
		})
	}

	onConnect := func(c mqtt.Client) {
		log.Println("MQTT connected successfully")
		sub := "intecs/site/+/device/+/telemetry"
		token := c.Subscribe(sub, 1, onMessage)
		token.WaitTimeout(5 * time.Second)
		log.Printf("Subscribed to: %s", sub)
		hub.SetMQTTStatus(true, "")
	}

	onDisconnect := func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v. Will reconnect...", err)
		hub.SetMQTTStatus(false, "")
	}

	opts := mqtt.NewClientOptions().
		AddBroker(mqttURL).
		SetClientID("intecs-backend").
		SetUsername(mqttUser).
		SetPassword(mqttPass).
		SetKeepAlive(30).
		SetAutoReconnect(true).
		SetOnConnectHandler(onConnect).
		SetConnectionLostHandler(onDisconnect)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.WaitTimeout(10 * time.Second)
	if token.Error != nil {
		log.Printf("Warning: MQTT connection failed initially, will retry automatically: %v", token.Error)
	}

	log.Printf("REST API starting on :8080 with WebSocket support")
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

func parsePayload(payload []byte) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil
	}
	return m
}

func safeNum(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case nil:
		return 0
	default:
		return 0
	}
}
