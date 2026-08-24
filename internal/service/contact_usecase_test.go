package service

import (
	"reflect"
	"testing"
	"time"

	"pulsegrid/internal/domain"
)

func TestSortContactsByEngagementDoesNotMutateInput(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	// Input ordered least- to most-recent. The engagement sort will reverse it.
	input := []domain.Contact{
		{ID: "a", Email: "a@example.com", LastEngagedAt: &t0, Tags: []string{"vip"}},
		{ID: "b", Email: "b@example.com", LastEngagedAt: &t1, Tags: []string{}},
		{ID: "c", Email: "c@example.com", LastEngagedAt: &t2, Tags: []string{}},
	}
	snapshot := make([]domain.Contact, len(input))
	for i := range input {
		snapshot[i] = input[i]
	}

	_ = SortContactsByEngagement(input)

	// The caller's slice and backing arrays must be unchanged in identity and order.
	if !reflect.DeepEqual(input, snapshot) {
		t.Fatalf("SortContactsByEngagement mutated its input:\n got=%v\nwant=%v", input, snapshot)
	}
}

func TestSortContactsByEngagementDoesNotMutateStoreViaInput(t *testing.T) {
	// Mirrors the reported symptom: a stored contact whose Tags slice is shared
	// with the input must not have its tags reordered/contaminated when the
	// result slice is later mutated by the caller.
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)

	storeContact := domain.Contact{
		ID: "a", Email: "a@example.com", LastEngagedAt: &t0, Tags: []string{"vip", "early"},
	}
	input := []domain.Contact{storeContact.Clone(), {ID: "c", Email: "c@example.com", LastEngagedAt: &t2, Tags: []string{}}}

	result := SortContactsByEngagement(input)

	// Engagement sort puts the most-recent contact first; locate the one that
	// carries tags rather than assuming a fixed index.
	var taggedIndex = -1
	for i := range result {
		if result[i].ID == "a" {
			taggedIndex = i
			break
		}
	}
	if taggedIndex == -1 {
		t.Fatalf("tagged contact missing from result")
	}

	// Caller mutates the result freely.
	result[taggedIndex].Tags[0] = "mutated"

	// The store-side clone (which shared the slice before the fix) must be intact.
	if got, want := storeContact.Tags, []string{"vip", "early"}; !equalStringSlices(got, want) {
		t.Fatalf("store tags leaked through result mutation: got %v want %v", got, want)
	}
	if got := input[0].Tags; !equalStringSlices(got, []string{"vip", "early"}) {
		t.Fatalf("input tags leaked through result mutation: got %v", got)
	}
}

func equalStringSlices(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
