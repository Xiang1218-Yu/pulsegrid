package format

import "time"

func ISODate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func ISOTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func ParseISO(value string) (time.Time, error) {
	if len(value) == len("2006-01-02") {
		return time.Parse("2006-01-02", value)
	}
	return time.Parse(time.RFC3339, value)
}
