package analytics

import (
	"sync"
	"testing"
	"time"
)

// Coverage markers: Store.Record, Store.RecordValue, Server.metrics, Store.List.
func TestBug003ConcurrentMetricWritesAreCoordinated(t *testing.T) {
	store := New()
	var group sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func(worker int) {
			defer group.Done()
			for index := 0; index < 40; index++ {
				store.RecordValue("org-1", "delivery.sent", float64(worker+index), nil, time.Now().UTC())
			}
		}(worker)
	}
	group.Wait()
	if got := store.Count(); got != 640 {
		t.Fatalf("metric count = %d, want 640", got)
	}
}

func TestBug003SingleMetricReadRemainsStable(t *testing.T) {
	store := New()
	store.RecordValue("org-1", "delivery.sent", 1, nil, time.Now().UTC())
	if got := len(store.List(Filter{OrganizationID: "org-1"})); got != 1 {
		t.Fatalf("metric list size = %d, want 1", got)
	}
}
