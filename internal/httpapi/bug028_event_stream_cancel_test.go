package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsegrid/internal/events"
)

// GET /v1/events, request context, events handler, Bus.Publish
func TestBug028EventStreamCancelStopsAfterCancel(t *testing.T) {
	bus := events.New(events.Config{Buffer: 1, DropWhenBusy: false})
	server := New(Config{Events: bus})
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/v1/events?topic=contact.created", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		server.Handler().ServeHTTP(recorder, request)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event stream did not stop after request cancellation")
	}
}

func TestBug028EventStreamCancelPublishRegression(t *testing.T) {
	bus := events.New(events.Config{Buffer: 1, DropWhenBusy: false})
	subscription := bus.Subscribe("contact.created")
	defer subscription.Close()
	if err := bus.Publish(context.Background(), events.NewEvent("org-1", "contact.created", "contact-1", "contact-1", nil)); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	select {
	case <-subscription.Events():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("event was not delivered")
	}
}
