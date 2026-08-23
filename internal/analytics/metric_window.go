package analytics

import "time"

func DayStart(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func DayEnd(value time.Time) time.Time {
	return DayStart(value).Add(24*time.Hour - time.Nanosecond)
}

func MetricWindow(now time.Time, days int) (time.Time, time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if days < 1 {
		days = 1
	}
	return DayStart(now.AddDate(0, 0, -days+1)), DayEnd(now)
}
