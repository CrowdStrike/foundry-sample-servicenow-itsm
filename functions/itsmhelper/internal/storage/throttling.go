package storage

import (
	"fmt"
	"time"
)

// timeNow is a variable that can be replaced in tests
var timeNow = time.Now

// calculateTimeBucket generates a time bucket string based on current time and bucket type
func calculateTimeBucket(tb TimeBucket) (string, error) {
	// Validate bucket type first
	switch tb {
	case TimeBucketForever, TimeBucketFiveMin, TimeBucketThirtyMin:
		// Valid bucket type
	default:
		return "", fmt.Errorf("invalid time bucket: %s", tb)
	}

	// For "forever" type, no time division is needed
	if tb == TimeBucketForever {
		return "forever_bucket", nil
	}

	// Get current time
	now := timeNow().UTC()

	// Format the date part (same for all time-based buckets)
	datePart := now.Format("2006-01-02")

	// Calculate hour and rounded minutes based on bucket type
	hour := now.Hour()
	var minutePart string

	switch tb {
	case TimeBucketFiveMin:
		// Round down to the nearest 5-minute interval
		minuteInterval := (now.Minute() / 5) * 5
		minutePart = fmt.Sprintf("%02d:%02d", hour, minuteInterval)

	case TimeBucketThirtyMin:
		// Round down to the nearest 30-minute interval
		minuteInterval := (now.Minute() / 30) * 30
		minutePart = fmt.Sprintf("%02d:%02d", hour, minuteInterval)
	}

	// Combine into final bucket string
	return fmt.Sprintf("%s_%s", datePart, minutePart), nil
}
