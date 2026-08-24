package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pulsegrid/internal/jobs"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// Source markers: Queue.Enqueue StatusForError writeError campaign start
func TestBug006CampaignErrorQueueFailure(t *testing.T) {
	app, server := newBug006App(t, false)
	campaignID := seedBug006Campaign(t, app)

	request := httptest.NewRequest(http.MethodPost, "/v1/campaigns/"+campaignID+"/start", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected queue failure to reach HTTP as 503, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "queue is not started") {
		t.Fatalf("expected response to retain queue error, got %s", response.Body.String())
	}
}

func TestBug006CampaignErrorQueueRegression(t *testing.T) {
	app, server := newBug006App(t, true)
	campaignID := seedBug006Campaign(t, app)

	request := httptest.NewRequest(http.MethodPost, "/v1/campaigns/"+campaignID+"/start", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected running queue to allow campaign start, got %d: %s", response.Code, response.Body.String())
	}
}

func newBug006App(t *testing.T, startQueue bool) (*service.App, *Server) {
	t.Helper()
	repository := store.NewMemory()
	queue := jobs.New(jobs.Config{Workers: 1, Capacity: 4})
	if startQueue {
		queue.Start()
		t.Cleanup(queue.Stop)
	}
	app := service.New(service.Config{Repository: repository, Jobs: queue})
	return app, New(Config{App: app, Jobs: queue})
}

func seedBug006Campaign(t *testing.T, app *service.App) string {
	t.Helper()
	ctx := context.Background()
	organization, err := app.CreateOrganization(ctx, service.OrganizationInput{Name: "Acme", Owner: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	audience, err := app.CreateAudience(ctx, service.AudienceInput{OrganizationID: organization.ID, Name: "Subscribers"})
	if err != nil {
		t.Fatal(err)
	}
	template, err := app.CreateTemplate(ctx, service.TemplateInput{
		OrganizationID: organization.ID,
		Name:           "Welcome",
		Subject:        "Hello",
		Body:           "Welcome {{name}}",
		Channel:        "email",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateContact(ctx, service.ContactInput{
		OrganizationID: organization.ID,
		Email:          "person@example.com",
		Name:           "Person",
	}); err != nil {
		t.Fatal(err)
	}
	campaign, err := app.CreateCampaign(ctx, service.CampaignInput{
		OrganizationID: organization.ID,
		Name:           "Welcome campaign",
		AudienceID:     audience.ID,
		TemplateID:     template.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	return campaign.ID
}
