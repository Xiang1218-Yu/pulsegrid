package batch

func SuccessRate(value Report) float64 {
	if value.Summary.Total == 0 {
		return 0
	}
	return float64(value.Summary.Succeeded) / float64(value.Summary.Total) * 100
}

func HasFailures(value Report) bool {
	return value.Summary.Failed > 0
}

func RetryCount(value Report) int {
	count := 0
	for _, result := range value.Results {
		if !result.Success && result.Attempts > 1 {
			count++
		}
	}
	return count
}
