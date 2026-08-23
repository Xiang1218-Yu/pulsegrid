package domain

func CampaignProgress(value Campaign) float64 {
	if value.TargetCount <= 0 {
		return 0
	}
	return float64(value.DeliveredCount) / float64(value.TargetCount) * 100
}

func CampaignIsTerminal(value Campaign) bool {
	return value.Status == CampaignCompleted || value.Status == CampaignCancelled
}
