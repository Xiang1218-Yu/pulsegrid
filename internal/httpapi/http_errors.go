package httpapi

import (
	"errors"
	"net/http"

	"pulsegrid/internal/domain"
)

func StatusForError(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrConflict) || errors.Is(err, domain.ErrInvalidState):
		return http.StatusConflict
	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
