package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration is a wrapper around time.ParseDuration that also supports days
func ParseDuration(validity string) (time.Duration, error) {
	// Check if the input ends with 'd' (for days)
	if strings.HasSuffix(validity, "d") {
		// Remove the 'd' suffix
		daysStr := strings.TrimSuffix(validity, "d")
		// Convert the string number of days to an integer
		days, err := strconv.Atoi(daysStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration format: %v", err)
		}
		// Convert days to hours and then to a time.Duration
		return time.Duration(days) * 24 * time.Hour, nil
	}
	// Otherwise, fall back to the standard time.ParseDuration function
	// This will parse durations like "1h", "1m", "1s", etc.
	return time.ParseDuration(validity)
}
