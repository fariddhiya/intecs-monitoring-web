package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

type Telemetry struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   string  `json:"timestamp"`
	FuelPercent float64 `json:"fuel_percentage"`
	FuelLevel   float64 `json:"fuel_level"`
	Temperature float64 `json:"temperature"`
	FlowRate    float64 `json:"flow_rate"`
	EquipStatus string  `json:"equipment_status"`
}

type DeviceState struct {
	DeviceID      string
	FuelPercent   float64
	FuelLevel     float64
	Temperature   float64
	FlowRate      float64
	EquipStatus   string
	StatusChangeTimer int
	RunTimer      int
}

var equipmentStatuses = []string{"running", "idle", "maintenance"}

func main() {
	_ = godotenv.Load()

	deviceCount := getEnvInt("DEVICE_COUNT", 10)
	publishInterval := getEnvDuration("PUBLISH_INTERVAL", 5*time.Second)
	siteID := getEnvOrDefault("SITE_ID", "sangatta")

	log.Printf("Starting IoT Simulator with %d devices, interval=%v, site=%s", deviceCount, publishInterval, siteID)

	brokerURL := getEnvOrDefault("MQTT_BROKER_URL", "tcp://localhost:1883")
	mqttUser := getEnvOrDefault("MQTT_USERNAME", "intecs")
	mqttPass := getEnvOrDefault("MQTT_PASSWORD", "intecs123")

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID("intecs-simulator").
		SetUsername(mqttUser).
		SetPassword(mqttPass)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.WaitTimeout(10 * time.Second)
	if token.Error != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error)
	}

	states := make([]DeviceState, deviceCount)
	for i := 0; i < deviceCount; i++ {
		deviceID := fmt.Sprintf("DT-%03d", i+1)
		states[i] = DeviceState{
			DeviceID:      deviceID,
			FuelPercent:   50 + rand.Float64()*40,
			FuelLevel:     2500 + rand.Float64()*2500,
			Temperature:   70 + rand.Float64()*20,
			FlowRate:      20 + rand.Float64()*30,
			EquipStatus:   "running",
			RunTimer:      rand.Intn(10),
		}
		go simulateDevice(&states[i], siteID, client, publishInterval)
	}

	log.Println("Simulator running... Press Ctrl+C to stop")
	select {}
}

func simulateDevice(state *DeviceState, siteID string, client mqtt.Client, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		updateState(state)
		payload := Telemetry{
			DeviceID:    state.DeviceID,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			FuelPercent: math.Round(state.FuelPercent*100) / 100,
			FuelLevel:   math.Round(state.FuelLevel*100) / 100,
			Temperature: math.Round(state.Temperature*100) / 100,
			FlowRate:    math.Round(state.FlowRate*100) / 100,
			EquipStatus: state.EquipStatus,
		}

		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Error marshaling telemetry for %s: %v", state.DeviceID, err)
			continue
		}

		topic := fmt.Sprintf("intecs/site/%s/device/%s/telemetry", siteID, state.DeviceID)
		token := client.Publish(topic, 1, false, data)
		token.WaitTimeout(3 * time.Second)
		if token.Error != nil {
			log.Printf("Error publishing to %s: %v", topic, token.Error)
		}
	}
}

func updateState(state *DeviceState) {
	state.RunTimer++
	state.StatusChangeTimer++

	// Fuel decreases slowly over time (simulates usage)
	fuelDecay := 0.05 + rand.Float64()*0.1
	state.FuelPercent -= fuelDecay
	if state.FuelPercent < 5 {
		state.FuelPercent = 5 + rand.Float64()*5
	}
	state.FuelLevel = state.FuelPercent * 100

	// Temperature fluctuates in realistic range
	tempChange := (rand.Float64() - 0.5) * 2
	state.Temperature += tempChange
	if state.Temperature < 60 {
		state.Temperature = 60 + rand.Float64()*10
	} else if state.Temperature > 95 {
		state.Temperature = 70 + rand.Float64()*10
	}

	// Flow rate varies based on status
	switch state.EquipStatus {
	case "running":
		state.FlowRate = 25 + rand.Float64()*30
	case "idle":
		state.FlowRate = 5 + rand.Float64()*10
	case "maintenance":
		state.FlowRate = 0
	}

	// Equipment status changes occasionally
	if state.StatusChangeTimer >= 30+rand.Intn(30) {
		state.StatusChangeTimer = 0
		currentIdx := indexOf(equipmentStatuses, state.EquipStatus)
		nextIdx := (currentIdx + 1) % len(equipmentStatuses)
		if rand.Float64() > 0.3 {
			nextIdx = rand.Intn(len(equipmentStatuses))
		}
		state.EquipStatus = equipmentStatuses[nextIdx]
	}
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return 0
}

func getEnvOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
