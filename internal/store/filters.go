package store

import "pulsegrid/internal/domain"

func FilterSubscribed(values []domain.Contact) []domain.Contact {
	result := make([]domain.Contact, 0)
	for _, value := range values {
		if value.Status == domain.ContactSubscribed {
			result = append(result, value)
		}
	}
	return result
}

func FilterCampaignStatus(values []domain.Campaign, status domain.CampaignStatus) []domain.Campaign {
	result := make([]domain.Campaign, 0)
	for _, value := range values {
		if value.Status == status {
			result = append(result, value)
		}
	}
	return result
}
