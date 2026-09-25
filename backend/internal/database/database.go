package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func New(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to PostgreSQL successfully")
	return db, nil
}

func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			site_id TEXT UNIQUE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			device_id TEXT UNIQUE NOT NULL,
			site_id UUID REFERENCES sites(id),
			name TEXT,
			type TEXT,
			status TEXT DEFAULT 'OFFLINE',
			last_seen TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			device_id TEXT NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			fuel_percentage NUMERIC(5,2),
			fuel_level NUMERIC(10,2),
			temperature NUMERIC(5,2),
			flow_rate NUMERIC(8,2),
			equipment_status TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			device_id TEXT NOT NULL,
			type TEXT NOT NULL,
			severity TEXT NOT NULL,
			message TEXT NOT NULL,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			resolved_at TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_device_timestamp ON telemetry(device_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute query: %w\nQuery: %s", err, q)
		}
	}

	// Seed site if not exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sites WHERE site_id = $1", "sangatta").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO sites (name, site_id) VALUES ($1, $2)", "Sangatta Site", "sangatta")
	}

	log.Println("Database migration completed")
	return nil
}
