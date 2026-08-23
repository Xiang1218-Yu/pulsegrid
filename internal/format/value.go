package format

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func Percent(value float64, decimals int) string {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return "0%"
	}
	if decimals < 0 {
		decimals = 0
	}
	return strconv.FormatFloat(value, 'f', decimals, 64) + "%"
}

func Number(value float64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	return strconv.FormatFloat(value, 'f', decimals, 64)
}

func Duration(value time.Duration) string {
	if value < time.Second {
		return fmt.Sprintf("%dms", value.Milliseconds())
	}
	if value < time.Minute {
		return fmt.Sprintf("%.1fs", value.Seconds())
	}
	if value < time.Hour {
		return fmt.Sprintf("%.1fm", value.Minutes())
	}
	return fmt.Sprintf("%.1fh", value.Hours())
}

func Bytes(value int64) string {
	if value < 0 {
		return "-" + Bytes(-value)
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(value)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", value, units[unit])
	}
	return fmt.Sprintf("%.2f %s", size, units[unit])
}

func Quote(value string, max int) string {
	value = strings.TrimSpace(value)
	if max > 0 && len([]rune(value)) > max {
		runes := []rune(value)
		value = string(runes[:max-1]) + "…"
	}
	return strconv.Quote(value)
}

func JoinNonEmpty(separator string, values ...string) string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return strings.Join(result, separator)
}
