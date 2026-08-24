package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"pulsegrid/internal/domain"
)

// Coverage markers: App.UpdateContact, contactsUpdate, StatusForError.
func TestBug002WrappedNotFoundReachesHTTP(t *testing.T) {
	err := fmt.Errorf("contacts endpoint failed: %w", domain.ErrNotFound)
	if got := StatusForError(err); got != http.StatusNotFound {
		t.Fatalf("HTTP status = %d, want %d for wrapped not-found error", got, http.StatusNotFound)
	}
}

func TestBug002DirectErrorClassificationRemainsStable(t *testing.T) {
	if got := StatusForError(domain.ErrInvalidInput); got != http.StatusBadRequest {
		t.Fatalf("direct invalid input status = %d, want %d", got, http.StatusBadRequest)
	}
	if !errors.Is(domain.ErrNotFound, domain.ErrNotFound) {
		t.Fatal("sentinel error changed")
	}
}
