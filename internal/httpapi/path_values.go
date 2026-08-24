package httpapi

import (
	"errors"
	"net/http"
)

func RequiredPath(r *http.Request, name string) (string, error) {
	value := r.PathValue(name)
	if value == "" {
		return "", errors.New("path parameter " + name + " is required")
	}
	return value, nil
}
