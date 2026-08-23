package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// POST /v1/automations/process, ProcessEvent, StatusForError
func TestBug027AutomationErrorResponse(t *testing.T) {
	repository := store.NewMemory()
	app := service.New(service.Config{Repository: repository})
	organization, err := app.CreateOrganization(context.Background(), service.OrganizationInput{Name: "Acme", Owner: "owner"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if _, err := app.CreateAutomation(context.Background(), service.AutomationInput{
		OrganizationID: organization.ID,
		Name:           "Tag new contacts",
		Trigger:        "contact.created",
		Status:         domain.WorkflowEnabled,
		Actions:        []domain.Action{{Type: "tag-contact", Value: "new"}},
	}); err != nil {
		t.Fatalf("create automation: %v", err)
	}

	server := New(Config{App: app})
	body, _ := json.Marshal(domain.Event{
		OrganizationID: organization.ID,
		Type:           "contact.created",
		Subject:        "missing-contact",
		ContactID:      "missing-contact",
	})
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("automation action failure status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
}

func TestBug027AutomationSuccessRegression(t *testing.T) {
	repository := store.NewMemory()
	app := service.New(service.Config{Repository: repository})
	organization, err := app.CreateOrganization(context.Background(), service.OrganizationInput{Name: "Acme", Owner: "owner"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	if _, err := app.CreateAutomation(context.Background(), service.AutomationInput{
		OrganizationID: organization.ID,
		Name:           "Record event",
		Trigger:        "contact.created",
		Status:         domain.WorkflowEnabled,
		Actions:        []domain.Action{{Type: "metric", Value: "contact.created"}},
	}); err != nil {
		t.Fatalf("create automation: %v", err)
	}

	server := New(Config{App: app})
	body, _ := json.Marshal(domain.Event{OrganizationID: organization.ID, Type: "contact.created"})
	request := httptest.NewRequest(http.MethodPost, "/v1/automations/process", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("successful automation status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
}
