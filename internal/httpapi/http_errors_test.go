package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"pulsegrid/internal/domain"
)

func TestStatusForError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not found direct", domain.ErrNotFound, http.StatusNotFound},
		{"not found wrapped", fmt.Errorf("get delivery x: %w", domain.ErrNotFound), http.StatusNotFound},
		{"already exists direct", domain.ErrAlreadyExists, http.StatusConflict},
		{"already exists wrapped", fmt.Errorf("dup: %w", domain.ErrAlreadyExists), http.StatusConflict},
		{"conflict wrapped", fmt.Errorf("c: %w", domain.ErrConflict), http.StatusConflict},
		{"invalid state wrapped", fmt.Errorf("s: %w", domain.ErrInvalidState), http.StatusConflict},
		{"invalid input direct", domain.ErrInvalidInput, http.StatusBadRequest},
		{"invalid input wrapped", fmt.Errorf("organization name: %w", domain.ErrInvalidInput), http.StatusBadRequest},
		{"unknown error", errors.New("boom"), http.StatusInternalServerError},
		{"nil-like unknown", errors.New("context deadline exceeded"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := StatusForError(c.err); got != c.want {
				t.Fatalf("StatusForError(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}
