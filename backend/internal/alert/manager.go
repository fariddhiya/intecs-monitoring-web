package alert

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	ws "github.com/intecs/iot-monitoring/backend/internal/websocket"
	"github.com/intecs/iot-monitoring/backend/internal/mqtt"
)

type AlertType string

const (
	LowFuel       AlertType = "LOW_FUEL"
	CriticalFuel  AlertType = "CRITICAL_FUEL"
	HighTemp      AlertType = "HIGH_TEMPERATURE"
	CriticalTemp  AlertType = "CRITICAL_TEMPERATURE"
	DeviceOffline AlertType = "DEVICE_OFFLINE"
)

var SeverityMap = map[AlertType]string{
	LowFuel:              "warning",
	CriticalFuel:         "critical",
	HighTemp:             "warning",
	CriticalTemp:         "critical",
	DeviceOffline:        "medium",
}

type DeviceThresholds struct {
	WarnTempThreshold   float64 `json:"warn_temp_threshold"`
	CritTempThreshold    float64 `json:"crit_temp_threshold"`
	WarnFuelThreshold    float64 `json:"warn_fuel_threshold"`
	CritFuelThreshold    float64 `json:"crit_fuel_threshold"`
	CooldownSeconds     int     `json:"cooldown_seconds"`
}

type Manager struct {
	db                  *sql.DB
	globalTempThreshold float64
	alertCooldown       time.Duration
	lastAlertTimes      map[string]time.Time
	mu                  sync.RWMutex
	wsHub               *ws.Hub
}

func NewManager(db *sql.DB, wsHub *ws.Hub, globalTempThreshold string) *Manager {
	var threshold float64
	fmt.Sscanf(globalTempThreshold, "%f", &threshold)
	return &Manager{
		db:                db,
		globalTempThreshold: threshold,
		wsHub:             wsHub,
		alertCooldown:     30 * time.Second,
		lastAlertTimes:    make(map[string]time.Time),
	}
}

func (m *Manager) SetHub(hub *ws.Hub) {
	m.wsHub = hub
}

func (m *Manager) GetDefaultThresholds() *DeviceThresholds {
	return &DeviceThresholds{
		WarnTempThreshold: 85,
		CritTempThreshold: 90,
		WarnFuelThreshold: 20,
		CritFuelThreshold: 10,
		CooldownSeconds:   30,
	}
}

func (m *Manager) GetThresholds(deviceID string) (*DeviceThresholds, error) {
	var t DeviceThresholds
	err := m.db.QueryRow(`
		SELECT warn_temp_threshold, crit_temp_threshold, warn_fuel_threshold, crit_fuel_threshold, cooldown_seconds
		FROM device_thresholds WHERE device_id = $1
	`, deviceID).Scan(&t.WarnTempThreshold, &t.CritTempThreshold, &t.WarnFuelThreshold, &t.CritFuelThreshold, &t.CooldownSeconds)
	
	if err == sql.ErrNoRows {
		return m.GetDefaultThresholds(), nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (m *Manager) UpsertThresholds(deviceID string, thresholds *DeviceThresholds) error {
	_, err := m.db.Exec(`
		INSERT INTO device_thresholds (device_id, warn_temp_threshold, crit_temp_threshold, warn_fuel_threshold, crit_fuel_threshold, cooldown_seconds)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id) DO UPDATE SET
			warn_temp_threshold = EXCLUDED.warn_temp_threshold,
			crit_temp_threshold = EXCLUDED.crit_temp_threshold,
			warn_fuel_threshold = EXCLUDED.warn_fuel_threshold,
			crit_fuel_threshold = EXCLUDED.crit_fuel_threshold,
			cooldown_seconds = EXCLUDED.cooldown_seconds,
			updated_at = NOW()
	`, deviceID, thresholds.WarnTempThreshold, thresholds.CritTempThreshold, 
	   thresholds.WarnFuelThreshold, thresholds.CritFuelThreshold, thresholds.CooldownSeconds)
	return err
}

func (m *Manager) CheckAlerts(message mqtt.Message) {
	tel, err := mqtt.ParseMessage(message.Payload())
	if err != nil {
		log.Printf("Error parsing telemetry for alert check: %v", err)
		return
	}

	m.checkFuel(tel)
	m.checkTemperature(tel)
}

func (m *Manager) canCreateAlert(deviceID string, alertType AlertType) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key := fmt.Sprintf("%s:%s", deviceID, alertType)
	lastTime, exists := m.lastAlertTimes[key]
	if !exists {
		return true
	}
	
	thresholds, _ := m.GetThresholds(deviceID)
	return time.Since(lastTime) > time.Duration(thresholds.CooldownSeconds)*time.Second
}

func (m *Manager) recordAlert(deviceID string, alertType AlertType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", deviceID, alertType)
	m.lastAlertTimes[key] = time.Now()
}

func (m *Manager) checkFuel(tel *mqtt.Telemetry) {
	deviceID := tel.DeviceID
	thresholds, err := m.GetThresholds(deviceID)
	if err != nil {
		log.Printf("Error fetching thresholds for %s: %v", deviceID, err)
		return
	}

	// Check critical fuel first (< 10%)
	if tel.FuelPercent < 0 || tel.FuelPercent >= thresholds.CritFuelThreshold {
		// No action needed or already covered by warning level
	} else if tel.FuelPercent < thresholds.WarnFuelThreshold {
		// Low fuel warning
		if !m.canCreateAlert(deviceID, LowFuel) {
			return
		}
		if err := m.createAlert(deviceID, LowFuel, fmt.Sprintf("Fuel level low: %.1f%% (below %.1f%%)", tel.FuelPercent, thresholds.WarnFuelThreshold)); err != nil {
			log.Printf("Error creating LOW_FUEL alert: %v", err)
			return
		}
		m.recordAlert(deviceID, LowFuel)
		log.Printf("ALERT: Low fuel for %s (%.1f%%)", deviceID, tel.FuelPercent)
	}

	// Check critical fuel (< 10%)
	if tel.FuelPercent >= 0 && tel.FuelPercent < thresholds.CritFuelThreshold {
		if !m.canCreateAlert(deviceID, CriticalFuel) {
			return
		}
		if err := m.createAlert(deviceID, CriticalFuel, fmt.Sprintf("Fuel level critical: %.1f%% (below %.1f%%)", tel.FuelPercent, thresholds.CritFuelThreshold)); err != nil {
			log.Printf("Error creating CRITICAL_FUEL alert: %v", err)
			return
		}
		m.recordAlert(deviceID, CriticalFuel)
		log.Printf("ALERT: Critical fuel for %s (%.1f%%)", deviceID, tel.FuelPercent)
	}
}

func (m *Manager) checkTemperature(tel *mqtt.Telemetry) {
	deviceID := tel.DeviceID
	thresholds, err := m.GetThresholds(deviceID)
	if err != nil {
		log.Printf("Error fetching thresholds for %s: %v", deviceID, err)
		return
	}

	temp := tel.Temperature

	// Check critical temperature first (> 90C or custom)
	if temp > thresholds.CritTempThreshold {
		if !m.canCreateAlert(deviceID, CriticalTemp) {
			return
		}
		if err := m.createAlert(deviceID, CriticalTemp, fmt.Sprintf("Temperature critical: %.1f°C (above %.1f°C)", temp, thresholds.CritTempThreshold)); err != nil {
			log.Printf("Error creating CRITICAL_TEMP alert: %v", err)
			return
		}
		m.recordAlert(deviceID, CriticalTemp)
		log.Printf("ALERT: Critical temperature for %s (%.1f°C)", deviceID, temp)
	} else if temp > thresholds.WarnTempThreshold {
		// Warning temperature (> 85C or custom)
		if !m.canCreateAlert(deviceID, HighTemp) {
			return
		}
		if err := m.createAlert(deviceID, HighTemp, fmt.Sprintf("Temperature high: %.1f°C (above %.1f°C)", temp, thresholds.WarnTempThreshold)); err != nil {
			log.Printf("Error creating HIGH_TEMP alert: %v", err)
			return
		}
		m.recordAlert(deviceID, HighTemp)
		log.Printf("ALERT: High temperature for %s (%.1f°C)", deviceID, temp)
	}
}

func (m *Manager) createAlert(deviceID string, alertType AlertType, message string) error {
	severity := SeverityMap[alertType]
	_, err := m.db.Exec(
		`INSERT INTO alerts (device_id, type, severity, message, status)
		VALUES ($1, $2, $3, $4, 'active')`,
		deviceID, string(alertType), severity, message,
	)
	if err != nil {
		return err
	}

	// Broadcast via WebSocket
	if m.wsHub != nil {
		m.wsHub.Broadcast(ws.MessageTypeAlert, map[string]interface{}{
			"device_id": deviceID,
			"type":      string(alertType),
			"severity":  severity,
			"message":   message,
		})
	}

	return nil
}
