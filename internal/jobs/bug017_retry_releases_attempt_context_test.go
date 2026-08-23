package jobs

// Coverage markers: MaxRetry, context.Context, queue.Enqueue.

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestBug017RetryReleasesAttemptContext(t *testing.T) {
	queue := New(Config{Workers: 1, Capacity: 2})
	var mu sync.Mutex
	var calls int
	var first context.Context
	result := make(chan error, 1)
	queue.Register("retry-context", func(ctx context.Context, job Job) error {
		mu.Lock()
		calls++
		call := calls
		if call == 1 {
			first = ctx
			mu.Unlock()
			return errors.New("retry once")
		}
		previous := first
		mu.Unlock()
		if previous == nil || previous.Err() == nil {
			result <- errors.New("previous attempt context was not released")
		} else {
			result <- nil
		}
		return nil
	})
	queue.Start()
	defer queue.Stop()
	if err := queue.Enqueue(context.Background(), Job{Type: "retry-context", MaxRetry: 1}); err != nil {
		t.Fatalf("enqueue retry job: %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("retry handler did not complete")
	}
}

func TestBug017SingleAttemptCompletesNormally(t *testing.T) {
	queue := New(Config{Workers: 1, Capacity: 1})
	finished := make(chan struct{})
	queue.Register("single-attempt", func(ctx context.Context, job Job) error {
		close(finished)
		return nil
	})
	queue.Start()
	defer queue.Stop()
	if err := queue.Enqueue(context.Background(), Job{Type: "single-attempt"}); err != nil {
		t.Fatalf("enqueue single-attempt job: %v", err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("single-attempt handler did not complete")
	}
}
