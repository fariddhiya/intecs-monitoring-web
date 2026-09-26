package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSMessageType string

const (
	MessageTypeTelemetry   WSMessageType = "telemetry"
	MessageTypeAlert       WSMessageType = "alert"
	MessageTypeMQTTStatus  WSMessageType = "mqtt_status"
	MessageTypeHeartbeat   WSMessageType = "heartbeat"
	MessageTypeError       WSMessageType = "error"
)

type WSMessage struct {
	Type      WSMessageType `json:"type"`
	Payload   interface{}   `json:"payload,omitempty"`
	Timestamp string        `json:"timestamp,omitempty"`
}

type WSClient struct {
	conn   *websocket.Conn
	send   chan []byte
	mu     sync.Mutex
}

func (c *WSClient) readPump(hub *Hub) {
	defer func() {
		hub.RemoveClient(c)
		c.close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	c.conn.SetCloseHandler(func(code int, text string) error {
		switch code {
		case websocket.CloseGoingAway, websocket.CloseNormalClosure:
			return nil
		default:
			log.Printf("WebSocket abnormal close: code=%d", code)
		}
		return nil
	})

	pingResetTimer := time.NewTimer(5 * time.Second)
	defer pingResetTimer.Stop()

	for {
		pingResetTimer.Reset(5 * time.Second)
		_, msg, err := c.conn.ReadMessage()
		
		select {
		case <-pingResetTimer.C:
		default:
			c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		}

		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}
		_ = msg
	}
}

func (c *WSClient) writePump(hub *Hub) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case msg := <-c.send:
			c.mu.Lock()
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		case <-ticker.C:
			c.mu.Lock()
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (c *WSClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.Close()
}

func (c *WSClient) SendMessage(msgType WSMessageType, payload interface{}) {
	msg := WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}
	c.send <- data
}

type Hub struct {
	clients     map[*WSClient]bool
	broadcast   chan []byte
	mqttConnected bool
	mqttLastMsg string
	mqttMu      sync.RWMutex
	upgrader    websocket.Upgrader
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*WSClient]bool),
		broadcast: make(chan []byte, 256),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Hub) Run() {
	for {
		select {
		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				case <-time.After(100 * time.Millisecond):
					log.Printf("Slow client dropped, send channel full")
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) AddClient(conn *websocket.Conn) *WSClient {
	client := &WSClient{
		conn: conn,
		send: make(chan []byte, 256),
	}
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()

	go client.writePump(h)
	go client.readPump(h)

	h.updateMQTTStatus()

	return client
}

func (h *Hub) RemoveClient(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		close(client.send)
		delete(h.clients, client)
	}
}

func (h *Hub) Broadcast(msgType WSMessageType, payload interface{}) {
	msg := WSMessage{
		Type:      msgType,
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error broadcasting WebSocket message: %v", err)
		return
	}
	select {
	case h.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

func (h *Hub) SetMQTTStatus(connected bool, lastMsg string) {
	h.mqttMu.Lock()
	h.mqttConnected = connected
	h.mqttLastMsg = lastMsg
	h.mqttMu.Unlock()

	statusPayload := map[string]interface{}{
		"connected":   connected,
		"last_msg_at": lastMsg,
	}

	var status WSMessageType
	if connected {
		status = MessageTypeMQTTStatus
	} else {
		status = MessageTypeMQTTStatus
	}

	h.Broadcast(status, statusPayload)
}

func (h *Hub) GetMQTTStatus() (bool, string) {
	h.mqttMu.RLock()
	defer h.mqttMu.RUnlock()
	return h.mqttConnected, h.mqttLastMsg
}

func (h *Hub) updateMQTTStatus() {
	h.GetMQTTStatus()
}
