package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// Coverage markers: GET /v1/deliveries/{id}, App.GetDelivery, StatusForError, writeError, GET /healthz.
func TestBug23WrappedErrorStatus(t *testing.T) {
	app := service.New(service.Config{Repository: store.NewMemory(), Logger: slog.Default()})
	server := New(Config{App: app, Logger: slog.Default()}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/v1/deliveries/missing-delivery", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing delivery should be a 404, got %d with body %s", response.Code, response.Body.String())
	}
}

func TestBug23HealthEndpointStillWorks(t *testing.T) {
	server := New(Config{Logger: slog.Default()}).Handler()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("health endpoint should remain available, got %d", response.Code)
	}
}
