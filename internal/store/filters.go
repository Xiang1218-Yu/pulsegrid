package store

import "pulsegrid/internal/domain"

func FilterSubscribed(values []domain.Contact) []domain.Contact {
	result := values[:0]
	for _, value := range values {
		if value.Status == domain.ContactSubscribed {
			result = append(result, value)
		}
	}
	return result
}

func FilterCampaignStatus(values []domain.Campaign, status domain.CampaignStatus) []domain.Campaign {
	result := values[:0]
	for _, value := range values {
		if value.Status == status {
			result = append(result, value)
		}
	}
	return result
}
