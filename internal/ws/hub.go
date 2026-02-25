package ws

import "sync"

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{subscribers: map[string]map[*Client]struct{}{}}
}

func (h *Hub) Subscribe(channel string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[channel]; !ok {
		h.subscribers[channel] = map[*Client]struct{}{}
	}
	h.subscribers[channel][client] = struct{}{}
	client.addChannel(channel)
}

func (h *Hub) UnsubscribeAll(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, channel := range client.listChannels() {
		if subs, ok := h.subscribers[channel]; ok {
			delete(subs, client)
			if len(subs) == 0 {
				delete(h.subscribers, channel)
			}
		}
	}
}

func (h *Hub) Publish(channel string, payload []byte) {
	h.mu.RLock()
	subs := h.subscribers[channel]
	h.mu.RUnlock()

	for c := range subs {
		c.send(payload)
	}
}
