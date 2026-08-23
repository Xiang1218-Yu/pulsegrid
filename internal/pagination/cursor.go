package pagination

import "strings"

func EncodeCursor(value string) string {
	return strings.TrimSpace(value)
}

func DecodeCursor(value string) string {
	return strings.TrimSpace(value)
}

func HasCursor(value string) bool {
	return DecodeCursor(value) != ""
}
