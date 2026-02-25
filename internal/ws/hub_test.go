package ws

import (
	"testing"
	"time"
)

func TestHubSubscribeAndPublish(t *testing.T) {
	hub := NewHub()
	client := NewClient(nil)

	hub.Subscribe("pool:repayments:pool-1", client)
	hub.Publish("pool:repayments:pool-1", []byte(`{"event":"repayment_recorded"}`))

	select {
	case msg := <-client.out:
		if string(msg) != `{"event":"repayment_recorded"}` {
			t.Fatalf("unexpected payload: %s", string(msg))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for message")
	}

	hub.UnsubscribeAll(client)
}
