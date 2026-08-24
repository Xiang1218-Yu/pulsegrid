package service

import (
	"context"
	"errors"
	"testing"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

// When a delivery record no longer exists, the lookup must surface the
// not-found sentinel unwrapped so downstream mapping (httpapi.StatusForError)
// classifies it as 404 rather than a 500 server fault. This guards the original
// bug where GetDelivery wrapped the store error and the sentinel was lost.
func TestGetDeliveryMissingMapsToNotFound(t *testing.T) {
	app := New(Config{Repository: store.NewMemory()})

	_, err := app.GetDelivery(context.Background(), "missing-id")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	// The error must be the unwrapped sentinel so callers see the same
	// "resource not found" message as every other Get endpoint and so the
	// status mapping resolves to 404.
	if err.Error() != domain.ErrNotFound.Error() {
		t.Fatalf("error message = %q, want %q", err.Error(), domain.ErrNotFound.Error())
	}
}
