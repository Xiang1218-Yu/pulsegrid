package analytics

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

// TestRecordValueStoresSingleMetric verifies that a single recorded metric is
// persisted and readable, matching the service's normal record/read path.
func TestRecordValueStoresSingleMetric(t *testing.T) {
	store := New()

	at := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	metric := store.RecordValue("org-1", "delivery.sent", 1, map[string]string{"campaign_id": "camp-1"}, at)

	if metric.ID == "" {
		t.Fatal("expected RecordValue to return a metric with an id")
	}
	if got := store.Count(); got != 1 {
		t.Fatalf("expected count 1, got %d", got)
	}

	rows := store.List(Filter{OrganizationID: "org-1", Name: "delivery.sent"})
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored metric, got %d", len(rows))
	}
	if rows[0].Value != 1 {
		t.Fatalf("expected stored value 1, got %v", rows[0].Value)
	}
	if rows[0].RecordedAt != at {
		t.Fatalf("expected recorded at %v, got %v", at, rows[0].RecordedAt)
	}
	// Mutating the returned metric must not affect stored state.
	rows[0].Dimensions["campaign_id"] = "tampered"
	again := store.List(Filter{OrganizationID: "org-1"})
	if again[0].Dimensions["campaign_id"] != "camp-1" {
		t.Fatalf("stored metric was mutated by caller: got %q", again[0].Dimensions["campaign_id"])
	}
}

// TestRecordFillsRecordedAt verifies Record stamps a zero RecordedAt rather
// than leaving it unset, which would corrupt Daily grouping.
func TestRecordFillsRecordedAt(t *testing.T) {
	store := New()
	store.Record(domain.Metric{ID: "m1", OrganizationID: "org-1", Name: "delivery.sent", Value: 1})

	rows := store.List(Filter{OrganizationID: "org-1"})
	if len(rows) != 1 {
		t.Fatalf("expected 1 stored metric, got %d", len(rows))
	}
	if rows[0].RecordedAt.IsZero() {
		t.Fatal("expected Record to populate RecordedAt")
	}
}

// TestRecordConcurrentPreservesCount reproduces the peak-hour defect: many
// tasks recording delivery metrics at once must each land in the store, and
// the summary count must equal the number of completed records. Run with
// -race to confirm no concurrent-map/slice access remains.
func TestRecordConcurrentPreservesCount(t *testing.T) {
	store := New()

	const workers = 64
	const perWorker = 128
	want := workers * perWorker

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				store.RecordValue("org-1", "delivery.sent", 1, nil, time.Time{})
			}
		}()
	}
	wg.Wait()

	if got := store.Count(); got != want {
		t.Fatalf("expected count %d, got %d (metrics were lost)", want, got)
	}

	summary := store.Summarize(Filter{OrganizationID: "org-1", Name: "delivery.sent"})
	if len(summary) != 1 {
		t.Fatalf("expected 1 summary row, got %d", len(summary))
	}
	if summary[0].Count != want {
		t.Fatalf("expected summary count %d, got %d", want, summary[0].Count)
	}

	daily := store.Daily(Filter{OrganizationID: "org-1", Name: "delivery.sent"})
	if len(daily) != 1 {
		t.Fatalf("expected 1 daily row, got %d", len(daily))
	}
	if daily[0].Count != want {
		t.Fatalf("expected daily count %d, got %d", want, daily[0].Count)
	}
}

// TestRecordConcurrentWithReaders ensures readers observe a consistent slice
// while writers append, reproducing the unstable concurrent read symptom.
func TestRecordConcurrentWithReaders(t *testing.T) {
	store := New()

	const writers = 16
	const readers = 8
	const perWriter = 256
	const readsPerReader = 1000
	want := writers * perWriter

	var wg sync.WaitGroup
	wg.Add(writers)
	for w := 0; w < writers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				store.RecordValue("org-1", "delivery.delivered", 1, nil, time.Time{})
			}
		}()
	}
	wg.Add(readers)
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < readsPerReader; i++ {
				_ = store.List(Filter{OrganizationID: "org-1"})
				_ = store.Count()
				runtime.Gosched()
			}
		}()
	}
	wg.Wait()

	if got := store.Count(); got != want {
		t.Fatalf("expected count %d, got %d (metrics were lost)", want, got)
	}
}
