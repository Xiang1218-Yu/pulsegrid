package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// Queue.Register, Queue.Enqueue, Queue.execute
func TestBug029NilWorkerDoesNotCrash(t *testing.T) {
	queue := New(Config{Workers: 1, Capacity: 2})
	queue.Register("nil-worker", nil)
	queue.Start()
	defer queue.Stop()
	if err := queue.Enqueue(context.Background(), Job{Type: "nil-worker"}); err != nil {
		t.Fatalf("enqueue nil worker job: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if got := queue.Stats().Failed; got != 1 {
		t.Fatalf("failed jobs = %d, want 1", got)
	}
}

func TestBug029RegisteredWorkerRegression(t *testing.T) {
	queue := New(Config{Workers: 1, Capacity: 2})
	var calls atomic.Int32
	queue.Register("healthy-worker", func(context.Context, Job) error {
		calls.Add(1)
		return nil
	})
	queue.Start()
	defer queue.Stop()
	if err := queue.Enqueue(context.Background(), Job{Type: "healthy-worker"}); err != nil {
		t.Fatalf("enqueue healthy worker job: %v", err)
	}
	deadline := time.Now().Add(100 * time.Millisecond)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() != 1 {
		t.Fatalf("worker calls = %d, want 1", calls.Load())
	}
}
