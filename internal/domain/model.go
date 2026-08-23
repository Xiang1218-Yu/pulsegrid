package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type OrganizationStatus string

const (
	OrganizationActive    OrganizationStatus = "active"
	OrganizationSuspended OrganizationStatus = "suspended"
	OrganizationArchived  OrganizationStatus = "archived"
)

type ContactStatus string

const (
	ContactSubscribed   ContactStatus = "subscribed"
	ContactUnsubscribed ContactStatus = "unsubscribed"
	ContactBounced      ContactStatus = "bounced"
	ContactArchived     ContactStatus = "archived"
)

type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignScheduled CampaignStatus = "scheduled"
	CampaignRunning   CampaignStatus = "running"
	CampaignPaused    CampaignStatus = "paused"
	CampaignCompleted CampaignStatus = "completed"
	CampaignCancelled CampaignStatus = "cancelled"
)

type MessageStatus string

const (
	MessageQueued     MessageStatus = "queued"
	MessageSent       MessageStatus = "sent"
	MessageDelivered  MessageStatus = "delivered"
	MessageOpened     MessageStatus = "opened"
	MessageClicked    MessageStatus = "clicked"
	MessageFailed     MessageStatus = "failed"
	MessageSuppressed MessageStatus = "suppressed"
)

type WorkflowStatus string

const (
	WorkflowEnabled  WorkflowStatus = "enabled"
	WorkflowDisabled WorkflowStatus = "disabled"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrConflict      = errors.New("resource conflict")
	ErrInvalidInput  = errors.New("invalid input")
	ErrInvalidState  = errors.New("invalid state transition")
	ErrAlreadyExists = errors.New("resource already exists")
)

type Organization struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Owner      string             `json:"owner"`
	Timezone   string             `json:"timezone"`
	Status     OrganizationStatus `json:"status"`
	Plan       string             `json:"plan"`
	Tags       []string           `json:"tags,omitempty"`
	Metadata   map[string]string  `json:"metadata,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	ArchivedAt *time.Time         `json:"archived_at,omitempty"`
}

type Contact struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organization_id"`
	Email          string            `json:"email"`
	Name           string            `json:"name"`
	Company        string            `json:"company,omitempty"`
	Status         ContactStatus     `json:"status"`
	Locale         string            `json:"locale"`
	Timezone       string            `json:"timezone"`
	Tags           []string          `json:"tags,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	SubscribedAt   *time.Time        `json:"subscribed_at,omitempty"`
	UnsubscribedAt *time.Time        `json:"unsubscribed_at,omitempty"`
	LastEngagedAt  *time.Time        `json:"last_engaged_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

type Audience struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	AllOf          []Filter  `json:"all_of,omitempty"`
	AnyOf          []Filter  `json:"any_of,omitempty"`
	Excluded       []Filter  `json:"excluded,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Filter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type Template struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Subject        string    `json:"subject"`
	Body           string    `json:"body"`
	Channel        string    `json:"channel"`
	Version        int       `json:"version"`
	Variables      []string  `json:"variables,omitempty"`
	Published      bool      `json:"published"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Campaign struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	AudienceID     string         `json:"audience_id"`
	TemplateID     string         `json:"template_id"`
	Status         CampaignStatus `json:"status"`
	ScheduledAt    *time.Time     `json:"scheduled_at,omitempty"`
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	TargetCount    int            `json:"target_count"`
	SentCount      int            `json:"sent_count"`
	DeliveredCount int            `json:"delivered_count"`
	OpenedCount    int            `json:"opened_count"`
	ClickedCount   int            `json:"clicked_count"`
	FailedCount    int            `json:"failed_count"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Delivery struct {
	ID          string        `json:"id"`
	CampaignID  string        `json:"campaign_id"`
	ContactID   string        `json:"contact_id"`
	TemplateID  string        `json:"template_id"`
	Channel     string        `json:"channel"`
	Status      MessageStatus `json:"status"`
	ProviderID  string        `json:"provider_id,omitempty"`
	Attempts    int           `json:"attempts"`
	LastError   string        `json:"last_error,omitempty"`
	QueuedAt    time.Time     `json:"queued_at"`
	SentAt      *time.Time    `json:"sent_at,omitempty"`
	DeliveredAt *time.Time    `json:"delivered_at,omitempty"`
	OpenedAt    *time.Time    `json:"opened_at,omitempty"`
	ClickedAt   *time.Time    `json:"clicked_at,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type Automation struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Name           string         `json:"name"`
	Status         WorkflowStatus `json:"status"`
	Trigger        string         `json:"trigger"`
	Conditions     []Filter       `json:"conditions,omitempty"`
	Actions        []Action       `json:"actions"`
	RunCount       int            `json:"run_count"`
	LastRunAt      *time.Time     `json:"last_run_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Action struct {
	Type   string            `json:"type"`
	Value  string            `json:"value,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

type Event struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	Type           string         `json:"type"`
	Subject        string         `json:"subject"`
	ContactID      string         `json:"contact_id,omitempty"`
	Data           map[string]any `json:"data,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at"`
}

type Metric struct {
	ID             string            `json:"id"`
	OrganizationID string            `json:"organization_id"`
	Name           string            `json:"name"`
	Value          float64           `json:"value"`
	Dimensions     map[string]string `json:"dimensions,omitempty"`
	RecordedAt     time.Time         `json:"recorded_at"`
}

type Overview struct {
	GeneratedAt      time.Time `json:"generated_at"`
	Organizations    int       `json:"organizations"`
	Contacts         int       `json:"contacts"`
	Subscribed       int       `json:"subscribed"`
	Campaigns        int       `json:"campaigns"`
	RunningCampaigns int       `json:"running_campaigns"`
	Deliveries       int       `json:"deliveries"`
	Delivered        int       `json:"delivered"`
	Opened           int       `json:"opened"`
	Clicked          int       `json:"clicked"`
	OpenRate         float64   `json:"open_rate"`
	ClickRate        float64   `json:"click_rate"`
}

func NewOrganization(id, name, owner string, now time.Time) Organization {
	return Organization{ID: id, Name: strings.TrimSpace(name), Owner: strings.TrimSpace(owner), Timezone: "UTC", Status: OrganizationActive, Plan: "starter", Tags: []string{}, Metadata: map[string]string{}, CreatedAt: now, UpdatedAt: now}
}

func NewContact(id, organizationID, email, name string, now time.Time) Contact {
	return Contact{ID: id, OrganizationID: organizationID, Email: strings.ToLower(strings.TrimSpace(email)), Name: strings.TrimSpace(name), Status: ContactSubscribed, Locale: "en-US", Timezone: "UTC", Tags: []string{}, Attributes: map[string]string{}, CreatedAt: now, UpdatedAt: now}
}

func NewAudience(id, organizationID, name string, now time.Time) Audience {
	return Audience{ID: id, OrganizationID: organizationID, Name: strings.TrimSpace(name), AllOf: []Filter{}, AnyOf: []Filter{}, Excluded: []Filter{}, CreatedAt: now, UpdatedAt: now}
}

func NewTemplate(id, organizationID, name string, now time.Time) Template {
	return Template{ID: id, OrganizationID: organizationID, Name: strings.TrimSpace(name), Channel: "email", Version: 1, Variables: []string{}, CreatedAt: now, UpdatedAt: now}
}

func NewCampaign(id, organizationID, name string, now time.Time) Campaign {
	return Campaign{ID: id, OrganizationID: organizationID, Name: strings.TrimSpace(name), Status: CampaignDraft, CreatedAt: now, UpdatedAt: now}
}

func NewAutomation(id, organizationID, name, trigger string, now time.Time) Automation {
	return Automation{ID: id, OrganizationID: organizationID, Name: strings.TrimSpace(name), Status: WorkflowEnabled, Trigger: trigger, Conditions: []Filter{}, Actions: []Action{}, CreatedAt: now, UpdatedAt: now}
}

func (o Organization) Validate() error {
	if strings.TrimSpace(o.Name) == "" || len([]rune(o.Name)) > 160 {
		return fmt.Errorf("organization name: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(o.Owner) == "" {
		return fmt.Errorf("organization owner: %w", ErrInvalidInput)
	}
	if !validOrganizationStatus(o.Status) {
		return fmt.Errorf("organization status: %w", ErrInvalidInput)
	}
	return nil
}

func (c Contact) Validate() error {
	if strings.TrimSpace(c.OrganizationID) == "" {
		return fmt.Errorf("organization id: %w", ErrInvalidInput)
	}
	if !strings.Contains(c.Email, "@") || len(c.Email) > 255 {
		return fmt.Errorf("email: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("contact name: %w", ErrInvalidInput)
	}
	if !validContactStatus(c.Status) {
		return fmt.Errorf("contact status: %w", ErrInvalidInput)
	}
	return nil
}

func (a Audience) Validate() error {
	if strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("audience fields: %w", ErrInvalidInput)
	}
	if len(a.AllOf)+len(a.AnyOf)+len(a.Excluded) > 30 {
		return fmt.Errorf("too many filters: %w", ErrInvalidInput)
	}
	return nil
}

func (t Template) Validate() error {
	if strings.TrimSpace(t.OrganizationID) == "" || strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("template fields: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(t.Subject) == "" || strings.TrimSpace(t.Body) == "" {
		return fmt.Errorf("template content: %w", ErrInvalidInput)
	}
	if t.Channel != "email" && t.Channel != "sms" && t.Channel != "push" {
		return fmt.Errorf("template channel: %w", ErrInvalidInput)
	}
	return nil
}

func (c Campaign) Validate() error {
	if strings.TrimSpace(c.OrganizationID) == "" || strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("campaign fields: %w", ErrInvalidInput)
	}
	if c.AudienceID == "" || c.TemplateID == "" {
		return fmt.Errorf("campaign references: %w", ErrInvalidInput)
	}
	if !validCampaignStatus(c.Status) {
		return fmt.Errorf("campaign status: %w", ErrInvalidInput)
	}
	return nil
}

func (a Automation) Validate() error {
	if strings.TrimSpace(a.OrganizationID) == "" || strings.TrimSpace(a.Name) == "" || strings.TrimSpace(a.Trigger) == "" {
		return fmt.Errorf("automation fields: %w", ErrInvalidInput)
	}
	if len(a.Actions) == 0 {
		return fmt.Errorf("automation actions: %w", ErrInvalidInput)
	}
	if a.Status != WorkflowEnabled && a.Status != WorkflowDisabled {
		return fmt.Errorf("automation status: %w", ErrInvalidInput)
	}
	return nil
}

func (o *Organization) Suspend(now time.Time) error {
	if o.Status != OrganizationActive {
		return ErrInvalidState
	}
	o.Status = OrganizationSuspended
	o.UpdatedAt = now
	return nil
}

func (o *Organization) Resume(now time.Time) error {
	if o.Status != OrganizationSuspended {
		return ErrInvalidState
	}
	o.Status = OrganizationActive
	o.UpdatedAt = now
	return nil
}

func (o *Organization) Archive(now time.Time) error {
	if o.Status == OrganizationArchived {
		return ErrInvalidState
	}
	o.Status = OrganizationArchived
	o.ArchivedAt = timePtr(now)
	o.UpdatedAt = now
	return nil
}

func (c *Contact) Unsubscribe(now time.Time) error {
	if c.Status == ContactArchived || c.Status == ContactUnsubscribed {
		return ErrInvalidState
	}
	c.Status = ContactUnsubscribed
	c.UnsubscribedAt = timePtr(now)
	c.UpdatedAt = now
	return nil
}

func (c *Contact) Subscribe(now time.Time) error {
	if c.Status == ContactArchived || c.Status == ContactBounced {
		return ErrInvalidState
	}
	c.Status = ContactSubscribed
	c.SubscribedAt = timePtr(now)
	c.UnsubscribedAt = nil
	c.UpdatedAt = now
	return nil
}

func (c *Campaign) Schedule(now, at time.Time) error {
	if c.Status != CampaignDraft && c.Status != CampaignPaused {
		return ErrInvalidState
	}
	if at.Before(now) {
		return fmt.Errorf("schedule time is in the past: %w", ErrInvalidInput)
	}
	c.Status = CampaignScheduled
	c.ScheduledAt = timePtr(at)
	c.UpdatedAt = now
	return nil
}

func (c *Campaign) Start(now time.Time) error {
	if c.Status != CampaignDraft && c.Status != CampaignScheduled && c.Status != CampaignPaused {
		return ErrInvalidState
	}
	c.Status = CampaignRunning
	c.StartedAt = timePtr(now)
	c.UpdatedAt = now
	return nil
}

func (c *Campaign) Pause(now time.Time) error {
	if c.Status != CampaignRunning {
		return ErrInvalidState
	}
	c.Status = CampaignPaused
	c.UpdatedAt = now
	return nil
}

func (c *Campaign) Complete(now time.Time) error {
	if c.Status != CampaignRunning && c.Status != CampaignPaused {
		return ErrInvalidState
	}
	c.Status = CampaignCompleted
	c.CompletedAt = timePtr(now)
	c.UpdatedAt = now
	return nil
}

func (c *Campaign) Cancel(now time.Time) error {
	if c.Status == CampaignCompleted || c.Status == CampaignCancelled {
		return ErrInvalidState
	}
	c.Status = CampaignCancelled
	c.UpdatedAt = now
	return nil
}

func (d *Delivery) Advance(next MessageStatus, now time.Time) error {
	if DeliveryStatusRank(next) < DeliveryStatusRank(d.Status) {
		d.Status = next
		d.UpdatedAt = now
		return nil
	}
	if !deliveryTransition(d.Status, next) {
		return ErrInvalidState
	}
	d.Status = next
	d.Attempts++
	d.UpdatedAt = now
	switch next {
	case MessageSent:
		d.SentAt = timePtr(now)
	case MessageDelivered:
		d.DeliveredAt = timePtr(now)
	case MessageOpened:
		d.OpenedAt = timePtr(now)
	case MessageClicked:
		d.ClickedAt = timePtr(now)
	}
	return nil
}

func (a *Automation) MarkRun(now time.Time) {
	a.RunCount++
	a.LastRunAt = timePtr(now)
	a.UpdatedAt = now
}

func (a Automation) Matches(event Event) bool {
	if a.Status != WorkflowEnabled || a.Trigger != event.Type {
		return false
	}
	for _, filter := range a.Conditions {
		if !matchesFilter(filter, event) {
			return false
		}
	}
	return true
}

func matchesFilter(filter Filter, event Event) bool {
	actual, ok := eventField(filter.Field, event)
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

func eventField(field string, event Event) (string, bool) {
	switch field {
	case "type":
		return event.Type, true
	case "subject":
		return event.Subject, true
	case "contact_id":
		return event.ContactID, true
	case "organization_id":
		return event.OrganizationID, true
	}
	value, ok := event.Data[field]
	if !ok {
		return "", false
	}
	return fmt.Sprint(value), true
}

func validOrganizationStatus(value OrganizationStatus) bool {
	return value == OrganizationActive || value == OrganizationSuspended || value == OrganizationArchived
}
func validContactStatus(value ContactStatus) bool {
	return value == ContactSubscribed || value == ContactUnsubscribed || value == ContactBounced || value == ContactArchived
}
func validCampaignStatus(value CampaignStatus) bool {
	return value == CampaignDraft || value == CampaignScheduled || value == CampaignRunning || value == CampaignPaused || value == CampaignCompleted || value == CampaignCancelled
}

func deliveryTransition(from, to MessageStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case MessageQueued:
		return to == MessageSent || to == MessageFailed || to == MessageSuppressed
	case MessageSent:
		return to == MessageDelivered || to == MessageFailed
	case MessageDelivered:
		return to == MessageOpened || to == MessageFailed
	case MessageOpened:
		return to == MessageClicked
	default:
		return false
	}
}

func (o Organization) Clone() Organization {
	out := o
	out.Tags = append([]string(nil), o.Tags...)
	out.Metadata = cloneStrings(o.Metadata)
	return out
}
func (c Contact) Clone() Contact {
	out := c
	out.Tags = append([]string(nil), c.Tags...)
	out.Attributes = cloneStrings(c.Attributes)
	return out
}
func (a Audience) Clone() Audience {
	out := a
	out.AllOf = append([]Filter(nil), a.AllOf...)
	out.AnyOf = append([]Filter(nil), a.AnyOf...)
	out.Excluded = append([]Filter(nil), a.Excluded...)
	return out
}
func (t Template) Clone() Template {
	out := t
	out.Variables = append([]string(nil), t.Variables...)
	return out
}
func (c Campaign) Clone() Campaign { return c }
func (d Delivery) Clone() Delivery { return d }
func (a Automation) Clone() Automation {
	out := a
	out.Conditions = append([]Filter(nil), a.Conditions...)
	out.Actions = cloneActions(a.Actions)
	return out
}

func NormalizeTags(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			seen[value] = true
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cloneStrings(source map[string]string) map[string]string {
	if source == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func cloneActions(source []Action) []Action {
	out := make([]Action, len(source))
	for index, action := range source {
		action.Params = cloneStrings(action.Params)
		out[index] = action
	}
	return out
}

func timePtr(value time.Time) *time.Time { copy := value; return &copy }
