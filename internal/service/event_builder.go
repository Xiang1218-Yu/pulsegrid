package service

import (
	"time"

	"pulsegrid/internal/domain"
)

func BuildContactEvent(contact domain.Contact, eventType string) domain.Event {
	return domain.Event{
		ID:             "event-" + contact.ID + "-" + time.Now().UTC().Format("20060102150405"),
		OrganizationID: contact.OrganizationID,
		Type:           eventType,
		Subject:        contact.ID,
		ContactID:      contact.ID,
		OccurredAt:     time.Now().UTC(),
		Data:           map[string]any{"email": contact.Email, "name": contact.Name},
	}
}

func BuildCampaignEvent(campaign domain.Campaign, eventType string) domain.Event {
	return domain.Event{
		ID:             "event-" + campaign.ID + "-" + time.Now().UTC().Format("20060102150405"),
		OrganizationID: campaign.OrganizationID,
		Type:           eventType,
		Subject:        campaign.ID,
		OccurredAt:     time.Now().UTC(),
		Data:           map[string]any{"name": campaign.Name, "status": campaign.Status},
	}
}
