package events

import (
	"context"
	"fmt"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

// Source markers: GET /v1/events Subscription.Close Bus.Publish Server.events
func TestBug009SubscriptionLifecycleClose(t *testing.T) {
	bus := New(Config{Buffer: 1, DropWhenBusy: false})
	subscription := bus.Subscribe("delivery.sent")
	defer bus.Close()

	if err := bus.Publish(context.Background(), domain.Event{Type: "delivery.sent", Subject: "first"}); err != nil {
		t.Fatal(err)
	}

	result := make(chan error, 1)
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				result <- fmt.Errorf("publisher panicked while subscription closed: %v", recovered)
			}
		}()
		result <- bus.Publish(context.Background(), domain.Event{Type: "delivery.sent", Subject: "second"})
	}()

	time.Sleep(time.Millisecond)
	subscription.Close()

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("publisher remained blocked after subscription close")
	}
}

func TestBug009SubscriptionLifecycleDelivery(t *testing.T) {
	bus := New(Config{Buffer: 2})
	subscription := bus.Subscribe("delivery.sent")
	subscription.Close()
	defer bus.Close()

	if err := bus.Publish(context.Background(), domain.Event{Type: "delivery.sent"}); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-subscription.Events():
		if ok {
			t.Fatal("closed subscription received a new event")
		}
	case <-time.After(time.Second):
		t.Fatal("closed subscription did not close its event stream")
	}
}
