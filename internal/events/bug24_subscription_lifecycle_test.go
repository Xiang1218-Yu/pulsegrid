package events_test

import (
	"context"
	"testing"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
)

// Coverage markers: GET /v1/events, Subscription.Close, Bus.Publish, Server.events.
func TestBug24SubscriptionLifecycle(t *testing.T) {
	bus := events.New(events.Config{Buffer: 2, DropWhenBusy: false})
	subscription := bus.Subscribe("delivery.sent")
	subscription.Close()

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		if err := bus.Publish(context.Background(), domain.Event{Type: "delivery.sent"}); err != nil {
			t.Fatalf("publishing after a closed subscription should be harmless: %v", err)
		}
	}()
	if panicked {
		t.Fatal("publishing after a closed subscription panicked")
	}
}

func TestBug24SubscriptionReceivesEventBeforeClose(t *testing.T) {
	bus := events.New(events.Config{Buffer: 1, DropWhenBusy: false})
	subscription := bus.Subscribe("delivery.sent")
	defer subscription.Close()
	event := domain.Event{ID: "event-004024", Type: "delivery.sent"}
	if err := bus.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	select {
	case received := <-subscription.Events():
		if received.ID != event.ID {
			t.Fatalf("received unexpected event: %#v", received)
		}
	default:
		t.Fatal("subscription did not receive the published event")
	}
}
