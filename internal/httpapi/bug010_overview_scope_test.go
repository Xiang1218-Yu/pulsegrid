package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// Source markers: GET /v1/overview App.Overview ListOrganizations ListDeliveries OrganizationID
func TestBug010OverviewScopeIsolation(t *testing.T) {
	app, _, organizationID := newBug010App(t)
	server := New(Config{App: app})

	request := httptest.NewRequest(http.MethodGet, "/v1/overview?organization_id="+organizationID, nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected overview success, got %d: %s", response.Code, response.Body.String())
	}

	var overview domain.Overview
	if err := json.NewDecoder(response.Body).Decode(&overview); err != nil {
		t.Fatal(err)
	}
	if overview.Organizations != 1 || overview.Contacts != 1 || overview.Campaigns != 1 || overview.Deliveries != 1 || overview.Delivered != 1 {
		t.Fatalf("overview leaked another organization: %+v", overview)
	}
}

func TestBug010OverviewScopeUnfiltered(t *testing.T) {
	app, _, _ := newBug010App(t)
	server := New(Config{App: app})

	request := httptest.NewRequest(http.MethodGet, "/v1/overview", nil)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)

	var overview domain.Overview
	if err := json.NewDecoder(response.Body).Decode(&overview); err != nil {
		t.Fatal(err)
	}
	if overview.Organizations != 2 || overview.Contacts != 2 || overview.Campaigns != 2 || overview.Deliveries != 2 {
		t.Fatalf("unexpected unfiltered overview: %+v", overview)
	}
}

func newBug010App(t *testing.T) (*service.App, *store.Memory, string) {
	t.Helper()
	ctx := context.Background()
	repository := store.NewMemory()
	app := service.New(service.Config{Repository: repository})
	organizationIDs := make([]string, 0, 2)
	for _, name := range []string{"Acme", "Beta"} {
		organization, err := app.CreateOrganization(ctx, service.OrganizationInput{Name: name, Owner: "owner"})
		if err != nil {
			t.Fatal(err)
		}
		organizationIDs = append(organizationIDs, organization.ID)
		audience, err := app.CreateAudience(ctx, service.AudienceInput{OrganizationID: organization.ID, Name: "Audience"})
		if err != nil {
			t.Fatal(err)
		}
		template, err := app.CreateTemplate(ctx, service.TemplateInput{
			OrganizationID: organization.ID,
			Name:           "Template",
			Subject:        "Subject",
			Body:           "Body",
			Channel:        "email",
		})
		if err != nil {
			t.Fatal(err)
		}
		contact, err := app.CreateContact(ctx, service.ContactInput{
			OrganizationID: organization.ID,
			Email:          name + "@example.com",
			Name:           name,
		})
		if err != nil {
			t.Fatal(err)
		}
		campaign, err := app.CreateCampaign(ctx, service.CampaignInput{
			OrganizationID: organization.ID,
			Name:           "Campaign",
			AudienceID:     audience.ID,
			TemplateID:     template.ID,
		})
		if err != nil {
			t.Fatal(err)
		}
		now := time.Now().UTC()
		if err := repository.CreateDelivery(ctx, domain.Delivery{
			ID:         "delivery-" + organization.ID,
			CampaignID: campaign.ID,
			ContactID:  contact.ID,
			TemplateID: template.ID,
			Channel:    "email",
			Status:     domain.MessageDelivered,
			QueuedAt:   now,
			UpdatedAt:  now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return app, repository, organizationIDs[0]
}
