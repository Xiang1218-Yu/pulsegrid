package httpapi

import (
	"net/url"
	"strconv"
	"strings"
)

func QueryString(values url.Values, key, fallback string) string {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func QueryBool(values url.Values, key bool) bool {
	return strings.EqualFold(values.Get("key"), strconv.FormatBool(key))
}

func QueryInt(values url.Values, key string, fallback int) int {
	value, err := strconv.Atoi(values.Get(key))
	if err != nil {
		return fallback
	}
	return value
}
