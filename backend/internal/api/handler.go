package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/intecs/iot-monitoring/backend/internal/alert"
	"github.com/intecs/iot-monitoring/backend/internal/device"
	"github.com/intecs/iot-monitoring/backend/internal/websocket"
)

type Server struct {
	router   *mux.Router
	db       *sql.DB
	devMgr   *device.Manager
	alertMgr *alert.Manager
	wsHub    *websocket.Hub
}

func NewRouter(db *sql.DB, devMgr *device.Manager) (http.Handler, *websocket.Hub) {
	hub := websocket.NewHub()
	go hub.Run()

	s := &Server{
		router:   mux.NewRouter(),
		db:       db,
		devMgr:   devMgr,
		wsHub:    hub,
	}
	s.routes()
	return s.router, hub
}

func (s *Server) SetAlertManager(alertMgr *alert.Manager) {
	s.alertMgr = alertMgr
}

func (s *Server) routes() {
	api := s.router.PathPrefix("/api").Subrouter()

	api.HandleFunc("/dashboard", s.handleDashboard).Methods("GET")
	api.HandleFunc("/devices", s.handleDevices).Methods("GET")
	api.HandleFunc("/devices/{id}", s.handleDeviceByID).Methods("GET")
	api.HandleFunc("/devices/{id}/telemetry", s.handleDeviceTelemetry).Methods("GET")
	api.HandleFunc("/alerts", s.handleAlerts).Methods("GET")
	api.HandleFunc("/alerts/history", s.handleAlertHistory).Methods("GET")
	api.HandleFunc("/alerts/{id}/acknowledge", s.handleAckAlert).Methods("POST")
	api.HandleFunc("/thresholds", s.handleListThresholds).Methods("GET")
	api.HandleFunc("/thresholds", s.handleUpdateThresholds).Methods("POST")
	api.HandleFunc("/ws", handleWebSocket(s.wsHub)).Methods("GET")
	api.HandleFunc("/mqtt/status", handleMQTTStatus(s.wsHub)).Methods("GET")
}

func Start(handler http.Handler, addr string) error {
	log.Printf("Starting server on %s", addr)
	return http.ListenAndServe(addr, handler)
}

// Dashboard response types
type DashboardResponse struct {
	TotalDevices   int    `json:"total_devices"`
	OnlineDevices  int    `json:"online_devices"`
	OfflineDevices int    `json:"offline_devices"`
	ActiveAlerts   int    `json:"active_alerts"`
	Devices        []DeviceView `json:"devices"`
}

type DeviceView struct {
	ID             string  `json:"id"`
	DeviceID       string  `json:"device_id"`
	Site           string  `json:"site"`
	FuelPercent    float64 `json:"fuel_percent"`
	Temperature    float64 `json:"temperature"`
	FlowRate       float64 `json:"flow_rate"`
	EquipStatus    string  `json:"equipment_status"`
	Connection     string  `json:"connection"`
	LastSeen       string  `json:"last_seen"`
}

type AlertView struct {
	ID        string `json:"id"`
	DeviceID  string `json:"device_id"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func computeConnection(lastSeen time.Time) string {
	now := time.Now()
	diff := now.Sub(lastSeen)
	if diff < time.Minute {
		return device.StatusOnline
	}
	if diff < time.Minute*5 {
		return device.StatusStale
	}
	return device.StatusOffline
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	var totalDevices, onlineDevices, offlineDevices, activeAlerts int

	s.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalDevices)
	s.db.QueryRow("SELECT COUNT(*) FROM alerts WHERE status = 'active'").Scan(&activeAlerts)

	rows, err := s.db.Query(`
		SELECT d.device_id, t.fuel_percentage, t.temperature, t.flow_rate, t.equipment_status, d.last_seen
		FROM devices d
		LEFT JOIN LATERAL (
			SELECT fuel_percentage, temperature, flow_rate, equipment_status
			FROM telemetry WHERE device_id = d.device_id ORDER BY timestamp DESC LIMIT 1
		) t ON TRUE
		ORDER BY d.device_id
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []DeviceView
	for rows.Next() {
		var d DeviceView
		var lastSeen time.Time
		var fuelPct, temp, flow sql.NullFloat64
		var equipStatus string

		err := rows.Scan(&d.DeviceID, &fuelPct, &temp, &flow, &equipStatus, &lastSeen)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		d.FuelPercent = safeFloat64(fuelPct.Float64)
		d.Temperature = safeFloat64(temp.Float64)
		d.FlowRate = safeFloat64(flow.Float64)
		d.EquipStatus = equipStatus
		d.LastSeen = lastSeen.Format(time.RFC3339)
		d.Connection = computeConnection(lastSeen)
		d.ID = d.DeviceID
		d.Site = "Sangatta"

		conn := computeConnection(lastSeen)
		if conn == device.StatusOnline || conn == device.StatusStale {
			onlineDevices++
		} else {
			offlineDevices++
		}

		devices = append(devices, d)
	}

	resp := DashboardResponse{
		TotalDevices:   totalDevices,
		OnlineDevices:  onlineDevices,
		OfflineDevices: offlineDevices,
		ActiveAlerts:   activeAlerts,
		Devices:        devices,
	}

	writeJSON(w, resp)
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT d.device_id, t.fuel_percentage, t.temperature, t.flow_rate, t.equipment_status, d.last_seen
		FROM devices d
		LEFT JOIN LATERAL (
			SELECT fuel_percentage, temperature, flow_rate, equipment_status
			FROM telemetry WHERE device_id = d.device_id ORDER BY timestamp DESC LIMIT 1
		) t ON TRUE
		ORDER BY d.device_id
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []DeviceView
	for rows.Next() {
		var d DeviceView
		var lastSeen time.Time
		var fuelPct, temp, flow sql.NullFloat64
		var equipStatus string

		err := rows.Scan(&d.DeviceID, &fuelPct, &temp, &flow, &equipStatus, &lastSeen)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		d.FuelPercent = safeFloat64(fuelPct.Float64)
		d.Temperature = safeFloat64(temp.Float64)
		d.FlowRate = safeFloat64(flow.Float64)
		d.EquipStatus = equipStatus
		d.Connection = computeConnection(lastSeen)
		d.LastSeen = lastSeen.Format(time.RFC3339)
		d.ID = d.DeviceID
		d.Site = "Sangatta"

		devices = append(devices, d)
	}

	writeJSON(w, devices)
}

func (s *Server) handleDeviceByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	row := s.db.QueryRow(`
		SELECT d.device_id, s.name, d.status, COALESCE(t.fuel_percentage, 0), COALESCE(t.fuel_level, 0), COALESCE(t.temperature, 0), 
			   COALESCE(t.flow_rate, 0), t.equipment_status, t.timestamp, d.last_seen
		FROM devices d
		LEFT JOIN sites s ON d.site_id = s.id
		LEFT JOIN (SELECT * FROM telemetry WHERE device_id = $1 ORDER BY timestamp DESC LIMIT 1) t ON TRUE
	`, deviceID)

	var resp struct {
		DeviceID    string  `json:"device_id"`
		Site        string  `json:"site"`
		Status      string  `json:"status"`
		FuelPercent float64 `json:"fuel_percent"`
		FuelLevel   float64 `json:"fuel_level"`
		Temperature float64 `json:"temperature"`
		FlowRate    float64 `json:"flow_rate"`
		EquipStatus string  `json:"equipment_status"`
		LastTelTime string  `json:"last_telemetry_time"`
		LastSeen    string  `json:"last_seen"`
		Connection  string  `json:"connection"`
	}

	var lastSeenTime time.Time
	err := row.Scan(&resp.DeviceID, &resp.Site, &resp.Status, &resp.FuelPercent, &resp.FuelLevel, &resp.Temperature, &resp.FlowRate, &resp.EquipStatus, &resp.LastTelTime, &lastSeenTime)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, fmt.Sprintf("Device %s not found", deviceID), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.LastSeen = lastSeenTime.Format(time.RFC3339)
	resp.Connection = computeConnection(lastSeenTime)
	writeJSON(w, resp)
}

func (s *Server) handleDeviceTelemetry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deviceID := vars["id"]

	rangeStr := r.URL.Query().Get("range")
	limitStr := r.URL.Query().Get("limit")

	timeRange, pointsPerRange := parseTimeRange(rangeStr)

	limit := pointsPerRange
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	now := time.Now()
	startTime := now.Add(-timeRange)

	rows, err := s.db.Query(
		"SELECT timestamp, fuel_percentage, fuel_level, temperature, flow_rate FROM telemetry WHERE device_id = $1 AND timestamp >= $2 ORDER BY timestamp ASC LIMIT $3",
		deviceID, startTime, limit,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type TelPoint struct {
		Timestamp   string  `json:"timestamp"`
		FuelPercent float64 `json:"fuel_percent"`
		FuelLevel   float64 `json:"fuel_level"`
		Temperature float64 `json:"temperature"`
		FlowRate    float64 `json:"flow_rate"`
	}

	var telements []TelPoint
	for rows.Next() {
		var tp TelPoint
		var ts time.Time
		var fuelPct, fuelLvl, temp, flow sql.NullFloat64
		if err := rows.Scan(&ts, &fuelPct, &fuelLvl, &temp, &flow); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tp.Timestamp = ts.Format(time.RFC3339)
		tp.FuelPercent = safeFloat64(fuelPct.Float64)
		tp.FuelLevel = safeFloat64(fuelLvl.Float64)
		tp.Temperature = safeFloat64(temp.Float64)
		tp.FlowRate = safeFloat64(flow.Float64)
		telements = append(telements, tp)
	}

	writeJSON(w, telements)
}

func parseTimeRange(rangeStr string) (time.Duration, int) {
	switch rangeStr {
	case "1h":
		return time.Hour, 60
	case "6h":
		return 6 * time.Hour, 72
	case "24h":
		return 24 * time.Hour, 288
	case "7d":
		return 7 * 24 * time.Hour, 1008
	default:
		return 24 * time.Hour, 288
	}
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := "SELECT id, device_id, type, severity, message, status, created_at FROM alerts"
	args := []interface{}{}
	argCount := 1

	if status != "" {
		query += fmt.Sprintf(" WHERE status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var alerts []AlertView
	for rows.Next() {
		var a AlertView
		var createdAt time.Time
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Type, &a.Severity, &a.Message, &a.Status, &createdAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.CreatedAt = createdAt.Format(time.RFC3339)
		alerts = append(alerts, a)
	}

	writeJSON(w, alerts)
}

func (s *Server) handleAckAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	_, err := s.db.Exec("UPDATE alerts SET status = 'acknowledged', resolved_at = NOW() WHERE id = $1 AND status = 'active'", alertID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAlertHistory(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	query := `SELECT id, device_id, type, severity, message, status, created_at, resolved_at FROM alerts`
	args := []interface{}{}
	argCount := 1

	var conditions []string
	if deviceID != "" {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", argCount))
		args = append(args, deviceID)
		argCount++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argCount))
		args = append(args, status)
		argCount++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AlertHistoryView struct {
		ID        string     `json:"id"`
		DeviceID  string     `json:"device_id"`
		Type      string     `json:"type"`
		Severity  string     `json:"severity"`
		Message   string     `json:"message"`
		Status    string     `json:"status"`
		CreatedAt string     `json:"created_at"`
		ResolvedAt *string   `json:"resolved_at,omitempty"`
	}

	var alerts []AlertHistoryView
	for rows.Next() {
		var a AlertHistoryView
		var createdAt time.Time
		var resolvedAt sql.NullTime
		if err := rows.Scan(&a.ID, &a.DeviceID, &a.Type, &a.Severity, &a.Message, &a.Status, &createdAt, &resolvedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		a.CreatedAt = createdAt.Format(time.RFC3339)
		if resolvedAt.Valid {
			r := resolvedAt.Time.Format(time.RFC3339)
			a.ResolvedAt = &r
		}
		alerts = append(alerts, a)
	}

	writeJSON(w, alerts)
}

func (s *Server) handleListThresholds(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")

	if deviceID == "" {
		rows, err := s.db.Query(`
			SELECT device_id, warn_temp_threshold, crit_temp_threshold, 
			       warn_fuel_threshold, crit_fuel_threshold, cooldown_seconds
			FROM device_thresholds ORDER BY device_id
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type ThresholdListEntry struct {
			DeviceID        string  `json:"device_id"`
			WarnTemp        float64 `json:"warn_temp_threshold"`
			CritTemp        float64 `json:"crit_temp_threshold"`
			WarnFuel        float64 `json:"warn_fuel_threshold"`
			CritFuel        float64 `json:"crit_fuel_threshold"`
			CooldownSeconds int     `json:"cooldown_seconds"`
		}

		var entries []ThresholdListEntry
		for rows.Next() {
			var e ThresholdListEntry
			if err := rows.Scan(&e.DeviceID, &e.WarnTemp, &e.CritTemp, &e.WarnFuel, &e.CritFuel, &e.CooldownSeconds); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			entries = append(entries, e)
		}
		writeJSON(w, entries)
		return
	}

	t, err := s.alertMgr.GetThresholds(deviceID)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"warn_temp_threshold": 85,
			"crit_temp_threshold": 90,
			"warn_fuel_threshold": 20,
			"crit_fuel_threshold": 10,
			"cooldown_seconds": 30,
		})
		return
	}
	writeJSON(w, t)
}

func (s *Server) handleUpdateThresholds(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeviceID        string  `json:"device_id"`
		WarnTempThreshold float64 `json:"warn_temp_threshold"`
		CritTempThreshold float64 `json:"crit_temp_threshold"`
		WarnFuelThreshold float64 `json:"warn_fuel_threshold"`
		CritFuelThreshold float64 `json:"crit_fuel_threshold"`
		CooldownSeconds   int     `json:"cooldown_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.DeviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	thresholds := &alert.DeviceThresholds{
		WarnTempThreshold: input.WarnTempThreshold,
		CritTempThreshold: input.CritTempThreshold,
		WarnFuelThreshold: input.WarnFuelThreshold,
		CritFuelThreshold: input.CritFuelThreshold,
		CooldownSeconds:   input.CooldownSeconds,
	}

	if thresholds.WarnTempThreshold == 0 {
		thresholds.WarnTempThreshold = 85
	}
	if thresholds.CritTempThreshold == 0 {
		thresholds.CritTempThreshold = 90
	}
	if thresholds.WarnFuelThreshold == 0 {
		thresholds.WarnFuelThreshold = 20
	}
	if thresholds.CritFuelThreshold == 0 {
		thresholds.CritFuelThreshold = 10
	}
	if thresholds.CooldownSeconds == 0 {
		thresholds.CooldownSeconds = 30
	}

	if err := s.alertMgr.UpsertThresholds(input.DeviceID, thresholds); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"status": "ok", "device_id": input.DeviceID})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func safeFloat64(v float64) float64 {
	if v < 0 || v > 100 {
		return v
	}
	return v
}
