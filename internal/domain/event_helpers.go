package domain

import "fmt"

func EventDataString(value Event, key string) string {
	raw, ok := value.Data[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(raw)
}

func EventName(value Event) string {
	if value.Type == "" {
		return "unknown"
	}
	return value.Type
}
