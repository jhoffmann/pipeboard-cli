package types

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	successStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("40"))  // Green
	inProgressStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220")) // Yellow
	failedStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")) // Red
)

// Pipeline represents a CodePipeline with its status and execution information
type Pipeline struct {
	Name              string
	Status            string
	LastExecutionTime time.Time
}

// Description returns a formatted description of the pipeline status.
// Includes colorized status and last execution time if available.
func (p Pipeline) Description() string {
	coloredStatus := ColorizeStatus(p.Status)

	if p.LastExecutionTime.IsZero() {
		return fmt.Sprintf("%s", coloredStatus)
	}

	timeAgo := FormatTimeAgo(time.Since(p.LastExecutionTime))
	return fmt.Sprintf("%s | Last execution: %s", coloredStatus, timeAgo)
}

// ActionExecution represents a pipeline action execution with timing and status information
type ActionExecution struct {
	StageName      string
	ActionName     string
	ActionType     string
	Status         string
	StartTime      time.Time
	LastUpdateTime time.Time
	EndTime        time.Time
}

// Description returns a formatted description of the action execution.
// Includes colorized status, action type, timing information, and duration calculations.
func (a ActionExecution) Description() string {
	coloredStatus := ColorizeStatus(a.Status)

	if a.StartTime.IsZero() {
		return fmt.Sprintf("%s | %s | Started Never", coloredStatus, a.ActionType)
	}

	timeAgo := FormatTimeAgo(time.Since(a.StartTime))

	// If we have an end time, calculate and show duration
	if !a.EndTime.IsZero() {
		duration := a.EndTime.Sub(a.StartTime)
		durationStr := FormatDuration(duration)
		return fmt.Sprintf("%s | %s | Started %s (took %s)", coloredStatus, a.ActionType, timeAgo, durationStr)
	}

	// For running actions, show current runtime
	if a.Status == "InProgress" {
		duration := time.Since(a.StartTime)
		durationStr := FormatDuration(duration)
		return fmt.Sprintf("%s | %s | Started %s (running %s)", coloredStatus, a.ActionType, timeAgo, durationStr)
	}

	return fmt.Sprintf("%s | %s | Started %s", coloredStatus, a.ActionType, timeAgo)
}

// ColorizeStatus applies color styling to status strings using lipgloss.
// Succeeded is green, InProgress is yellow, Failed/Stopped is red, others remain unstyled.
func ColorizeStatus(status string) string {
	switch status {
	case "Succeeded":
		return successStyle.Render(status)
	case "InProgress":
		return inProgressStyle.Render(status)
	case "Failed", "Stopped":
		return failedStyle.Render(status)
	default:
		return status
	}
}

// FormatTimeAgo formats a duration as a human-readable "time ago" string.
// Returns formats like "Just now", "5 minutes ago", "2 hours ago", "3 days ago".
func FormatTimeAgo(duration time.Duration) string {
	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// FormatDuration formats a duration as a compact string.
// Returns formats like "30s", "5m", "2h30m" for easy reading in terminal UI.
func FormatDuration(duration time.Duration) string {
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		if seconds == 1 {
			return "1s"
		}
		return fmt.Sprintf("%ds", seconds)
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", minutes)
	} else {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours == 1 && minutes == 0 {
			return "1h"
		} else if minutes == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
}
