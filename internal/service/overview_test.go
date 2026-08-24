package service_test

import (
	"context"
	"testing"
	"time"

	"pulsegrid/internal/domain"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

// newTestApp returns an App wired to an in-memory repository. Only the
// repository is configured; the optional event bus, job queue, and metrics
// store are left nil, which the service guards against.
func newTestApp(t *testing.T) (*service.App, *store.Memory) {
	t.Helper()
	repo := store.NewMemory()
	app := service.New(service.Config{Repository: repo})
	return app, repo
}

// seedOrg provisions an organization with one subscribed contact, one running
// campaign, and one delivered delivery so every overview counter is exercised.
// All entities are created through the repository so the org-scoped relations
// (delivery -> campaign -> organization) are populated exactly as in production.
func seedOrg(t *testing.T, repo store.Repository, orgID, suffix string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	org := domain.NewOrganization(orgID, "Org "+suffix, "owner-"+suffix, now)
	if err := repo.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("create organization %q: %v", orgID, err)
	}

	audience := domain.NewAudience("aud-"+suffix, orgID, "Audience "+suffix, now)
	if err := repo.CreateAudience(ctx, audience); err != nil {
		t.Fatalf("create audience %q: %v", orgID, err)
	}

	template := domain.NewTemplate("tpl-"+suffix, orgID, "Template "+suffix, now)
	template.Subject = "Subject " + suffix
	template.Body = "Body " + suffix
	if err := repo.CreateTemplate(ctx, template); err != nil {
		t.Fatalf("create template %q: %v", orgID, err)
	}

	contact := domain.NewContact("contact-"+suffix, orgID, "person"+suffix+"@example.com", "Person "+suffix, now)
	if err := repo.CreateContact(ctx, contact); err != nil {
		t.Fatalf("create contact %q: %v", orgID, err)
	}

	campaign := domain.NewCampaign("campaign-"+suffix, orgID, "Campaign "+suffix, now)
	campaign.AudienceID = audience.ID
	campaign.TemplateID = template.ID
	campaign.Status = domain.CampaignRunning
	if err := repo.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("create campaign %q: %v", orgID, err)
	}

	delivery := domain.Delivery{
		ID:         "delivery-"+suffix,
		CampaignID: campaign.ID,
		ContactID:  contact.ID,
		TemplateID: template.ID,
		Channel:    "email",
		Status:     domain.MessageDelivered,
		QueuedAt:   now,
		UpdatedAt:  now,
	}
	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("create delivery %q: %v", orgID, err)
	}
}

func TestOverviewSingleOrganizationIsScoped(t *testing.T) {
	app, repo := newTestApp(t)
	seedOrg(t, repo, "org-a", "A")
	seedOrg(t, repo, "org-b", "B")

	ctx := context.Background()
	got, err := app.Overview(ctx, service.OverviewOptions{OrganizationID: "org-a"})
	if err != nil {
		t.Fatalf("overview for org-a: %v", err)
	}

	// A single-organization overview must report exactly that organization and
	// only its own contacts, campaigns, and deliveries — never the totals from
	// other organizations. Before the fix, Organizations leaked every org (2).
	if got.Organizations != 1 {
		t.Errorf("Organizations = %d, want 1 (single-org overview must not leak other orgs)", got.Organizations)
	}
	if got.Contacts != 1 {
		t.Errorf("Contacts = %d, want 1", got.Contacts)
	}
	if got.Subscribed != 1 {
		t.Errorf("Subscribed = %d, want 1", got.Subscribed)
	}
	if got.Campaigns != 1 {
		t.Errorf("Campaigns = %d, want 1", got.Campaigns)
	}
	if got.RunningCampaigns != 1 {
		t.Errorf("RunningCampaigns = %d, want 1", got.RunningCampaigns)
	}
	if got.Deliveries != 1 {
		t.Errorf("Deliveries = %d, want 1", got.Deliveries)
	}
	if got.Delivered != 1 {
		t.Errorf("Delivered = %d, want 1", got.Delivered)
	}
}

func TestOverviewUnfilteredReportsAllOrganizations(t *testing.T) {
	app, repo := newTestApp(t)
	seedOrg(t, repo, "org-a", "A")
	seedOrg(t, repo, "org-b", "B")

	ctx := context.Background()
	got, err := app.Overview(ctx, service.OverviewOptions{})
	if err != nil {
		t.Fatalf("global overview: %v", err)
	}

	// The unfiltered overview must aggregate every organization's data. The
	// scoped single-org path and this global path must agree in shape but differ
	// only in scope, so the totals reflect both organizations.
	if got.Organizations != 2 {
		t.Errorf("Organizations = %d, want 2 (unfiltered overview must include all orgs)", got.Organizations)
	}
	if got.Contacts != 2 {
		t.Errorf("Contacts = %d, want 2", got.Contacts)
	}
	if got.Campaigns != 2 {
		t.Errorf("Campaigns = %d, want 2", got.Campaigns)
	}
	if got.Deliveries != 2 {
		t.Errorf("Deliveries = %d, want 2", got.Deliveries)
	}
	if got.Subscribed != 2 {
		t.Errorf("Subscribed = %d, want 2", got.Subscribed)
	}
	if got.RunningCampaigns != 2 {
		t.Errorf("RunningCampaigns = %d, want 2", got.RunningCampaigns)
	}
	if got.Delivered != 2 {
		t.Errorf("Delivered = %d, want 2", got.Delivered)
	}
}

func TestOverviewUnknownOrganizationIsEmpty(t *testing.T) {
	app, repo := newTestApp(t)
	seedOrg(t, repo, "org-a", "A")

	ctx := context.Background()
	got, err := app.Overview(ctx, service.OverviewOptions{OrganizationID: "does-not-exist"})
	if err != nil {
		t.Fatalf("overview for unknown org: %v", err)
	}

	// A scoped overview for an organization that does not exist must be
	// consistently empty across every counter — no leakage from real orgs and
	// no mismatch between the org count and the related counters.
	if got.Organizations != 0 || got.Contacts != 0 || got.Campaigns != 0 || got.Deliveries != 0 {
		t.Errorf("overview for unknown org is not empty: %+v", got)
	}
}
