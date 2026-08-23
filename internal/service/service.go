package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/jobs"
	"pulsegrid/internal/store"
)

type Config struct {
	Repository store.Repository
	Events     *events.Bus
	Jobs       *jobs.Queue
	Metrics    *analytics.Store
	Logger     *slog.Logger
}

type App struct {
	config   Config
	sequence atomic.Uint64
}

type OrganizationInput struct {
	Name     string            `json:"name"`
	Owner    string            `json:"owner"`
	Timezone string            `json:"timezone"`
	Plan     string            `json:"plan"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

type ContactInput struct {
	OrganizationID string            `json:"organization_id"`
	Email          string            `json:"email"`
	Name           string            `json:"name"`
	Company        string            `json:"company"`
	Locale         string            `json:"locale"`
	Timezone       string            `json:"timezone"`
	Tags           []string          `json:"tags"`
	Attributes     map[string]string `json:"attributes"`
}

type AudienceInput struct {
	OrganizationID string          `json:"organization_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	AllOf          []domain.Filter `json:"all_of"`
	AnyOf          []domain.Filter `json:"any_of"`
	Excluded       []domain.Filter `json:"excluded"`
}

type TemplateInput struct {
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Subject        string   `json:"subject"`
	Body           string   `json:"body"`
	Channel        string   `json:"channel"`
	Variables      []string `json:"variables"`
	Published      bool     `json:"published"`
}

type CampaignInput struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	AudienceID     string `json:"audience_id"`
	TemplateID     string `json:"template_id"`
}

type AutomationInput struct {
	OrganizationID string                `json:"organization_id"`
	Name           string                `json:"name"`
	Trigger        string                `json:"trigger"`
	Status         domain.WorkflowStatus `json:"status"`
	Conditions     []domain.Filter       `json:"conditions"`
	Actions        []domain.Action       `json:"actions"`
}

type OverviewOptions struct {
	OrganizationID string
	From           time.Time
	To             time.Time
}

func New(config Config) *App {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &App{config: config}
}

func (a *App) Seed(ctx context.Context) {
	now := time.Now().UTC()
	organization := domain.NewOrganization(a.nextID("org"), "PulseGrid Demo", "system", now)
	organization.Tags = []string{"demo", "starter"}
	if err := a.config.Repository.CreateOrganization(ctx, organization); err != nil {
		a.config.Logger.Debug("seed organization skipped", "error", err)
		return
	}
	template := domain.NewTemplate(a.nextID("tpl"), organization.ID, "Welcome", now)
	template.Subject = "Welcome to PulseGrid"
	template.Body = "Hello {{name}}, welcome to PulseGrid."
	template.Variables = []string{"name"}
	template.Published = true
	if err := a.config.Repository.CreateTemplate(ctx, template); err != nil {
		a.config.Logger.Debug("seed template skipped", "error", err)
	}
}

func (a *App) CreateOrganization(ctx context.Context, input OrganizationInput) (domain.Organization, error) {
	now := time.Now().UTC()
	value := domain.NewOrganization(a.nextID("org"), input.Name, input.Owner, now)
	value.Timezone = input.Timezone
	value.Plan = input.Plan
	value.Tags = domain.NormalizeTags(input.Tags)
	value.Metadata = copyStrings(input.Metadata)
	if value.Timezone == "" {
		value.Timezone = "UTC"
	}
	if value.Plan == "" {
		value.Plan = "starter"
	}
	if err := a.config.Repository.CreateOrganization(ctx, value); err != nil {
		return domain.Organization{}, err
	}
	a.recordMetric(value.ID, "organization.created", 1)
	a.publish(ctx, events.NewEvent(value.ID, "organization.created", value.ID, "", map[string]any{"name": value.Name}))
	return value, nil
}

func (a *App) GetOrganization(ctx context.Context, id string) (domain.Organization, error) {
	return a.config.Repository.GetOrganization(ctx, id)
}

func (a *App) ListOrganizations(ctx context.Context, filter store.OrganizationFilter) ([]domain.Organization, error) {
	return a.config.Repository.ListOrganizations(ctx, filter)
}

func (a *App) UpdateOrganization(ctx context.Context, id string, input OrganizationInput) (domain.Organization, error) {
	value, err := a.config.Repository.GetOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}
	value.Name = strings.TrimSpace(input.Name)
	value.Owner = strings.TrimSpace(input.Owner)
	value.Timezone = strings.TrimSpace(input.Timezone)
	value.Plan = strings.TrimSpace(input.Plan)
	value.Tags = domain.NormalizeTags(input.Tags)
	value.Metadata = copyStrings(input.Metadata)
	value.UpdatedAt = time.Now().UTC()
	if value.Timezone == "" {
		value.Timezone = "UTC"
	}
	if value.Plan == "" {
		value.Plan = "starter"
	}
	if err := a.config.Repository.UpdateOrganization(ctx, value); err != nil {
		return domain.Organization{}, err
	}
	a.publish(ctx, events.NewEvent(value.ID, "organization.updated", value.ID, "", map[string]any{"name": value.Name}))
	return value, nil
}

func (a *App) SuspendOrganization(ctx context.Context, id string) (domain.Organization, error) {
	value, err := a.config.Repository.GetOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}
	if err := value.Suspend(time.Now().UTC()); err != nil {
		return domain.Organization{}, err
	}
	if err := a.config.Repository.UpdateOrganization(ctx, value); err != nil {
		return domain.Organization{}, err
	}
	a.publish(ctx, events.NewEvent(value.ID, "organization.suspended", value.ID, "", nil))
	return value, nil
}

func (a *App) ResumeOrganization(ctx context.Context, id string) (domain.Organization, error) {
	value, err := a.config.Repository.GetOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}
	if err := value.Resume(time.Now().UTC()); err != nil {
		return domain.Organization{}, err
	}
	if err := a.config.Repository.UpdateOrganization(ctx, value); err != nil {
		return domain.Organization{}, err
	}
	a.publish(ctx, events.NewEvent(value.ID, "organization.resumed", value.ID, "", nil))
	return value, nil
}

func (a *App) ArchiveOrganization(ctx context.Context, id string) (domain.Organization, error) {
	value, err := a.config.Repository.GetOrganization(ctx, id)
	if err != nil {
		return domain.Organization{}, err
	}
	if err := value.Archive(time.Now().UTC()); err != nil {
		return domain.Organization{}, err
	}
	if err := a.config.Repository.UpdateOrganization(ctx, value); err != nil {
		return domain.Organization{}, err
	}
	a.publish(ctx, events.NewEvent(value.ID, "organization.archived", value.ID, "", nil))
	return value, nil
}

func (a *App) CreateContact(ctx context.Context, input ContactInput) (domain.Contact, error) {
	now := time.Now().UTC()
	value := domain.NewContact(a.nextID("contact"), input.OrganizationID, input.Email, input.Name, now)
	value.Company = strings.TrimSpace(input.Company)
	value.Locale = input.Locale
	value.Timezone = input.Timezone
	value.Tags = domain.NormalizeTags(input.Tags)
	value.Attributes = copyStrings(input.Attributes)
	if value.Locale == "" {
		value.Locale = "en-US"
	}
	if value.Timezone == "" {
		value.Timezone = "UTC"
	}
	if err := a.config.Repository.CreateContact(ctx, value); err != nil {
		return domain.Contact{}, err
	}
	a.recordMetric(value.OrganizationID, "contact.created", 1)
	a.publish(ctx, events.NewEvent(value.OrganizationID, "contact.created", value.ID, value.ID, map[string]any{
		"email": value.Email, "name": value.Name,
	}))
	return value, nil
}

func (a *App) GetContact(ctx context.Context, id string) (domain.Contact, error) {
	return a.config.Repository.GetContact(ctx, id)
}

func (a *App) ListContacts(ctx context.Context, filter store.ContactFilter) ([]domain.Contact, error) {
	return a.config.Repository.ListContacts(ctx, filter)
}

func (a *App) UpdateContact(ctx context.Context, id string, input ContactInput) (domain.Contact, error) {
	value, err := a.config.Repository.GetContact(ctx, id)
	if err != nil {
		return domain.Contact{}, err
	}
	if input.Email != "" {
		value.Email = strings.ToLower(strings.TrimSpace(input.Email))
	}
	if input.Name != "" {
		value.Name = strings.TrimSpace(input.Name)
	}
	value.Company = strings.TrimSpace(input.Company)
	value.Locale = input.Locale
	value.Timezone = input.Timezone
	value.Tags = domain.NormalizeTags(input.Tags)
	value.Attributes = copyStrings(input.Attributes)
	value.UpdatedAt = time.Now().UTC()
	if value.Locale == "" {
		value.Locale = "en-US"
	}
	if value.Timezone == "" {
		value.Timezone = "UTC"
	}
	if err := a.config.Repository.UpdateContact(ctx, value); err != nil {
		return domain.Contact{}, err
	}
	a.publish(ctx, events.NewEvent(value.OrganizationID, "contact.updated", value.ID, value.ID, nil))
	return value, nil
}

func (a *App) UnsubscribeContact(ctx context.Context, id string) (domain.Contact, error) {
	value, err := a.config.Repository.GetContact(ctx, id)
	if err != nil {
		return domain.Contact{}, err
	}
	if err := value.Unsubscribe(time.Now().UTC()); err != nil {
		return domain.Contact{}, err
	}
	if err := a.config.Repository.UpdateContact(ctx, value); err != nil {
		return domain.Contact{}, err
	}
	a.publish(ctx, events.NewEvent(value.OrganizationID, "contact.unsubscribed", value.ID, value.ID, nil))
	return value, nil
}

func (a *App) SubscribeContact(ctx context.Context, id string) (domain.Contact, error) {
	value, err := a.config.Repository.GetContact(ctx, id)
	if err != nil {
		return domain.Contact{}, err
	}
	if err := value.Subscribe(time.Now().UTC()); err != nil {
		return domain.Contact{}, err
	}
	if err := a.config.Repository.UpdateContact(ctx, value); err != nil {
		return domain.Contact{}, err
	}
	a.publish(ctx, events.NewEvent(value.OrganizationID, "contact.subscribed", value.ID, value.ID, nil))
	return value, nil
}

func (a *App) CreateAudience(ctx context.Context, input AudienceInput) (domain.Audience, error) {
	now := time.Now().UTC()
	value := domain.NewAudience(a.nextID("audience"), input.OrganizationID, input.Name, now)
	value.Description = strings.TrimSpace(input.Description)
	value.AllOf = append([]domain.Filter(nil), input.AllOf...)
	value.AnyOf = append([]domain.Filter(nil), input.AnyOf...)
	value.Excluded = append([]domain.Filter(nil), input.Excluded...)
	if err := a.config.Repository.CreateAudience(ctx, value); err != nil {
		return domain.Audience{}, err
	}
	return value, nil
}

func (a *App) GetAudience(ctx context.Context, id string) (domain.Audience, error) {
	return a.config.Repository.GetAudience(ctx, id)
}

func (a *App) ListAudiences(ctx context.Context, organizationID string) ([]domain.Audience, error) {
	return a.config.Repository.ListAudiences(ctx, organizationID)
}

func (a *App) UpdateAudience(ctx context.Context, id string, input AudienceInput) (domain.Audience, error) {
	value, err := a.config.Repository.GetAudience(ctx, id)
	if err != nil {
		return domain.Audience{}, err
	}
	value.Name = strings.TrimSpace(input.Name)
	value.Description = strings.TrimSpace(input.Description)
	value.AllOf = append([]domain.Filter(nil), input.AllOf...)
	value.AnyOf = append([]domain.Filter(nil), input.AnyOf...)
	value.Excluded = append([]domain.Filter(nil), input.Excluded...)
	value.UpdatedAt = time.Now().UTC()
	if err := a.config.Repository.UpdateAudience(ctx, value); err != nil {
		return domain.Audience{}, err
	}
	return value, nil
}

func (a *App) ResolveAudience(ctx context.Context, id string) ([]domain.Contact, error) {
	audience, err := a.config.Repository.GetAudience(ctx, id)
	if err != nil {
		return nil, err
	}
	contacts, err := a.config.Repository.ListContacts(ctx, store.ContactFilter{OrganizationID: audience.OrganizationID, Limit: 100000})
	if err != nil {
		return nil, err
	}
	result := make([]domain.Contact, 0)
	for _, contact := range contacts {
		if contact.Status != domain.ContactSubscribed {
			continue
		}
		if !matchesContact(contact, audience) {
			continue
		}
		result = append(result, contact)
	}
	return result, nil
}

func (a *App) CreateTemplate(ctx context.Context, input TemplateInput) (domain.Template, error) {
	now := time.Now().UTC()
	value := domain.NewTemplate(a.nextID("template"), input.OrganizationID, input.Name, now)
	value.Subject = strings.TrimSpace(input.Subject)
	value.Body = input.Body
	value.Channel = input.Channel
	value.Variables = unique(input.Variables)
	value.Published = input.Published
	if value.Channel == "" {
		value.Channel = "email"
	}
	if err := a.config.Repository.CreateTemplate(ctx, value); err != nil {
		return domain.Template{}, err
	}
	return value, nil
}

func (a *App) GetTemplate(ctx context.Context, id string) (domain.Template, error) {
	return a.config.Repository.GetTemplate(ctx, id)
}

func (a *App) ListTemplates(ctx context.Context, organizationID string) ([]domain.Template, error) {
	return a.config.Repository.ListTemplates(ctx, organizationID)
}

func (a *App) UpdateTemplate(ctx context.Context, id string, input TemplateInput) (domain.Template, error) {
	value, err := a.config.Repository.GetTemplate(ctx, id)
	if err != nil {
		return domain.Template{}, err
	}
	value.Name = strings.TrimSpace(input.Name)
	value.Subject = strings.TrimSpace(input.Subject)
	value.Body = input.Body
	value.Channel = input.Channel
	value.Variables = unique(input.Variables)
	value.Published = input.Published
	value.Version++
	value.UpdatedAt = time.Now().UTC()
	if value.Channel == "" {
		value.Channel = "email"
	}
	if err := a.config.Repository.UpdateTemplate(ctx, value); err != nil {
		return domain.Template{}, err
	}
	return value, nil
}

func (a *App) CreateCampaign(ctx context.Context, input CampaignInput) (domain.Campaign, error) {
	now := time.Now().UTC()
	value := domain.NewCampaign(a.nextID("campaign"), input.OrganizationID, input.Name, now)
	value.Description = strings.TrimSpace(input.Description)
	value.AudienceID = input.AudienceID
	value.TemplateID = input.TemplateID
	if err := a.config.Repository.CreateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	return value, nil
}

func (a *App) GetCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	return a.config.Repository.GetCampaign(ctx, id)
}

func (a *App) ListCampaigns(ctx context.Context, filter store.CampaignFilter) ([]domain.Campaign, error) {
	return a.config.Repository.ListCampaigns(ctx, filter)
}

func (a *App) ScheduleCampaign(ctx context.Context, id string, at time.Time) (domain.Campaign, error) {
	value, err := a.config.Repository.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := value.Schedule(time.Now().UTC(), at); err != nil {
		return domain.Campaign{}, err
	}
	if err := a.config.Repository.UpdateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	return value, nil
}

func (a *App) StartCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	value, err := a.config.Repository.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := value.Start(time.Now().UTC()); err != nil {
		return domain.Campaign{}, err
	}
	contacts, err := a.ResolveAudience(ctx, value.AudienceID)
	if err != nil {
		return domain.Campaign{}, err
	}
	value.TargetCount = len(contacts)
	if err := a.config.Repository.UpdateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	for _, contact := range contacts {
		delivery := domain.Delivery{ID: a.nextID("delivery"), CampaignID: value.ID, ContactID: contact.ID, TemplateID: value.TemplateID, Channel: "email", Status: domain.MessageQueued, QueuedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		if err := a.config.Repository.CreateDelivery(ctx, delivery); err != nil {
			return domain.Campaign{}, err
		}
		if a.config.Jobs != nil {
			_ = a.config.Jobs.Enqueue(ctx, jobs.Job{Type: "deliver-message", Payload: map[string]any{"delivery_id": delivery.ID}, MaxRetry: 2})
		}
	}
	a.publish(ctx, events.NewEvent(value.OrganizationID, "campaign.started", value.ID, "", map[string]any{"target_count": value.TargetCount}))
	return value, nil
}

func (a *App) PauseCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	value, err := a.config.Repository.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := value.Pause(time.Now().UTC()); err != nil {
		return domain.Campaign{}, err
	}
	if err := a.config.Repository.UpdateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	return value, nil
}

func (a *App) CompleteCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	value, err := a.config.Repository.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := value.Complete(time.Now().UTC()); err != nil {
		return domain.Campaign{}, err
	}
	if err := a.config.Repository.UpdateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	return value, nil
}

func (a *App) CancelCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	value, err := a.config.Repository.GetCampaign(ctx, id)
	if err != nil {
		return domain.Campaign{}, err
	}
	if err := value.Cancel(time.Now().UTC()); err != nil {
		return domain.Campaign{}, err
	}
	if err := a.config.Repository.UpdateCampaign(ctx, value); err != nil {
		return domain.Campaign{}, err
	}
	return value, nil
}

func (a *App) ListDeliveries(ctx context.Context, filter store.DeliveryFilter) ([]domain.Delivery, error) {
	return a.config.Repository.ListDeliveries(ctx, filter)
}

func (a *App) GetDelivery(ctx context.Context, id string) (domain.Delivery, error) {
	return a.config.Repository.GetDelivery(ctx, id)
}

func (a *App) RegisterDeliveryEvent(ctx context.Context, id string, status domain.MessageStatus) (domain.Delivery, error) {
	value, err := a.config.Repository.GetDelivery(ctx, id)
	if err != nil {
		return domain.Delivery{}, err
	}
	if err := value.Advance(status, time.Now().UTC()); err != nil {
		return domain.Delivery{}, err
	}
	if err := a.config.Repository.UpdateDelivery(ctx, value); err != nil {
		return domain.Delivery{}, err
	}
	campaign, err := a.config.Repository.GetCampaign(ctx, value.CampaignID)
	if err == nil {
		switch status {
		case domain.MessageSent:
			campaign.SentCount++
		case domain.MessageDelivered:
			campaign.DeliveredCount++
		case domain.MessageOpened:
			campaign.OpenedCount++
		case domain.MessageClicked:
			campaign.ClickedCount++
		case domain.MessageFailed:
			campaign.FailedCount++
		}
		_ = a.config.Repository.UpdateCampaign(ctx, campaign)
		a.recordMetric(campaign.OrganizationID, "delivery."+string(status), 1)
		a.publish(ctx, events.NewEvent(campaign.OrganizationID, "delivery."+string(status), value.ID, value.ContactID, map[string]any{"campaign_id": campaign.ID}))
	}
	return value, nil
}

func (a *App) CreateAutomation(ctx context.Context, input AutomationInput) (domain.Automation, error) {
	now := time.Now().UTC()
	value := domain.NewAutomation(a.nextID("automation"), input.OrganizationID, input.Name, input.Trigger, now)
	value.Status = input.Status
	if value.Status == "" {
		value.Status = domain.WorkflowEnabled
	}
	value.Conditions = append([]domain.Filter(nil), input.Conditions...)
	value.Actions = cloneActions(input.Actions)
	if err := a.config.Repository.CreateAutomation(ctx, value); err != nil {
		return domain.Automation{}, err
	}
	return value, nil
}

func (a *App) GetAutomation(ctx context.Context, id string) (domain.Automation, error) {
	return a.config.Repository.GetAutomation(ctx, id)
}

func (a *App) ListAutomations(ctx context.Context, filter store.AutomationFilter) ([]domain.Automation, error) {
	return a.config.Repository.ListAutomations(ctx, filter)
}

func (a *App) UpdateAutomation(ctx context.Context, id string, input AutomationInput) (domain.Automation, error) {
	value, err := a.config.Repository.GetAutomation(ctx, id)
	if err != nil {
		return domain.Automation{}, err
	}
	value.Name = strings.TrimSpace(input.Name)
	value.Trigger = input.Trigger
	value.Status = input.Status
	value.Conditions = append([]domain.Filter(nil), input.Conditions...)
	value.Actions = cloneActions(input.Actions)
	value.UpdatedAt = time.Now().UTC()
	if value.Status == "" {
		value.Status = domain.WorkflowEnabled
	}
	if err := a.config.Repository.UpdateAutomation(ctx, value); err != nil {
		return domain.Automation{}, err
	}
	return value, nil
}

func (a *App) ProcessEvent(ctx context.Context, event domain.Event) error {
	if !ContextAllowsWork(ctx) {
		return ctx.Err()
	}
	automations, err := a.config.Repository.ListAutomations(ctx, store.AutomationFilter{OrganizationID: event.OrganizationID, Trigger: event.Type, Status: string(domain.WorkflowEnabled), Limit: 1000})
	if err != nil {
		return err
	}
	for _, automation := range automations {
		if !automation.Matches(event) {
			continue
		}
		for _, action := range automation.Actions {
			if err := a.executeAction(ctx, action, event); err != nil {
				a.config.Logger.Warn("automation action failed", "automation_id", automation.ID, "error", err)
				continue
			}
		}
		automation.MarkRun(time.Now().UTC())
		_ = a.config.Repository.UpdateAutomation(ctx, automation)
	}
	return nil
}

func (a *App) HandleRecalculate(ctx context.Context, job jobs.Job) error {
	organizationID, _ := job.Payload["organization_id"].(string)
	if organizationID != "" {
		a.recordMetric(organizationID, "job.recalculate", 1)
	}
	return nil
}

func (a *App) HandleDelivery(ctx context.Context, job jobs.Job) error {
	id, ok := job.Payload["delivery_id"].(string)
	if !ok || id == "" {
		return fmt.Errorf("delivery_id is required")
	}
	_, err := a.RegisterDeliveryEvent(ctx, id, domain.MessageSent)
	return err
}

func (a *App) Overview(ctx context.Context, options OverviewOptions) (domain.Overview, error) {
	organizations, err := a.config.Repository.ListOrganizations(ctx, store.OrganizationFilter{Limit: 100000})
	if err != nil {
		return domain.Overview{}, err
	}
	contacts, err := a.config.Repository.ListContacts(ctx, store.ContactFilter{OrganizationID: options.OrganizationID, Limit: 100000})
	if err != nil {
		return domain.Overview{}, err
	}
	campaigns, err := a.config.Repository.ListCampaigns(ctx, store.CampaignFilter{OrganizationID: options.OrganizationID, Limit: 100000})
	if err != nil {
		return domain.Overview{}, err
	}
	deliveries, err := a.config.Repository.ListDeliveries(ctx, store.DeliveryFilter{Limit: 100000})
	if err != nil {
		return domain.Overview{}, err
	}
	result := domain.Overview{GeneratedAt: time.Now().UTC(), Organizations: len(organizations), Contacts: len(contacts), Campaigns: len(campaigns), Deliveries: len(deliveries)}
	for _, contact := range contacts {
		if contact.Status == domain.ContactSubscribed {
			result.Subscribed++
		}
	}
	for _, campaign := range campaigns {
		if campaign.Status == domain.CampaignRunning {
			result.RunningCampaigns++
		}
	}
	for _, delivery := range deliveries {
		switch delivery.Status {
		case domain.MessageDelivered, domain.MessageOpened, domain.MessageClicked:
			result.Delivered++
		}
		if delivery.Status == domain.MessageOpened || delivery.Status == domain.MessageClicked {
			result.Opened++
		}
		if delivery.Status == domain.MessageClicked {
			result.Clicked++
		}
	}
	if result.Delivered > 0 {
		result.OpenRate = float64(result.Opened) / float64(result.Delivered) * 100
		result.ClickRate = float64(result.Clicked) / float64(result.Delivered) * 100
	}
	return result, nil
}

func (a *App) ExportOrganization(ctx context.Context, organizationID string) ([]byte, error) {
	organization, err := a.config.Repository.GetOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	contacts, err := a.config.Repository.ListContacts(ctx, store.ContactFilter{OrganizationID: organizationID, Limit: 100000})
	if err != nil {
		return nil, err
	}
	audiences, err := a.config.Repository.ListAudiences(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	templates, err := a.config.Repository.ListTemplates(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	campaigns, err := a.config.Repository.ListCampaigns(ctx, store.CampaignFilter{OrganizationID: organizationID, Limit: 100000})
	if err != nil {
		return nil, err
	}
	automations, err := a.config.Repository.ListAutomations(ctx, store.AutomationFilter{OrganizationID: organizationID, Limit: 100000})
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(map[string]any{"organization": organization, "contacts": contacts, "audiences": audiences, "templates": templates, "campaigns": campaigns, "automations": automations}, "", "  ")
}

func (a *App) executeAction(ctx context.Context, action domain.Action, event domain.Event) error {
	switch action.Type {
	case "metric":
		value := 1.0
		if raw, ok := action.Params["value"]; ok {
			_, _ = fmt.Sscanf(raw, "%f", &value)
		}
		a.recordMetric(event.OrganizationID, action.Value, value)
		return nil
	case "enqueue":
		if a.config.Jobs == nil {
			return fmt.Errorf("jobs are not configured")
		}
		payload := map[string]any{"event": event}
		return a.config.Jobs.Enqueue(ctx, jobs.Job{Type: action.Value, Payload: payload, MaxRetry: 1})
	case "tag-contact":
		contact, err := a.config.Repository.GetContact(ctx, event.ContactID)
		if err != nil {
			return err
		}
		contact.Tags = domain.NormalizeTags(append(contact.Tags, action.Value))
		return a.config.Repository.UpdateContact(ctx, contact)
	case "unsubscribe":
		contact, err := a.config.Repository.GetContact(ctx, event.ContactID)
		if err != nil {
			return err
		}
		if err := contact.Unsubscribe(time.Now().UTC()); err != nil {
			return err
		}
		return a.config.Repository.UpdateContact(ctx, contact)
	default:
		return fmt.Errorf("unsupported action %q", action.Type)
	}
}

func (a *App) publish(ctx context.Context, event domain.Event) {
	if a.config.Events != nil {
		_ = a.config.Events.Publish(ctx, event)
	}
	if a.config.Jobs != nil {
		_ = a.config.Jobs.Enqueue(ctx, jobs.Job{Type: "recalculate", Payload: map[string]any{"organization_id": event.OrganizationID}, MaxRetry: 1})
	}
	asyncCtx := events.AsyncContext(ctx)
	go func() { _ = a.ProcessEvent(asyncCtx, event) }()
}

func (a *App) recordMetric(organizationID, name string, value float64) {
	if a.config.Metrics != nil {
		a.config.Metrics.RecordValue(organizationID, name, value, nil, time.Now().UTC())
	}
}

func (a *App) nextID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), a.sequence.Add(1))
}

func matchesContact(contact domain.Contact, audience domain.Audience) bool {
	for _, filter := range audience.AllOf {
		if !matchContactFilter(contact, filter) {
			return false
		}
	}
	if len(audience.AnyOf) > 0 {
		any := false
		for _, filter := range audience.AnyOf {
			if matchContactFilter(contact, filter) {
				any = true
				break
			}
		}
		if !any {
			return false
		}
	}
	for _, filter := range audience.Excluded {
		if matchContactFilter(contact, filter) {
			return false
		}
	}
	return true
}

func matchContactFilter(contact domain.Contact, filter domain.Filter) bool {
	actual, ok := contactField(contact, filter.Field)
	if !ok {
		return false
	}
	switch filter.Operator {
	case "eq":
		return actual == filter.Value
	case "neq":
		return actual != filter.Value
	case "contains":
		return strings.Contains(actual, filter.Value)
	case "prefix":
		return strings.HasPrefix(actual, filter.Value)
	default:
		return false
	}
}

func contactField(contact domain.Contact, field string) (string, bool) {
	switch field {
	case "email":
		return contact.Email, true
	case "name":
		return contact.Name, true
	case "company":
		return contact.Company, true
	case "status":
		return string(contact.Status), true
	case "locale":
		return contact.Locale, true
	case "timezone":
		return contact.Timezone, true
	}
	if value, ok := contact.Attributes[field]; ok {
		return value, true
	}
	for _, tag := range contact.Tags {
		if field == "tag" {
			return tag, true
		}
	}
	return "", false
}

func copyStrings(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range source {
		out[key] = value
	}
	return out
}

func cloneActions(source []domain.Action) []domain.Action {
	out := make([]domain.Action, len(source))
	for index, action := range source {
		action.Params = copyStrings(action.Params)
		out[index] = action
	}
	return out
}

func unique(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}
