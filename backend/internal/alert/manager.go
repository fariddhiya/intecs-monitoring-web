package alert

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/intecs/iot-monitoring/backend/internal/mqtt"
)

type AlertType string

const (
	LowFuel       AlertType = "LOW_FUEL"
	CriticalFuel  AlertType = "CRITICAL_FUEL"
	HighTemp      AlertType = "HIGH_TEMPERATURE"
	DeviceOffline AlertType = "DEVICE_OFFLINE"
)

var SeverityMap = map[AlertType]string{
	LowFuel:       "warning",
	CriticalFuel:  "critical",
	HighTemp:      "high",
	DeviceOffline: "medium",
}

type Manager struct {
	db                  *sql.DB
	highTempThreshold   float64
	alertCooldown       time.Duration
	lastAlertTimes      map[string]time.Time
}

func NewManager(db *sql.DB, highTempThreshold string) *Manager {
	var threshold float64
	fmt.Sscanf(highTempThreshold, "%f", &threshold)
	return &Manager{
		db:                db,
		highTempThreshold: threshold,
		alertCooldown:     30 * time.Second,
		lastAlertTimes:    make(map[string]time.Time),
	}
}

func (m *Manager) CheckAlerts(message mqtt.Message) {
	tel, err := mqtt.ParseMessage(message.Payload())
	if err != nil {
		log.Printf("Error parsing telemetry for alert check: %v", err)
		return
	}

	m.checkLowFuel(tel)
	m.checkCriticalFuel(tel)
	m.checkHighTemperature(tel)
}

func (m *Manager) canCreateAlert(deviceID string, alertType AlertType) bool {
	key := fmt.Sprintf("%s:%s", deviceID, alertType)
	lastTime, exists := m.lastAlertTimes[key]
	if !exists {
		return true
	}
	return time.Since(lastTime) > m.alertCooldown
}

func (m *Manager) recordAlert(deviceID string, alertType AlertType) {
	key := fmt.Sprintf("%s:%s", deviceID, alertType)
	m.lastAlertTimes[key] = time.Now()
}

func (m *Manager) checkLowFuel(tel *mqtt.Telemetry) {
	if tel.FuelPercent >= 20 || tel.FuelPercent < 0 {
		return
	}
	if !m.canCreateAlert(tel.DeviceID, LowFuel) {
		return
	}

	if err := m.createAlert(tel.DeviceID, LowFuel, fmt.Sprintf("Fuel level low: %.1f%%", tel.FuelPercent)); err != nil {
		log.Printf("Error creating LOW_FUEL alert: %v", err)
		return
	}
	m.recordAlert(tel.DeviceID, LowFuel)
	log.Printf("ALERT: Low fuel for %s (%.1f%%)", tel.DeviceID, tel.FuelPercent)
}

func (m *Manager) checkCriticalFuel(tel *mqtt.Telemetry) {
	if tel.FuelPercent >= 10 || tel.FuelPercent < 0 {
		return
	}
	if !m.canCreateAlert(tel.DeviceID, CriticalFuel) {
		return
	}

	if err := m.createAlert(tel.DeviceID, CriticalFuel, fmt.Sprintf("Fuel level critical: %.1f%%", tel.FuelPercent)); err != nil {
		log.Printf("Error creating CRITICAL_FUEL alert: %v", err)
		return
	}
	m.recordAlert(tel.DeviceID, CriticalFuel)
	log.Printf("ALERT: Critical fuel for %s (%.1f%%)", tel.DeviceID, tel.FuelPercent)
}

func (m *Manager) checkHighTemperature(tel *mqtt.Telemetry) {
	if tel.Temperature <= m.highTempThreshold {
		return
	}
	if !m.canCreateAlert(tel.DeviceID, HighTemp) {
		return
	}

	if err := m.createAlert(tel.DeviceID, HighTemp, fmt.Sprintf("Temperature high: %.1f°C", tel.Temperature)); err != nil {
		log.Printf("Error creating HIGH_TEMPERATURE alert: %v", err)
		return
	}
	m.recordAlert(tel.DeviceID, HighTemp)
	log.Printf("ALERT: High temperature for %s (%.1f°C)", tel.DeviceID, tel.Temperature)
}

func (m *Manager) createAlert(deviceID string, alertType AlertType, message string) error {
	severity := SeverityMap[alertType]
	_, err := m.db.Exec(
		`INSERT INTO alerts (device_id, type, severity, message, status)
		VALUES ($1, $2, $3, $4, 'active')`,
		deviceID, string(alertType), severity, message,
	)
	return err
}
