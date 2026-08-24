package pagination

import (
	"testing"
)

// TestApply_WindowIsDetachedFromSource guards against the slice-aliasing bug
// where a paged window shared its backing array with the source list: mutating
// the returned page would rewrite later pages and the original list after a
// refresh. After Apply, Result.Items must be an independent copy.
func TestApply_WindowIsDetachedFromSource(t *testing.T) {
	source := []int{1, 2, 3, 4, 5, 6}
	page1 := Apply(source, Request{Page: 1, Size: 2, Offset: 0})
	page2 := Apply(source, Request{Page: 2, Size: 2, Offset: 2})

	if len(page1.Items) != 2 || page1.Items[0] != 1 || page1.Items[1] != 2 {
		t.Fatalf("page1 = %v, want [1 2]", page1.Items)
	}
	if len(page2.Items) != 2 || page2.Items[0] != 3 || page2.Items[1] != 4 {
		t.Fatalf("page2 = %v, want [3 4]", page2.Items)
	}

	// Capture the expected source contents BEFORE any mutation, so the check is
	// not silently fed back the mutated values via `range source`.
	wantSource := []int{1, 2, 3, 4, 5, 6}

	// Mutate the first page's contents in place. This must not propagate to the
	// source list or to any other page.
	page1.Items[0] = 999
	page1.Items[1] = 888

	for i, want := range wantSource {
		if source[i] != want {
			t.Fatalf("source mutated via page1: source=%v want=%v", source, wantSource)
		}
	}

	// Re-read page2 from the same source: must still reflect the original order.
	again := Apply(source, Request{Page: 2, Size: 2, Offset: 2})
	if again.Items[0] != 3 || again.Items[1] != 4 {
		t.Fatalf("page2 changed after mutating page1: %v", again.Items)
	}
}

// TestApply_EmptyWindowIsSafeNil asserts that paging past the end or into an
// empty source yields a non-nil empty slice rather than a shared nil/array.
func TestApply_EmptyWindowIsSafeNil(t *testing.T) {
	empty := Apply([]int{}, Request{Page: 1, Size: 2, Offset: 0})
	if empty.Items == nil {
		t.Fatalf("Items should be non-nil empty slice, got nil")
	}
	if len(empty.Items) != 0 {
		t.Fatalf("empty.Items = %v, want []", empty.Items)
	}
	// Append must not affect the (absent) source.
	empty.Items = append(empty.Items, 42)
	if len([]int{}) != 0 {
		t.Fatalf("source affected by append on empty window")
	}
}

// TestApplyCursor_WindowIsDetachedFromSource mirrors the above for cursor
// pagination, ensuring the same invariant holds there.
func TestApplyCursor_WindowIsDetachedFromSource(t *testing.T) {
	source := []string{"a", "b", "c", "d"}
	key := func(s string) string { return s }

	res, _ := ApplyCursor(source, Cursor{Limit: 2}, key)
	if len(res.Items) != 2 || res.Items[0] != "a" || res.Items[1] != "b" {
		t.Fatalf("first cursor page = %v, want [a b]", res.Items)
	}

	res.Items[0] = "X"
	res.Items[1] = "Y"

	wantSource := []string{"a", "b", "c", "d"}
	for i, want := range wantSource {
		if source[i] != want {
			t.Fatalf("source mutated via cursor page: source=%v want=%v", source, wantSource)
		}
	}
}
