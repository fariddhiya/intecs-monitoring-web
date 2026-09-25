package mqtt

type Telemetry struct {
	DeviceID      string  `json:"device_id"`
	Timestamp     string  `json:"timestamp"`
	FuelPercent   float64 `json:"fuel_percentage"`
	FuelLevel     float64 `json:"fuel_level"`
	Temperature   float64 `json:"temperature"`
	FlowRate      float64 `json:"flow_rate"`
	EquipStatus   string  `json:"equipment_status"`
}
