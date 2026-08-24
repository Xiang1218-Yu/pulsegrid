package batch

import (
	"context"
	"testing"
)

// Coverage markers: Processor.Run, recordResult, NormalizeWorkers.
func TestBug25ParallelReportIntegrity(t *testing.T) {
	const workers = 8
	const itemCount = 64
	ready := make(chan struct{}, workers)
	start := make(chan struct{})
	handler := func(context.Context, Item) error {
		select {
		case ready <- struct{}{}:
		default:
		}
		<-start
		return nil
	}
	processor := New(Config{Workers: workers, Continue: true, Handler: handler})
	items := make([]Item, itemCount)
	for index := range items {
		items[index] = Item{ID: "item-" + string(rune('a'+index%26))}
	}
	var report Report
	done := make(chan struct{})
	go func() {
		report = processor.Run(context.Background(), items)
		close(done)
	}()
	for index := 0; index < workers; index++ {
		<-ready
	}
	close(start)
	<-done
	if len(report.Results) != itemCount {
		t.Fatalf("parallel report should contain %d results, got %d", itemCount, len(report.Results))
	}
	if report.Summary.Succeeded != itemCount {
		t.Fatalf("parallel report should count %d successes, got %d", itemCount, report.Summary.Succeeded)
	}
}

func TestBug25WorkerLimitRemainsBounded(t *testing.T) {
	if got := NormalizeWorkers(1000); got != MaxBatchWorkers {
		t.Fatalf("worker limit should be %d, got %d", MaxBatchWorkers, got)
	}
}
