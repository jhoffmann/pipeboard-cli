package types

import (
	"strings"
	"testing"
	"time"
)

func TestPipelineDescription(t *testing.T) {
	tests := []struct {
		name     string
		pipeline Pipeline
		contains []string
	}{
		{
			name: "pipeline with execution time",
			pipeline: Pipeline{
				Name:              "test-pipeline",
				Status:            "Succeeded",
				LastExecutionTime: time.Now().Add(-30 * time.Minute),
			},
			contains: []string{"Succeeded", "Last execution"},
		},
		{
			name: "pipeline without execution time",
			pipeline: Pipeline{
				Name:              "test-pipeline",
				Status:            "InProgress",
				LastExecutionTime: time.Time{},
			},
			contains: []string{"InProgress"},
		},
		{
			name: "failed pipeline",
			pipeline: Pipeline{
				Name:              "failed-pipeline",
				Status:            "Failed",
				LastExecutionTime: time.Now().Add(-2 * time.Hour),
			},
			contains: []string{"Failed", "Last execution"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.pipeline.Description()
			for _, expected := range tt.contains {
				if !strings.Contains(description, expected) {
					t.Errorf("Pipeline.Description() = %v, should contain %v", description, expected)
				}
			}
		})
	}
}

func TestActionExecutionDescription(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		action   ActionExecution
		contains []string
	}{
		{
			name: "completed action with duration",
			action: ActionExecution{
				StageName:      "Build",
				ActionName:     "BuildAction",
				ActionType:     "Build",
				Status:         "Succeeded",
				StartTime:      now.Add(-10 * time.Minute),
				LastUpdateTime: now.Add(-5 * time.Minute),
				EndTime:        now.Add(-5 * time.Minute),
			},
			contains: []string{"Succeeded", "Build", "Started", "took"},
		},
		{
			name: "in progress action",
			action: ActionExecution{
				StageName:      "Deploy",
				ActionName:     "DeployAction",
				ActionType:     "Deploy",
				Status:         "InProgress",
				StartTime:      now.Add(-2 * time.Minute),
				LastUpdateTime: now.Add(-1 * time.Minute),
				EndTime:        time.Time{},
			},
			contains: []string{"InProgress", "Deploy", "Started", "running"},
		},
		{
			name: "action never started",
			action: ActionExecution{
				StageName:      "Test",
				ActionName:     "TestAction",
				ActionType:     "Test",
				Status:         "Not executed",
				StartTime:      time.Time{},
				LastUpdateTime: time.Time{},
				EndTime:        time.Time{},
			},
			contains: []string{"Not executed", "Test", "Started Never"},
		},
		{
			name: "failed action",
			action: ActionExecution{
				StageName:      "Build",
				ActionName:     "BuildAction",
				ActionType:     "Build",
				Status:         "Failed",
				StartTime:      now.Add(-30 * time.Minute),
				LastUpdateTime: now.Add(-25 * time.Minute),
				EndTime:        now.Add(-25 * time.Minute),
			},
			contains: []string{"Failed", "Build", "Started", "took"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tt.action.Description()
			for _, expected := range tt.contains {
				if !strings.Contains(description, expected) {
					t.Errorf("ActionExecution.Description() = %v, should contain %v", description, expected)
				}
			}
		})
	}
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected string
	}{
		{"succeeded status", "Succeeded", "Succeeded"},
		{"in progress status", "InProgress", "InProgress"},
		{"failed status", "Failed", "Failed"},
		{"stopped status", "Stopped", "Stopped"},
		{"unknown status", "Unknown", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ColorizeStatus(tt.status)
			// Since we can't easily test the exact styling, just ensure the status text is present
			if !strings.Contains(result, tt.status) {
				t.Errorf("ColorizeStatus(%v) = %v, should contain %v", tt.status, result, tt.status)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"just now", 30 * time.Second, "Just now"},
		{"one minute", 1 * time.Minute, "1 minute ago"},
		{"multiple minutes", 5 * time.Minute, "5 minutes ago"},
		{"one hour", 1 * time.Hour, "1 hour ago"},
		{"multiple hours", 3 * time.Hour, "3 hours ago"},
		{"one day", 24 * time.Hour, "1 day ago"},
		{"multiple days", 3 * 24 * time.Hour, "3 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo(%v) = %v, want %v", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"one second", 1 * time.Second, "1s"},
		{"multiple seconds", 30 * time.Second, "30s"},
		{"one minute", 1 * time.Minute, "1m"},
		{"multiple minutes", 5 * time.Minute, "5m"},
		{"one hour", 1 * time.Hour, "1h"},
		{"one hour with minutes", 1*time.Hour + 30*time.Minute, "1h30m"},
		{"multiple hours", 3 * time.Hour, "3h"},
		{"multiple hours with minutes", 2*time.Hour + 45*time.Minute, "2h45m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %v, want %v", tt.duration, result, tt.expected)
			}
		})
	}
}
