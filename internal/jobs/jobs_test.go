package jobs

import (
	"context"
	"errors"
	"testing"
)

func TestEnqueueNotStartedIsUnavailable(t *testing.T) {
	q := New(Config{Workers: 1, Capacity: 4})
	err := q.Enqueue(context.Background(), Job{Type: "deliver-message"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("enqueue before start: want %v, got %v", ErrUnavailable, err)
	}
	if !errors.Is(err, ErrNotStarted) {
		t.Fatalf("enqueue before start: want wrapped %v, got %v", ErrNotStarted, err)
	}
	if !Retryable(err) {
		t.Fatalf("enqueue before start: want retryable, got %v", err)
	}
}

func TestEnqueueAfterStopIsUnavailable(t *testing.T) {
	q := New(Config{Workers: 1, Capacity: 4})
	q.Start()
	q.Stop()
	err := q.Enqueue(context.Background(), Job{Type: "deliver-message"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("enqueue after stop: want %v, got %v", ErrUnavailable, err)
	}
	if !Retryable(err) {
		t.Fatalf("enqueue after stop: want retryable, got %v", err)
	}
}

func TestEnqueueSaturatedIsUnavailable(t *testing.T) {
	q := New(Config{Workers: 0, Capacity: 1})
	q.Start()
	defer q.Stop()
	// Fill the single-capacity buffer without a worker draining it.
	if err := q.Enqueue(context.Background(), Job{Type: "deliver-message"}); err != nil {
		t.Fatalf("first enqueue: %v", err)
	}
	err := q.Enqueue(context.Background(), Job{Type: "deliver-message"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("enqueue when full: want %v, got %v", ErrUnavailable, err)
	}
	if !Retryable(err) {
		t.Fatalf("enqueue when full: want retryable, got %v", err)
	}
}

func TestEnqueueStartedSucceeds(t *testing.T) {
	q := New(Config{Workers: 1, Capacity: 4})
	q.Register("deliver-message", func(context.Context, Job) error { return nil })
	q.Start()
	defer q.Stop()
	if err := q.Enqueue(context.Background(), Job{Type: "deliver-message", Payload: map[string]any{"delivery_id": "d-1"}, MaxRetry: 1}); err != nil {
		t.Fatalf("enqueue after start: %v", err)
	}
}

// Retryable only matches the unavailable family so genuine non-retryable errors
// are not mistaken for transient outages.
func TestRetryableRejectsForeignErrors(t *testing.T) {
	if Retryable(errors.New("totally unrelated")) {
		t.Fatalf("Retryable returned true for an unrelated error")
	}
}
