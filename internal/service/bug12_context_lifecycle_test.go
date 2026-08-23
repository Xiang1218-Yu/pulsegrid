package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/store"
)

const bug12SourceMarkers = "publish(ctx, event) ProcessEvent"

func newBug12App(t *testing.T) (*App, *store.Memory, domain.Organization) {
	t.Helper()
	repository := store.NewMemory()
	app := New(Config{Repository: repository, Metrics: analytics.New(), Logger: slog.Default()})
	organization, err := app.CreateOrganization(t.Context(), OrganizationInput{Name: "Acme", Owner: "alice"})
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	return app, repository, organization
}

func addBug12Automation(t *testing.T, repository *store.Memory, organization domain.Organization, id string) {
	t.Helper()
	automation := domain.NewAutomation(id, organization.ID, "record event", "contact.created", time.Now().UTC())
	automation.Actions = []domain.Action{{Type: "metric", Value: "contact.created"}}
	if err := repository.CreateAutomation(t.Context(), automation); err != nil {
		t.Fatalf("create automation: %v", err)
	}
}

func TestBug12ContextLifecycleAfterCancellation(t *testing.T) {
	t.Log(bug12SourceMarkers)
	app, repository, organization := newBug12App(t)
	addBug12Automation(t, repository, organization, "automation-12")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app.publish(ctx, domain.Event{OrganizationID: organization.ID, Type: "contact.created"})
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		value, err := repository.GetAutomation(t.Context(), "automation-12")
		if err != nil {
			t.Fatalf("get automation: %v", err)
		}
		if value.RunCount == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("published event was not processed after caller context cancellation")
}

func TestBug12ActiveContextProcessesEventNormally(t *testing.T) {
	app, repository, organization := newBug12App(t)
	addBug12Automation(t, repository, organization, "automation-12-regression")
	err := app.ProcessEvent(context.Background(), domain.Event{OrganizationID: organization.ID, Type: "contact.created"})
	if err != nil {
		t.Fatalf("process event: %v", err)
	}
	value, err := repository.GetAutomation(t.Context(), "automation-12-regression")
	if err != nil {
		t.Fatalf("get automation: %v", err)
	}
	if value.RunCount != 1 {
		t.Fatalf("automation run count = %d, want 1", value.RunCount)
	}
}
