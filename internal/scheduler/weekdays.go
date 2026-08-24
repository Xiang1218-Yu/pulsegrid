package scheduler

import "time"

func IsWeekend(value time.Time) bool {
	weekday := value.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func WeekdayName(value time.Weekday) string {
	return value.String()
}

func BusinessDays(start time.Time, days int) []time.Time {
	if days < 1 {
		return []time.Time{}
	}
	result := make([]time.Time, 0, days)
	for current := start; len(result) < days; current = current.AddDate(0, 0, 1) {
		if !IsWeekend(current) {
			result = append(result, current)
		}
	}
	return result
}
