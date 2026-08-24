package codec

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"pulsegrid/internal/domain"
)

type Snapshot struct {
	Version      int                 `json:"version"`
	ExportedAt   time.Time           `json:"exported_at"`
	Organization domain.Organization `json:"organization"`
	Contacts     []domain.Contact    `json:"contacts"`
	Audiences    []domain.Audience   `json:"audiences"`
	Templates    []domain.Template   `json:"templates"`
	Campaigns    []domain.Campaign   `json:"campaigns"`
	Automations  []domain.Automation `json:"automations"`
	Deliveries   []domain.Delivery   `json:"deliveries,omitempty"`
}

type ContactRow struct {
	ID             string
	OrganizationID string
	Email          string
	Name           string
	Company        string
	Status         string
	Locale         string
	Timezone       string
	Tags           string
}

type RenderedMessage struct {
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	Channel string            `json:"channel"`
	Values  map[string]string `json:"values,omitempty"`
}

func NewSnapshot(organization domain.Organization, contacts []domain.Contact, audiences []domain.Audience, templates []domain.Template, campaigns []domain.Campaign, automations []domain.Automation) Snapshot {
	return Snapshot{
		Version:      1,
		ExportedAt:   time.Now().UTC(),
		Organization: organization.Clone(),
		Contacts:     cloneContacts(contacts),
		Audiences:    cloneAudiences(audiences),
		Templates:    cloneTemplates(templates),
		Campaigns:    append([]domain.Campaign(nil), campaigns...),
		Automations:  cloneAutomations(automations),
	}
}

func EncodeSnapshot(snapshot Snapshot) ([]byte, error) {
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}
	if snapshot.ExportedAt.IsZero() {
		snapshot.ExportedAt = time.Now().UTC()
	}
	if err := ValidateSnapshot(snapshot); err != nil {
		return nil, err
	}
	return json.MarshalIndent(snapshot, "", "  ")
}

func DecodeSnapshot(data []byte) (Snapshot, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Snapshot{}, fmt.Errorf("snapshot is empty")
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	if err := ValidateSnapshot(snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func ValidateSnapshot(snapshot Snapshot) error {
	if snapshot.Version != 1 {
		return fmt.Errorf("unsupported snapshot version %d", snapshot.Version)
	}
	if err := snapshot.Organization.Validate(); err != nil {
		return fmt.Errorf("organization: %w", err)
	}
	contactIDs := map[string]bool{}
	for _, contact := range snapshot.Contacts {
		if err := contact.Validate(); err != nil {
			return fmt.Errorf("contact %s: %w", contact.ID, err)
		}
		if contact.OrganizationID != snapshot.Organization.ID {
			return fmt.Errorf("contact %s has wrong organization", contact.ID)
		}
		if contactIDs[contact.ID] {
			return fmt.Errorf("duplicate contact %s", contact.ID)
		}
		contactIDs[contact.ID] = true
	}
	for _, audience := range snapshot.Audiences {
		if err := audience.Validate(); err != nil {
			return fmt.Errorf("audience %s: %w", audience.ID, err)
		}
		if audience.OrganizationID != snapshot.Organization.ID {
			return fmt.Errorf("audience %s has wrong organization", audience.ID)
		}
	}
	for _, template := range snapshot.Templates {
		if err := template.Validate(); err != nil {
			return fmt.Errorf("template %s: %w", template.ID, err)
		}
		if template.OrganizationID != snapshot.Organization.ID {
			return fmt.Errorf("template %s has wrong organization", template.ID)
		}
	}
	for _, campaign := range snapshot.Campaigns {
		if err := campaign.Validate(); err != nil {
			return fmt.Errorf("campaign %s: %w", campaign.ID, err)
		}
		if campaign.OrganizationID != snapshot.Organization.ID {
			return fmt.Errorf("campaign %s has wrong organization", campaign.ID)
		}
	}
	for _, automation := range snapshot.Automations {
		if err := automation.Validate(); err != nil {
			return fmt.Errorf("automation %s: %w", automation.ID, err)
		}
		if automation.OrganizationID != snapshot.Organization.ID {
			return fmt.Errorf("automation %s has wrong organization", automation.ID)
		}
	}
	return nil
}

func ExportContactsCSV(contacts []domain.Contact) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"id", "organization_id", "email", "name", "company", "status", "locale", "timezone", "tags"}); err != nil {
		return nil, err
	}
	for _, contact := range contacts {
		if err := writer.Write([]string{
			contact.ID, contact.OrganizationID, contact.Email, contact.Name, contact.Company,
			string(contact.Status), contact.Locale, contact.Timezone, strings.Join(contact.Tags, "|"),
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buffer.Bytes(), writer.Error()
}

func ImportContactsCSV(data []byte, organizationID string) ([]domain.Contact, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	}
	if !equalHeader(header, []string{"id", "organization_id", "email", "name", "company", "status", "locale", "timezone", "tags"}) {
		return nil, fmt.Errorf("unexpected contact csv header")
	}
	result := make([]domain.Contact, 0)
	for line := 2; ; line++ {
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("csv row %d: %w", line, readErr)
		}
		if len(row) != 9 {
			return nil, fmt.Errorf("csv row %d has %d fields", line, len(row))
		}
		status := domain.ContactStatus(row[5])
		if status == "" {
			status = domain.ContactSubscribed
		}
		contact := domain.Contact{
			ID: row[0], OrganizationID: organizationID, Email: strings.ToLower(strings.TrimSpace(row[2])),
			Name: row[3], Company: row[4], Status: status, Locale: row[6], Timezone: row[7],
			Tags: splitTags(row[8]), Attributes: map[string]string{}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		if contact.Locale == "" {
			contact.Locale = "en-US"
		}
		if contact.Timezone == "" {
			contact.Timezone = "UTC"
		}
		if err := contact.Validate(); err != nil {
			return nil, fmt.Errorf("csv row %d: %w", line, err)
		}
		result = append(result, contact)
	}
	return result, nil
}

func Render(template domain.Template, values map[string]string) (RenderedMessage, error) {
	if err := template.Validate(); err != nil {
		return RenderedMessage{}, err
	}
	subject, err := renderText(template.Subject, values)
	if err != nil {
		return RenderedMessage{}, err
	}
	body, err := renderText(template.Body, values)
	if err != nil {
		return RenderedMessage{}, err
	}
	return RenderedMessage{Subject: subject, Body: body, Channel: template.Channel, Values: copyStrings(values)}, nil
}

func renderText(source string, values map[string]string) (string, error) {
	var output strings.Builder
	for index := 0; index < len(source); {
		open := strings.Index(source[index:], "{{")
		if open < 0 {
			output.WriteString(source[index:])
			break
		}
		open += index
		output.WriteString(source[index:open])
		closeIndex := strings.Index(source[open+2:], "}}")
		if closeIndex < 0 {
			return "", fmt.Errorf("unclosed template variable")
		}
		closeIndex += open + 2
		key := strings.TrimSpace(source[open+2 : closeIndex])
		if key == "" {
			return "", fmt.Errorf("empty template variable")
		}
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("template variable %q is missing", key)
		}
		output.WriteString(value)
		index = closeIndex + 2
	}
	return output.String(), nil
}

func ParseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func ParseFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func equalHeader(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range right {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func splitTags(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return domain.NormalizeTags(strings.Split(value, "|"))
}

func cloneContacts(values []domain.Contact) []domain.Contact {
	out := make([]domain.Contact, len(values))
	for index, value := range values {
		out[index] = value.Clone()
	}
	return out
}

func cloneAudiences(values []domain.Audience) []domain.Audience {
	out := make([]domain.Audience, len(values))
	for index, value := range values {
		out[index] = value.Clone()
	}
	return out
}

func cloneTemplates(values []domain.Template) []domain.Template {
	out := make([]domain.Template, len(values))
	for index, value := range values {
		out[index] = value.Clone()
	}
	return out
}

func cloneAutomations(values []domain.Automation) []domain.Automation {
	out := make([]domain.Automation, len(values))
	for index, value := range values {
		out[index] = value.Clone()
	}
	return out
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
