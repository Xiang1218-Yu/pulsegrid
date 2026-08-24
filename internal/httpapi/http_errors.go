package httpapi

import (
	"net/http"

	"pulsegrid/internal/domain"
)

func StatusForError(err error) int {
	switch {
	case err == domain.ErrNotFound:
		return http.StatusNotFound
	case err == domain.ErrAlreadyExists || err == domain.ErrConflict || err == domain.ErrInvalidState:
		return http.StatusConflict
	case err == domain.ErrInvalidInput:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
