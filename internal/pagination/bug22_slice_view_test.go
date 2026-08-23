package pagination_test

import (
	"testing"

	"pulsegrid/internal/batch"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/pagination"
	"pulsegrid/internal/store"
)

// Coverage markers: store.FilterSubscribed, pagination.Apply, batch.Chunk.
func TestBug22SliceViewIsolation(t *testing.T) {
	contacts := []domain.Contact{
		{ID: "contact-1", Email: "one@example.com", Status: domain.ContactSubscribed},
		{ID: "contact-2", Email: "two@example.com", Status: domain.ContactUnsubscribed},
	}
	filtered := store.FilterSubscribed(contacts)
	filtered[0].Email = "changed@example.com"
	if contacts[0].Email == filtered[0].Email {
		t.Errorf("filtered contacts must not mutate the caller's slice")
	}

	items := []string{"first", "second", "third"}
	page := pagination.Apply(items, pagination.Request{Page: 1, Size: 2})
	page.Items[0] = "changed"
	if items[0] == page.Items[0] {
		t.Errorf("page items must not mutate the caller's slice")
	}

	batchItems := []batch.Item{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	chunks := batch.Chunk(batchItems, 2)
	chunks[0][0].ID = "changed"
	if batchItems[0].ID == chunks[0][0].ID {
		t.Errorf("chunks must not expose the caller's backing array")
	}
}

func TestBug22SliceOperationsPreserveValues(t *testing.T) {
	items := []string{"first", "second", "third"}
	page := pagination.Apply(items, pagination.Request{Page: 1, Size: 2})
	if len(page.Items) != 2 || page.Items[0] != "first" || page.Items[1] != "second" {
		t.Fatalf("unexpected page result: %#v", page.Items)
	}
	chunks := batch.Chunk([]batch.Item{{ID: "a"}, {ID: "b"}, {ID: "c"}}, 2)
	if len(chunks) != 2 || chunks[1][0].ID != "c" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}
