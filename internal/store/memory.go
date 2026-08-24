package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"pulsegrid/internal/domain"
)

type Repository interface {
	CreateOrganization(context.Context, domain.Organization) error
	GetOrganization(context.Context, string) (domain.Organization, error)
	UpdateOrganization(context.Context, domain.Organization) error
	ListOrganizations(context.Context, OrganizationFilter) ([]domain.Organization, error)

	CreateContact(context.Context, domain.Contact) error
	GetContact(context.Context, string) (domain.Contact, error)
	UpdateContact(context.Context, domain.Contact) error
	ListContacts(context.Context, ContactFilter) ([]domain.Contact, error)

	CreateAudience(context.Context, domain.Audience) error
	GetAudience(context.Context, string) (domain.Audience, error)
	UpdateAudience(context.Context, domain.Audience) error
	ListAudiences(context.Context, string) ([]domain.Audience, error)

	CreateTemplate(context.Context, domain.Template) error
	GetTemplate(context.Context, string) (domain.Template, error)
	UpdateTemplate(context.Context, domain.Template) error
	ListTemplates(context.Context, string) ([]domain.Template, error)

	CreateCampaign(context.Context, domain.Campaign) error
	GetCampaign(context.Context, string) (domain.Campaign, error)
	UpdateCampaign(context.Context, domain.Campaign) error
	ListCampaigns(context.Context, CampaignFilter) ([]domain.Campaign, error)

	CreateDelivery(context.Context, domain.Delivery) error
	GetDelivery(context.Context, string) (domain.Delivery, error)
	UpdateDelivery(context.Context, domain.Delivery) error
	ListDeliveries(context.Context, DeliveryFilter) ([]domain.Delivery, error)

	CreateAutomation(context.Context, domain.Automation) error
	GetAutomation(context.Context, string) (domain.Automation, error)
	UpdateAutomation(context.Context, domain.Automation) error
	ListAutomations(context.Context, AutomationFilter) ([]domain.Automation, error)
}

type OrganizationFilter struct {
	Status string
	Owner  string
	Search string
	Tag    string
	Limit  int
}

type ContactFilter struct {
	OrganizationID string
	Status         string
	Search         string
	Tag            string
	Limit          int
}

type CampaignFilter struct {
	OrganizationID string
	Status         string
	Search         string
	Limit          int
}

type DeliveryFilter struct {
	CampaignID string
	ContactID  string
	Status     string
	Limit      int
}

type AutomationFilter struct {
	OrganizationID string
	Trigger        string
	Status         string
	Limit          int
}

type Memory struct {
	mu            sync.RWMutex
	organizations map[string]domain.Organization
	contacts      map[string]domain.Contact
	audiences     map[string]domain.Audience
	templates     map[string]domain.Template
	campaigns     map[string]domain.Campaign
	deliveries    map[string]domain.Delivery
	automations   map[string]domain.Automation
}

func NewMemory() *Memory {
	return &Memory{
		organizations: map[string]domain.Organization{},
		contacts:      map[string]domain.Contact{},
		audiences:     map[string]domain.Audience{},
		templates:     map[string]domain.Template{},
		campaigns:     map[string]domain.Campaign{},
		deliveries:    map[string]domain.Delivery{},
		automations:   map[string]domain.Automation{},
	}
}

func (m *Memory) CreateOrganization(ctx context.Context, value domain.Organization) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.organizations[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetOrganization(ctx context.Context, id string) (domain.Organization, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Organization{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.organizations[id]
	if !ok {
		return domain.Organization{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateOrganization(ctx context.Context, value domain.Organization) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.organizations[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListOrganizations(ctx context.Context, filter OrganizationFilter) ([]domain.Organization, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Organization, 0, len(m.organizations))
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, value := range m.organizations {
		if filter.Status != "" && string(value.Status) != filter.Status {
			continue
		}
		if filter.Owner != "" && value.Owner != filter.Owner {
			continue
		}
		if filter.Tag != "" && !has(value.Tags, filter.Tag) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(value.Name+" "+value.Owner), search) {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return capSlice(result, filter.Limit), nil
}

func (m *Memory) CreateContact(ctx context.Context, value domain.Contact) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.OrganizationID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.contacts[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.contacts[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetContact(ctx context.Context, id string) (domain.Contact, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Contact{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.contacts[id]
	if !ok {
		return domain.Contact{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateContact(ctx context.Context, value domain.Contact) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.contacts[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.contacts[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListContacts(ctx context.Context, filter ContactFilter) ([]domain.Contact, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Contact, 0, len(m.contacts))
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, value := range m.contacts {
		if filter.OrganizationID != "" && value.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Status != "" && string(value.Status) != filter.Status {
			continue
		}
		if filter.Tag != "" && !has(value.Tags, filter.Tag) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(value.Email+" "+value.Name+" "+value.Company), search) {
			continue
		}
		contact := value
		contact.Tags = value.Tags
		if contact.Tags == nil {
			contact.Tags = []string{}
		}
		result = append(result, contact)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return capSlice(result, filter.Limit), nil
}

func (m *Memory) CreateAudience(ctx context.Context, value domain.Audience) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.OrganizationID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.audiences[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.audiences[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetAudience(ctx context.Context, id string) (domain.Audience, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Audience{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.audiences[id]
	if !ok {
		return domain.Audience{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateAudience(ctx context.Context, value domain.Audience) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.audiences[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.audiences[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListAudiences(ctx context.Context, organizationID string) ([]domain.Audience, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Audience, 0)
	for _, value := range m.audiences {
		if organizationID != "" && value.OrganizationID != organizationID {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m *Memory) CreateTemplate(ctx context.Context, value domain.Template) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.OrganizationID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.templates[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.templates[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetTemplate(ctx context.Context, id string) (domain.Template, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Template{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.templates[id]
	if !ok {
		return domain.Template{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateTemplate(ctx context.Context, value domain.Template) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.templates[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.templates[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListTemplates(ctx context.Context, organizationID string) ([]domain.Template, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Template, 0)
	for _, value := range m.templates {
		if organizationID != "" && value.OrganizationID != organizationID {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result, nil
}

func (m *Memory) CreateCampaign(ctx context.Context, value domain.Campaign) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.OrganizationID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.audiences[value.AudienceID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.templates[value.TemplateID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.campaigns[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.campaigns[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetCampaign(ctx context.Context, id string) (domain.Campaign, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Campaign{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.campaigns[id]
	if !ok {
		return domain.Campaign{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateCampaign(ctx context.Context, value domain.Campaign) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.campaigns[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.campaigns[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListCampaigns(ctx context.Context, filter CampaignFilter) ([]domain.Campaign, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Campaign, 0)
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	for _, value := range m.campaigns {
		if filter.OrganizationID != "" && value.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Status != "" && string(value.Status) != filter.Status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(value.Name+" "+value.Description), search) {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return capSlice(result, filter.Limit), nil
}

func (m *Memory) CreateDelivery(ctx context.Context, value domain.Delivery) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.campaigns[value.CampaignID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.contacts[value.ContactID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.deliveries[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.deliveries[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetDelivery(ctx context.Context, id string) (domain.Delivery, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Delivery{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.deliveries[id]
	if !ok {
		return domain.Delivery{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateDelivery(ctx context.Context, value domain.Delivery) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deliveries[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.deliveries[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListDeliveries(ctx context.Context, filter DeliveryFilter) ([]domain.Delivery, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Delivery, 0)
	for _, value := range m.deliveries {
		if filter.CampaignID != "" && value.CampaignID != filter.CampaignID {
			continue
		}
		if filter.ContactID != "" && value.ContactID != filter.ContactID {
			continue
		}
		if filter.Status != "" && string(value.Status) != filter.Status {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].QueuedAt.Before(result[j].QueuedAt) })
	return capSlice(result, filter.Limit), nil
}

func (m *Memory) CreateAutomation(ctx context.Context, value domain.Automation) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.organizations[value.OrganizationID]; !ok {
		return domain.ErrNotFound
	}
	if _, ok := m.automations[value.ID]; ok {
		return domain.ErrAlreadyExists
	}
	m.automations[value.ID] = value.Clone()
	return nil
}

func (m *Memory) GetAutomation(ctx context.Context, id string) (domain.Automation, error) {
	if err := contextErr(ctx); err != nil {
		return domain.Automation{}, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.automations[id]
	if !ok {
		return domain.Automation{}, domain.ErrNotFound
	}
	return value.Clone(), nil
}

func (m *Memory) UpdateAutomation(ctx context.Context, value domain.Automation) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := value.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.automations[value.ID]; !ok {
		return domain.ErrNotFound
	}
	m.automations[value.ID] = value.Clone()
	return nil
}

func (m *Memory) ListAutomations(ctx context.Context, filter AutomationFilter) ([]domain.Automation, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]domain.Automation, 0)
	for _, value := range m.automations {
		if filter.OrganizationID != "" && value.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.Trigger != "" && value.Trigger != filter.Trigger {
			continue
		}
		if filter.Status != "" && string(value.Status) != filter.Status {
			continue
		}
		result = append(result, value.Clone())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return capSlice(result, filter.Limit), nil
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func has(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(value) == target {
			return true
		}
	}
	return false
}

func capSlice[T any](values []T, limit int) []T {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func timeNow() time.Time { return time.Now().UTC() }
