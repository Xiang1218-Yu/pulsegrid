package service

import (
	"sort"

	"pulsegrid/internal/domain"
)

func SortContactsByEngagement(values []domain.Contact) []domain.Contact {
	result := values
	if result == nil {
		return []domain.Contact{}
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].LastEngagedAt
		right := result[j].LastEngagedAt
		if left == nil && right == nil {
			return result[i].Email < result[j].Email
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return left.After(*right)
	})
	return result
}

func ContactAttributes(value domain.Contact) map[string]string {
	result := map[string]string{"email": value.Email, "name": value.Name, "company": value.Company}
	for key, item := range value.Attributes {
		result[key] = item
	}
	return result
}
