package domain

import "strings"

func ContactCanReceive(value Contact) bool {
	return value.Status == ContactSubscribed && strings.Contains(value.Email, "@")
}

func ContactDisplay(value Contact) string {
	if value.Name != "" {
		return value.Name + " <" + value.Email + ">"
	}
	return value.Email
}
