package ws

import (
	"sync"

	"golang.org/x/net/websocket"
)

type Client struct {
	conn *websocket.Conn
	out  chan []byte

	mu       sync.RWMutex
	channels map[string]struct{}
}

func NewClient(conn *websocket.Conn) *Client {
	return &Client{
		conn:     conn,
		out:      make(chan []byte, 64),
		channels: map[string]struct{}{},
	}
}

func (c *Client) send(payload []byte) {
	select {
	case c.out <- payload:
	default:
		_ = c.conn.Close()
	}
}

func (c *Client) addChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channel] = struct{}{}
}

func (c *Client) listChannels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		out = append(out, ch)
	}
	return out
}
