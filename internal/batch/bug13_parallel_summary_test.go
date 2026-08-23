package batch

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const bug13SourceMarkers = "Processor.Run report.Summary"

func TestBug13ParallelSummaryMatchesResults(t *testing.T) {
	t.Log(bug13SourceMarkers)
	const workers = 8
	items := make([]Item, 256)
	for index := range items {
		items[index] = Item{ID: "item-" + string(rune('a'+index%26)), Payload: map[string]any{"index": index}}
	}
	var entered sync.WaitGroup
	entered.Add(workers)
	gate := make(chan struct{})
	var started atomic.Int32
	processor := New(Config{
		Workers:  workers,
		Continue: true,
		Handler: func(context.Context, Item) error {
			if started.Add(1) <= workers {
				entered.Done()
				<-gate
			}
			return nil
		},
	})
	done := make(chan Report, 1)
	go func() {
		done <- processor.Run(context.Background(), items)
	}()
	entered.Wait()
	close(gate)
	select {
	case report := <-done:
		if report.Summary.Succeeded != len(items) {
			t.Fatalf("parallel succeeded = %d, want %d", report.Summary.Succeeded, len(items))
		}
	case <-time.After(time.Second):
		t.Fatal("parallel batch did not finish")
	}
}

func TestBug13SingleWorkerSummaryRemainsAccurate(t *testing.T) {
	processor := New(Config{
		Workers:  1,
		Continue: true,
		Handler:  func(context.Context, Item) error { return nil },
	})
	report := processor.Run(context.Background(), []Item{
		{ID: "one", Payload: map[string]any{}},
		{ID: "two", Payload: map[string]any{}},
	})
	if report.Summary.Succeeded != 2 || report.Summary.Failed != 0 {
		t.Fatalf("single-worker summary = %+v, want two successes", report.Summary)
	}
}
