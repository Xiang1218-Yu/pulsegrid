package pagination

import (
	"net/url"
	"testing"
)

// Source markers: pagination.Apply pagination.ApplyCursor cloneWindow NormalizePageRequest
func TestBug008PageWindowIsolation(t *testing.T) {
	source := []string{"first", "second", "third"}
	page := Apply(source, Request{Page: 1, Size: 2})
	page.Items[0] = "changed"
	if source[0] != "first" {
		t.Fatalf("page result shares source slice storage: source=%v", source)
	}

	cursorPage, _ := ApplyCursor(source, Cursor{Limit: 2}, func(value string) string { return value })
	cursorPage.Items[0] = "cursor changed"
	if source[0] != "first" {
		t.Fatalf("cursor result shares source slice storage: source=%v", source)
	}
}

func TestBug008PageWindowBounds(t *testing.T) {
	result := Apply([]int{1, 2, 3}, Parse(url.Values{"page": {"2"}, "size": {"2"}}))
	if result.Page != 2 || result.Size != 2 || result.Total != 3 || result.Pages != 2 {
		t.Fatalf("unexpected page metadata: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0] != 3 || result.HasNext || !result.HasPrevious {
		t.Fatalf("unexpected page window: %+v", result)
	}
}
