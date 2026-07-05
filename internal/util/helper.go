package util

import (
	"strconv"
	"strings"
)

// ParseStringToFloat64 safely converts a string to float64.
// If the conversion fails or the string is empty, it returns 0.
func ParseStringToFloat64(val string) float64 {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func ParseStringToInt(val string) int {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return parsed
}
