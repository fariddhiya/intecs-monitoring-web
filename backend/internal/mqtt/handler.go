package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Message = mqtt.Message

func ParseMessage(payload []byte) (*Telemetry, error) {
	var t Telemetry
	if err := json.Unmarshal(payload, &t); err != nil {
		return nil, fmt.Errorf("malformed JSON: %w", err)
	}
	if t.DeviceID == "" {
		return nil, fmt.Errorf("missing device_id")
	}
	return &t, nil
}

func ExtractDeviceInfo(topic string) (siteID, deviceID string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("invalid topic format: %s", topic)
	}
	return parts[2], parts[4], nil
}
