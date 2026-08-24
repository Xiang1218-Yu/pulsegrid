package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/jobs"
	"pulsegrid/internal/store"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestApp(t *testing.T, startQueue bool) (*App, *jobs.Queue) {
	t.Helper()
	repository := store.NewMemory()
	bus := events.New(events.Config{Buffer: 16, DropWhenBusy: true})
	workers := jobs.New(jobs.Config{Workers: 1, Capacity: 4, Logger: testLogger()})
	metrics := analytics.New()
	app := New(Config{Repository: repository, Events: bus, Jobs: workers, Metrics: metrics, Logger: testLogger()})
	workers.Register(jobs.JobDeliver, app.HandleDelivery)
	if startQueue {
		workers.Start()
	}
	return app, workers
}

func seedRunningFixture(t *testing.T, app *App) domain.Campaign {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	org := domain.NewOrganization(app.nextID("org"), "Acme", "ops", now)
	if err := app.config.Repository.CreateOrganization(ctx, org); err != nil {
		t.Fatalf("create org: %v", err)
	}
	template := domain.NewTemplate(app.nextID("tpl"), org.ID, "Welcome", now)
	template.Subject = "Hi"
	template.Body = "Hello {{name}}"
	template.Published = true
	if err := app.config.Repository.CreateTemplate(ctx, template); err != nil {
		t.Fatalf("create template: %v", err)
	}
	audience := domain.NewAudience(app.nextID("aud"), org.ID, "Subscribers", now)
	if err := app.config.Repository.CreateAudience(ctx, audience); err != nil {
		t.Fatalf("create audience: %v", err)
	}
	contact := domain.NewContact(app.nextID("contact"), org.ID, "person@example.com", "Person", now)
	if err := app.config.Repository.CreateContact(ctx, contact); err != nil {
		t.Fatalf("create contact: %v", err)
	}
	campaign := domain.NewCampaign(app.nextID("campaign"), org.ID, "Launch", now)
	campaign.AudienceID = audience.ID
	campaign.TemplateID = template.ID
	if err := app.config.Repository.CreateCampaign(ctx, campaign); err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	return campaign
}

// When the queue is up, starting a campaign enqueues deliveries and returns the
// running campaign — the normal behavior the fix must preserve.
func TestStartCampaignHappyPath(t *testing.T) {
	app, workers := newTestApp(t, true)
	defer workers.Stop()
	campaign := seedRunningFixture(t, app)

	value, err := app.StartCampaign(context.Background(), campaign.ID)
	if err != nil {
		t.Fatalf("start campaign: %v", err)
	}
	if value.Status != domain.CampaignRunning {
		t.Fatalf("status = %q, want %q", value.Status, domain.CampaignRunning)
	}
	if value.TargetCount != 1 {
		t.Fatalf("target count = %d, want 1", value.TargetCount)
	}
}

// When background processing is unavailable (queue not started), StartCampaign
// must surface a retryable error rather than an opaque 500 so callers can retry.
// The campaign record is already running at that point.
func TestStartCampaignQueueUnavailableIsRetryable(t *testing.T) {
	app, workers := newTestApp(t, false)
	defer workers.Stop()
	campaign := seedRunningFixture(t, app)

	_, err := app.StartCampaign(context.Background(), campaign.ID)
	if err == nil {
		t.Fatalf("start campaign: want error, got nil")
	}
	if !errors.Is(err, jobs.ErrUnavailable) {
		t.Fatalf("start campaign: want wrapped %v, got %v", jobs.ErrUnavailable, err)
	}
	if !jobs.Retryable(err) {
		t.Fatalf("start campaign: want retryable, got %v", err)
	}
	stored, getErr := app.config.Repository.GetCampaign(context.Background(), campaign.ID)
	if getErr != nil {
		t.Fatalf("get campaign: %v", getErr)
	}
	if stored.Status != domain.CampaignRunning {
		t.Fatalf("stored status = %q, want %q (already running)", stored.Status, domain.CampaignRunning)
	}
}
