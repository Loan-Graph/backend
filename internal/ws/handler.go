package ws

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

type subscribeMessage struct {
	Action   string `json:"action"`
	Channel  string `json:"channel"`
	PoolID   string `json:"poolId"`
	LenderID string `json:"lenderId"`
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	websocket.Handler(func(conn *websocket.Conn) {
		client := NewClient(conn)
		go h.writer(client)
		h.reader(client)
	}).ServeHTTP(c.Writer, c.Request)
}

func (h *Handler) reader(client *Client) {
	defer func() {
		h.hub.UnsubscribeAll(client)
		close(client.out)
		_ = client.conn.Close()
	}()

	for {
		var raw string
		if err := websocket.Message.Receive(client.conn, &raw); err != nil {
			return
		}
		var msg subscribeMessage
		if err := json.Unmarshal([]byte(raw), &msg); err != nil {
			continue
		}
		if strings.ToLower(strings.TrimSpace(msg.Action)) != "subscribe" {
			continue
		}
		topic := subscriptionTopic(msg)
		if topic == "" {
			continue
		}
		h.hub.Subscribe(topic, client)
	}
}

func (h *Handler) writer(client *Client) {
	for payload := range client.out {
		if err := websocket.Message.Send(client.conn, string(payload)); err != nil {
			return
		}
	}
}

func subscriptionTopic(msg subscribeMessage) string {
	channel := strings.ToLower(strings.TrimSpace(msg.Channel))
	switch channel {
	case "pool:repayments":
		poolID := strings.TrimSpace(msg.PoolID)
		if poolID == "" {
			return ""
		}
		return "pool:repayments:" + poolID
	case "lender:portfolio":
		lenderID := strings.TrimSpace(msg.LenderID)
		if lenderID == "" {
			return ""
		}
		return "lender:portfolio:" + lenderID
	default:
		return ""
	}
}
