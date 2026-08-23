package service

import (
	"sort"

	"pulsegrid/internal/domain"
)

type DeliveryPlan struct {
	CampaignID string   `json:"campaign_id"`
	TemplateID string   `json:"template_id"`
	Contacts   []string `json:"contacts"`
	Count      int      `json:"count"`
}

func BuildDeliveryPlan(campaign domain.Campaign, contacts []domain.Contact) DeliveryPlan {
	ids := make([]string, 0, len(contacts))
	for _, contact := range contacts {
		if domain.ContactCanReceive(contact) {
			ids = append(ids, contact.ID)
		}
	}
	sort.Strings(ids)
	return DeliveryPlan{CampaignID: campaign.ID, TemplateID: campaign.TemplateID, Contacts: ids, Count: len(ids)}
}

func PlanIsEmpty(plan DeliveryPlan) bool {
	return len(plan.Contacts) == 0
}

func ApplyDeliveryProgress(campaign *domain.Campaign, before, after domain.Delivery) {
	if !domain.DeliveryEventShouldCount(before, after) {
		return
	}
	switch after.Status {
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
}
