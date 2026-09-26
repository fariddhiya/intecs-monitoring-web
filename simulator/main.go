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
	DeviceID          string
	FuelPercent       float64
	FuelLevel         float64
	Temperature       float64
	FlowRate          float64
	EquipStatus       string
	StatusChangeTimer int
	RunTimer          int
}

var equipmentStatuses = []string{"running", "idle", "maintenance"}

// SimulationConfig holds all configurable randomization parameters for telemetry simulation.
type SimulationConfig struct {
	FuelPercentMin    float64
	FuelPercentMax    float64
	FuelDecayMin      float64
	FuelDecayMax      float64
	TemperatureMin    float64
	TemperatureMax    float64
	TemperatureChange float64
	RunningFlowMin    float64
	RunningFlowMax    float64
	IdleFlowMin       float64
	IdleFlowMax       float64
	StatusChangeMin   int
	StatusChangeMax   int
}

func loadSimulationConfig() SimulationConfig {
	config := SimulationConfig{
		FuelPercentMin:    getEnvFloat("FUEL_PERCENT_MIN", 50),
		FuelPercentMax:    getEnvFloat("FUEL_PERCENT_MAX", 90),
		FuelDecayMin:      getEnvFloat("FUEL_DECAY_MIN", 0.05),
		FuelDecayMax:      getEnvFloat("FUEL_DECAY_MAX", 0.15),
		TemperatureMin:    getEnvFloat("TEMPERATURE_MIN", 60),
		TemperatureMax:    getEnvFloat("TEMPERATURE_MAX", 95),
		TemperatureChange: getEnvFloat("TEMPERATURE_CHANGE", 2),
		RunningFlowMin:    getEnvFloat("RUNNING_FLOW_MIN", 25),
		RunningFlowMax:    getEnvFloat("RUNNING_FLOW_MAX", 55),
		IdleFlowMin:       getEnvFloat("IDLE_FLOW_MIN", 5),
		IdleFlowMax:       getEnvFloat("IDLE_FLOW_MAX", 15),
		StatusChangeMin:   getEnvInt("STATUS_CHANGE_MIN", 30),
		StatusChangeMax:   getEnvInt("STATUS_CHANGE_MAX", 60),
	}

	if err := validateConfig(&config); err != nil {
		log.Fatalf("Invalid simulation configuration: %v", err)
	}

	return config
}

func validateConfig(config *SimulationConfig) error {
	type violation struct {
		field string
		msg   string
	}

	var violations []violation

	checkMinMax := func(fieldMin, fieldMax string, min, max float64) {
		if min < 0 {
			violations = append(violations, violation{fieldMin, fmt.Sprintf("%s=%.2f cannot be negative", fieldMin, min)})
		}
		if min > max {
			violations = append(violations, violation{fmt.Sprintf("%s vs %s", fieldMin, fieldMax), fmt.Sprintf("%s (%.2f) must not exceed %s (%.2f)", fieldMin, min, fieldMax, max)})
		}
	}

	checkMinMax("FuelPercentMin", "FuelPercentMax", config.FuelPercentMin, config.FuelPercentMax)
	checkMinMax("FuelDecayMin", "FuelDecayMax", config.FuelDecayMin, config.FuelDecayMax)
	checkMinMax("TemperatureMin", "TemperatureMax", config.TemperatureMin, config.TemperatureMax)
	checkMinMax("RunningFlowMin", "RunningFlowMax", config.RunningFlowMin, config.RunningFlowMax)
	checkMinMax("IdleFlowMin", "IdleFlowMax", config.IdleFlowMin, config.IdleFlowMax)

	if config.StatusChangeMin < 1 {
		violations = append(violations, violation{"StatusChangeMin", fmt.Sprintf("must be >= 1, got %d", config.StatusChangeMin)})
	}
	if config.StatusChangeMax < 1 {
		violations = append(violations, violation{"StatusChangeMax", fmt.Sprintf("must be >= 1, got %d", config.StatusChangeMax)})
	}
	if config.StatusChangeMin > config.StatusChangeMax {
		violations = append(violations, violation{"StatusChangeMin vs StatusChangeMax",
			fmt.Sprintf("%d must not exceed %d", config.StatusChangeMin, config.StatusChangeMax)})
	}

	if len(violations) > 0 {
		for _, v := range violations {
			fmt.Fprintf(os.Stderr, "  - %s: %s\n", v.field, v.msg)
		}
		return fmt.Errorf("configuration validation failed (%d violations)", len(violations))
	}

	return nil
}

func main() {
	_ = godotenv.Load()

	deviceCount := getEnvInt("DEVICE_COUNT", 10)
	publishInterval := getEnvDuration("PUBLISH_INTERVAL", 5*time.Second)
	siteID := getEnvOrDefault("SITE_ID", "sangatta")

	config := loadSimulationConfig()

	log.Printf("Starting IoT Simulator with %d devices, interval=%v, site=%s", deviceCount, publishInterval, siteID)
	printConfigSummary(config)

	brokerURL := getEnvOrDefault("MQTT_BROKER_URL", "tcp://localhost:1883")
	mqttUser := getEnvOrDefault("MQTT_USERNAME", "intecs")
	mqttPass := getEnvOrDefault("MQTT_PASSWORD", "intecs123")

	var client mqtt.Client

	for attempts := 0; ; attempts++ {
		opts := mqtt.NewClientOptions().
			AddBroker(brokerURL).
			SetClientID(fmt.Sprintf("intecs-simulator-%d", attempts+1)).
			SetUsername(mqttUser).
			SetPassword(mqttPass)

		client = mqtt.NewClient(opts)
		token := client.Connect()

		if token.WaitTimeout(15*time.Second) && client.IsConnected() {
			break
		}

		if client.IsConnected() {
			break
		}

		log.Printf("MQTT connect attempt %d failed, retrying in 3s...", attempts+1)
		time.Sleep(3 * time.Second)
	}

	log.Println("Simulator running... Press Ctrl+C to stop")

	states := make([]DeviceState, deviceCount)
	for i := 0; i < deviceCount; i++ {
		deviceID := fmt.Sprintf("DT-%03d", i+1)
		states[i] = DeviceState{
			DeviceID:    deviceID,
			FuelPercent: randomFloat(config.FuelPercentMin, config.FuelPercentMax),
			FuelLevel:   randomFloat(config.FuelPercentMin, config.FuelPercentMax) * 100,
			Temperature: randomFloat(config.TemperatureMin, config.TemperatureMax),
			FlowRate:    randomFloat(config.RunningFlowMin, config.RunningFlowMax),
			EquipStatus: "running",
			RunTimer:    rand.Intn(10),
		}
		go simulateDevice(&states[i], siteID, client, publishInterval, config)
	}

	select {}
}

func printConfigSummary(config SimulationConfig) {
	log.Println("Simulation configuration:")
	log.Printf("  Fuel: %.0f-%.0f%%", config.FuelPercentMin, config.FuelPercentMax)
	log.Printf("  Fuel decay: %.2f-%.2f/%%tick", config.FuelDecayMin, config.FuelDecayMax)
	log.Printf("  Temperature: %.0f-%.0f°C", config.TemperatureMin, config.TemperatureMax)
	log.Printf("  Running flow: %.0f-%.0f", config.RunningFlowMin, config.RunningFlowMax)
	log.Printf("  Idle flow: %.0f-%.0f", config.IdleFlowMin, config.IdleFlowMax)
	log.Printf("  Status change: %d-%d ticks", config.StatusChangeMin, config.StatusChangeMax)
}

func simulateDevice(state *DeviceState, siteID string, client mqtt.Client, interval time.Duration, config SimulationConfig) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		updateState(state, config)
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
		client.Publish(topic, 1, false, data)
	}
}

func updateState(state *DeviceState, config SimulationConfig) {
	state.RunTimer++
	state.StatusChangeTimer++

	fuelDecay := randomFloat(config.FuelDecayMin, config.FuelDecayMax)
	state.FuelPercent -= fuelDecay
	if state.FuelPercent < config.FuelPercentMin {
		state.FuelPercent = randomFloat(config.FuelPercentMin, config.FuelPercentMin+5)
	}
	state.FuelLevel = state.FuelPercent * 100

	tempChange := (rand.Float64() - 0.5) * 2 * config.TemperatureChange
	state.Temperature += tempChange
	if state.Temperature < config.TemperatureMin {
		state.Temperature = randomFloat(config.TemperatureMin, config.TemperatureMin+10)
	} else if state.Temperature > config.TemperatureMax {
		state.Temperature = randomFloat(config.TemperatureMin, config.TemperatureMin+10)
	}

	switch state.EquipStatus {
	case "running":
		state.FlowRate = randomFloat(config.RunningFlowMin, config.RunningFlowMax)
	case "idle":
		state.FlowRate = randomFloat(config.IdleFlowMin, config.IdleFlowMax)
	case "maintenance":
		state.FlowRate = 0
	}

	statusChangeInterval := config.StatusChangeMin + rand.Intn(config.StatusChangeMax-config.StatusChangeMin+1)
	if state.StatusChangeTimer >= statusChangeInterval {
		state.StatusChangeTimer = 0
		currentIdx := indexOf(equipmentStatuses, state.EquipStatus)
		nextIdx := (currentIdx + 1) % len(equipmentStatuses)
		if rand.Float64() > 0.3 {
			nextIdx = rand.Intn(len(equipmentStatuses))
		}
		state.EquipStatus = equipmentStatuses[nextIdx]
	}
}

func randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
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

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
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
