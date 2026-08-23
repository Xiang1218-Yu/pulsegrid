package httpapi

import (
	"bytes"
	"context"
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

// Coverage markers: POST /v1/automations/process, App.ProcessEvent, domain.Automation.Matches, eventField.
func TestBug21AutomationEventNilPayload(t *testing.T) {
	repository := store.NewMemory()
	organization := domain.NewOrganization("org-001021", "Acme", "owner", testNow())
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatal(err)
	}
	automation := domain.NewAutomation("automation-001021", organization.ID, "VIP event", "contact.created", testNow())
	automation.Conditions = []domain.Filter{{Field: "segment", Operator: "eq", Value: "vip"}}
	automation.Actions = []domain.Action{{Type: "metric", Value: "segment.matched"}}
	if err := repository.CreateAutomation(context.Background(), automation); err != nil {
		t.Fatal(err)
	}
	app := service.New(service.Config{Repository: repository, Logger: slog.Default()})
	server := New(Config{App: app, Logger: slog.Default()}).Handler()

	body, err := json.Marshal(domain.Event{
		ID:             "event-001021",
		OrganizationID: organization.ID,
		Type:           "contact.created",
		Subject:        "contact-001021",
		Data:           map[string]any{"segment": nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", bytes.NewReader(body))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("automation event should be accepted, got status %d with body %s", response.Code, response.Body.String())
	}
}

func TestBug21AutomationEventWithoutMatchingAutomation(t *testing.T) {
	repository := store.NewMemory()
	organization := domain.NewOrganization("org-001021-regression", "Acme", "owner", testNow())
	if err := repository.CreateOrganization(context.Background(), organization); err != nil {
		t.Fatal(err)
	}
	app := service.New(service.Config{Repository: repository, Logger: slog.Default()})
	server := New(Config{App: app, Logger: slog.Default()}).Handler()
	body := bytes.NewBufferString(`{"id":"event-001021-regression","organization_id":"org-001021-regression","type":"contact.created"}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", body)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("unmatched automation event should be accepted, got %d", response.Code)
	}
}

func testNow() time.Time {
	return time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
}
