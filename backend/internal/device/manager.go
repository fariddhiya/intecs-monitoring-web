package device

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/intecs/iot-monitoring/backend/internal/mqtt"
)

const (
	StatusOnline    = "ONLINE"
	StatusStale     = "STALE"
	StatusOffline   = "OFFLINE"
	DefaultSiteName = "Sangatta Site"
)

type Manager struct {
	db              *sql.DB
	staleAfter      time.Duration
	unavailableAt   time.Duration
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db:              db,
		staleAfter:      time.Minute,
		unavailableAt:   time.Minute * 5,
	}
}

func (m *Manager) HandleTelemetry(message mqtt.Message) {
	tel, err := mqtt.ParseMessage(message.Payload())
	if err != nil {
		log.Printf("Error parsing telemetry: %v", err)
		return
	}

	siteID, deviceID, err := mqtt.ExtractDeviceInfo(message.Topic())
	if err != nil {
		log.Printf("Error extracting device info from topic %s: %v", message.Topic(), err)
		return
	}

	ctx := context.Background()

	if err := m.upsertSite(ctx, siteID); err != nil {
		log.Printf("Error upserting site: %v", err)
	}

	if err := m.upsertDevice(ctx, deviceID, siteID); err != nil {
		log.Printf("Error upserting device: %v", err)
	}

	if err := m.saveTelemetry(ctx, tel); err != nil {
		log.Printf("Error saving telemetry: %v", err)
	}

	if err := m.updateLastSeen(ctx, deviceID); err != nil {
		log.Printf("Error updating device status: %v", err)
	}
}

func (m *Manager) upsertSite(ctx context.Context, siteID string) error {
	var existingID sql.NullString
	err := m.db.QueryRowContext(ctx, 
		"SELECT id FROM sites WHERE site_id = $1", siteID).Scan(&existingID)
	
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existingID.Valid {
		return nil
	}

	_, err = m.db.ExecContext(ctx,
		"INSERT INTO sites (name, site_id) VALUES ($1, $2)", DefaultSiteName, siteID)
	return err
}

func (m *Manager) upsertDevice(ctx context.Context, deviceID, siteID string) error {
	var siteUUID string
	if err := m.db.QueryRowContext(ctx, "SELECT id FROM sites WHERE site_id = $1", siteID).Scan(&siteUUID); err != nil {
		return err
	}

	var exists bool
	err := m.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM devices WHERE device_id = $1)", deviceID).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		_, err = m.db.ExecContext(ctx,
			"INSERT INTO devices (device_id, site_id, name, type) VALUES ($1, $2, $3, 'Industrial Equipment')",
			deviceID, siteUUID, deviceID)
		return err
	}
	return nil
}

func (m *Manager) updateLastSeen(ctx context.Context, deviceID string) error {
	_, err := m.db.ExecContext(ctx, "UPDATE devices SET last_seen = NOW() WHERE device_id = $1", deviceID)
	return err
}

func (m *Manager) saveTelemetry(ctx context.Context, tel *mqtt.Telemetry) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO telemetry (device_id, timestamp, fuel_percentage, fuel_level, temperature, flow_rate, equipment_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tel.DeviceID, tel.Timestamp, tel.FuelPercent, tel.FuelLevel,
		tel.Temperature, tel.FlowRate, tel.EquipStatus,
	)
	return err
}
