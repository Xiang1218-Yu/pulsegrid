package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
)

const bug14SourceMarker = "GET /v1/events"

func waitForBug14Subscribers(t *testing.T, bus *events.Bus, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if bus.SubscriberCount() == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("subscriber count = %d, want %d", bus.SubscriberCount(), want)
}

func TestBug14EventSubscriptionReleasedAfterDisconnect(t *testing.T) {
	t.Log(bug14SourceMarker)
	bus := events.New(events.Config{Buffer: 4})
	server := New(Config{Events: bus, Logger: slog.Default()})
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/v1/events?topic=contact.created", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, request)
		close(done)
	}()
	waitForBug14Subscribers(t, bus, 1)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("event stream did not stop after disconnect")
	}
	waitForBug14Subscribers(t, bus, 0)
}

func TestBug14EventStreamDeliversPublishedEvent(t *testing.T) {
	bus := events.New(events.Config{Buffer: 4})
	server := New(Config{Events: bus, Logger: slog.Default()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/v1/events?topic=contact.created", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, request)
		close(done)
	}()
	waitForBug14Subscribers(t, bus, 1)
	if err := bus.Publish(context.Background(), domain.Event{Type: "contact.created", Subject: "contact-14"}); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) && response.Body.Len() == 0 {
		time.Sleep(time.Millisecond)
	}
	if response.Body.Len() == 0 {
		t.Fatal("event stream did not receive published event")
	}
	cancel()
	<-done
}
