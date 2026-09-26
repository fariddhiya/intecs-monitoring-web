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
	Name           string  `json:"name"`
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

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int         `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}

func paginate(totalItems, page, pageSize int) PaginatedResponse {
	totalPages := (totalItems + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	return PaginatedResponse{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}
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
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	
	page := 1
	pageSize := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	
	query := `
		SELECT d.device_id, t.fuel_percentage, t.temperature, t.flow_rate, 
		       t.equipment_status, d.last_seen, d.name, s.name as site_name
		FROM devices d
		LEFT JOIN sites s ON d.site_id = s.id
		LEFT JOIN LATERAL (
			SELECT fuel_percentage, temperature, flow_rate, equipment_status
			FROM telemetry WHERE device_id = d.device_id ORDER BY timestamp DESC LIMIT 1
		) t ON TRUE
	`
	
	args := []interface{}{}
	argCount := 1
	
	var conditions []string
	hasFilter := false
	
	search := r.URL.Query().Get("search")
	if search != "" {
		conditions = append(conditions, fmt.Sprintf("d.device_id ILIKE $%d", argCount))
		args = append(args, "%"+search+"%")
		argCount++
		hasFilter = true
	}
	
	connStatus := r.URL.Query().Get("connection")
	equipStatus := r.URL.Query().Get("equipment_status")
	if equipStatus != "" {
		conditions = append(conditions, fmt.Sprintf("t.equipment_status = $%d", argCount))
		args = append(args, equipStatus)
		argCount++
		hasFilter = true
	}
	
	if connStatus != "" {
		hasFilter = true
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	sortCol := r.URL.Query().Get("sort")
	sortDir := r.URL.Query().Get("dir")
	if sortCol == "" {
		sortCol = "device_id"
	}
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}
	
	sortMap := map[string]string{
		"device_id":     "d.device_id",
		"site":          "s.name",
		"fuel_percent":  "t.fuel_percentage",
		"temperature":   "t.temperature",
		"flow_rate":     "t.flow_rate",
		"connection":    "d.last_seen",
		"last_seen":     "d.last_seen",
		"equipment":     "t.equipment_status",
	}
	sortedCol := sortMap[sortCol]
	if sortedCol == "" {
		sortedCol = "d.device_id"
	}
	
	query += fmt.Sprintf(" ORDER BY %s %s", sortedCol, sortDir)
	
	offset := (page - 1) * pageSize
	paginatedQuery := query + fmt.Sprintf(" OFFSET %d LIMIT %d", offset, pageSize)
	
	rows, err := s.db.Query(paginatedQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	devices := []DeviceView{}
	for rows.Next() {
		var d DeviceView
		var lastSeen time.Time
		var fuelPct, temp, flow sql.NullFloat64
		var equipStatus, deviceName string
		var siteName sql.NullString

		err := rows.Scan(&d.DeviceID, &fuelPct, &temp, &flow, &equipStatus, &lastSeen, &deviceName, &siteName)
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
		d.Name = deviceName
		if siteName.Valid {
			d.Site = siteName.String
		} else {
			d.Site = "Unknown"
		}
		
		if connStatus != "" && d.Connection != connStatus {
			continue
		}

		devices = append(devices, d)
	}

	totalItems := len(devices)
	totalPages := 1
	if !hasFilter {
		var totalCount int
		err := s.db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&totalCount)
		if err == nil {
			totalItems = totalCount
			totalPages = (totalCount + pageSize - 1) / pageSize
			if totalPages < 1 {
				totalPages = 1
			}
		}
	} else {
		totalPages = (totalItems + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}
	}

	writeJSON(w, PaginatedResponse{
		Data:       devices,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalItems,
		TotalPages: totalPages,
	})
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

	var timeRange time.Duration
	switch rangeStr {
	case "1h":
		timeRange = time.Hour
	case "6h":
		timeRange = 6 * time.Hour
	case "24h":
		timeRange = 24 * time.Hour
	case "7d":
		timeRange = 7 * 24 * time.Hour
	default:
		timeRange = 24 * time.Hour
	}

	limit := 500
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
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

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	
	page := 1
	pageSize := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	
	status := r.URL.Query().Get("status")
	countQuery := "SELECT COUNT(*) FROM alerts"
	query := "SELECT id, device_id, type, severity, message, status, created_at FROM alerts"
	args := []interface{}{}
	argCount := 1

	if status != "" {
		countQuery += fmt.Sprintf(" WHERE status = $%d", argCount)
		query += fmt.Sprintf(" WHERE status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC OFFSET %d LIMIT %d", offset, pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	alerts := []AlertView{}
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

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	writeJSON(w, PaginatedResponse{
		Data:       alerts,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalCount,
		TotalPages: totalPages,
	})
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
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")
	
	page := 1
	pageSize := 20
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	deviceID := r.URL.Query().Get("device_id")
	status := r.URL.Query().Get("status")

	countQuery := `SELECT COUNT(*) FROM alerts`
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
		countQuery += " WHERE " + strings.Join(conditions, " AND ")
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var totalCount int
	err := s.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	offset := (page - 1) * pageSize
	query += fmt.Sprintf(" ORDER BY created_at DESC OFFSET %d LIMIT %d", offset, pageSize)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type AlertHistoryView struct {
		ID         string     `json:"id"`
		DeviceID   string     `json:"device_id"`
		Type       string     `json:"type"`
		Severity   string     `json:"severity"`
		Message    string     `json:"message"`
		Status     string     `json:"status"`
		CreatedAt  string     `json:"created_at"`
		ResolvedAt *string    `json:"resolved_at,omitempty"`
	}

	alerts := []AlertHistoryView{}
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

	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	writeJSON(w, PaginatedResponse{
		Data:       alerts,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: totalCount,
		TotalPages: totalPages,
	})
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
