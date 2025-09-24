package ui

import (
	"fmt"
	"time"
)

func formatRelativeTime(timeStr string) string {
	if timeStr == "" {
		return "Never"
	}

	// Parse the time string (format: "2006-01-02 15:04:05")
	parsedTime, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return timeStr // Return original if parsing fails
	}

	now := time.Now()
	duration := now.Sub(parsedTime)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else {
		weeks := int(duration.Hours() / (24 * 7))
		return fmt.Sprintf("%dw ago", weeks)
	}
}
