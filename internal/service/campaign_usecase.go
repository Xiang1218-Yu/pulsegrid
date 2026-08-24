package service

import "pulsegrid/internal/domain"

func CampaignRates(value domain.Campaign) map[string]float64 {
	rates := map[string]float64{"delivery": 0, "open": 0, "click": 0}
	if value.TargetCount == 0 {
		return rates
	}
	rates["delivery"] = float64(value.DeliveredCount) / float64(value.TargetCount) * 100
	rates["open"] = float64(value.OpenedCount) / float64(value.TargetCount) * 100
	rates["click"] = float64(value.ClickedCount) / float64(value.TargetCount) * 100
	return rates
}

func CampaignNeedsCompletion(value domain.Campaign) bool {
	return value.Status == domain.CampaignRunning && value.TargetCount > 0 && value.DeliveredCount >= value.TargetCount
}
