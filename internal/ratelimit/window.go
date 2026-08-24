package ratelimit

import "time"

func MinuteWindow() time.Duration {
	return time.Minute
}

func HourWindow() time.Duration {
	return time.Hour
}

func DayWindow() time.Duration {
	return 24 * time.Hour
}
