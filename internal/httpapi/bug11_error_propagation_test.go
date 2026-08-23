package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

const bug11SourceMarker = "POST /v1/automations/process"

func newBug11Server(t *testing.T) (*Server, *store.Memory, domain.Organization) {
	t.Helper()
	repository := store.NewMemory()
	app := service.New(service.Config{
		Repository: repository,
		Logger:     slog.Default(),
	})
	organization, err := app.CreateOrganization(t.Context(), service.OrganizationInput{Name: "Acme", Owner: "alice"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return New(Config{App: app, Logger: slog.Default()}), repository, organization
}

func TestBug11ErrorPropagationReachesHTTPStatus(t *testing.T) {
	t.Log(bug11SourceMarker)
	server, repository, organization := newBug11Server(t)
	automation := domain.NewAutomation("automation-11", organization.ID, "tag missing contact", "contact.created", tNow())
	automation.Actions = []domain.Action{{Type: "tag-contact", Value: "vip"}}
	if err := repository.CreateAutomation(t.Context(), automation); err != nil {
		t.Fatalf("create automation: %v", err)
	}
	body, err := json.Marshal(domain.Event{
		OrganizationID: organization.ID,
		Type:           "contact.created",
		ContactID:      "missing-contact",
	})
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("automation error status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestBug11AutomationMetricActionStillAccepted(t *testing.T) {
	server, repository, organization := newBug11Server(t)
	automation := domain.NewAutomation("automation-11-regression", organization.ID, "record metric", "contact.created", tNow())
	automation.Actions = []domain.Action{{Type: "metric", Value: "contact.created"}}
	if err := repository.CreateAutomation(t.Context(), automation); err != nil {
		t.Fatalf("create automation: %v", err)
	}
	body, err := json.Marshal(domain.Event{OrganizationID: organization.ID, Type: "contact.created"})
	if err != nil {
		t.Fatalf("encode event: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("successful automation status = %d, want %d", response.Code, http.StatusAccepted)
	}
}

func tNow() (value time.Time) {
	return time.Now().UTC()
}
