package service

import (
	"fmt"
	"time"
)

func StableID(prefix string, sequence uint64) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), sequence)
}

func PrefixID(value, prefix string) string {
	return prefix + "-" + value
}
