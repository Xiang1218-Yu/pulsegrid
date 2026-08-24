package scheduler

func MinuteOfDay(hour, minute int) int {
	return hour*60 + minute
}

func WithinWindow(value, start, end int) bool {
	return value >= start && value < end
}

func ClampMinute(value int) int {
	if value < 0 {
		return 0
	}
	if value > 1440 {
		return 1440
	}
	return value
}
