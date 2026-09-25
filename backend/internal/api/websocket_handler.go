package api

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	ws "github.com/intecs/iot-monitoring/backend/internal/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type MQTTSStatusResponse struct {
	Connected bool   `json:"connected"`
	LastMsgAt string `json:"last_msg_at"`
}

func handleWebSocket(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := hub.AddClient(conn)

		connected, lastMsg := hub.GetMQTTStatus()
		statusPayload := map[string]interface{}{
			"connected":   connected,
			"last_msg_at": lastMsg,
		}
		client.SendMessage(ws.MessageTypeMQTTStatus, statusPayload)

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}

		hub.RemoveClient(client)
	}
}

func handleMQTTStatus(hub *ws.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		connected, lastMsg := hub.GetMQTTStatus()
		writeJSON(w, MQTTSStatusResponse{
			Connected: connected,
			LastMsgAt: lastMsg,
		})
	}
}
