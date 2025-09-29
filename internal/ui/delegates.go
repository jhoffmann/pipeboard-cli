// Package ui provides delegate types and list item wrappers for the Bubble Tea interface
package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jhoffmann/pipeboard-cli/internal/types"
)

// compactDelegate is a custom list delegate with minimal spacing for logs
type compactDelegate struct{}

func (d compactDelegate) Height() int                             { return 1 }
func (d compactDelegate) Spacing() int                            { return 0 }
func (d compactDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(logItem)
	if !ok {
		return
	}

	str := i.Title()

	// Apply selection styling if this item is selected
	if index == m.Index() {
		str = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Render(str)
	}

	fmt.Fprint(w, str)
}

// pipelineItem wraps a Pipeline for use in the bubble tea list component.
type pipelineItem struct {
	pipeline types.Pipeline
}

// Title returns the pipeline name for list display.
func (i pipelineItem) Title() string { return i.pipeline.Name }

// Description returns the formatted pipeline status description.
func (i pipelineItem) Description() string { return i.pipeline.Description() }

// FilterValue returns the pipeline name for list filtering.
func (i pipelineItem) FilterValue() string { return i.pipeline.Name }

// actionItem wraps an ActionExecution for use in the bubble tea list component.
type actionItem struct {
	action types.ActionExecution
}

// Title returns the stage and action name for list display.
func (i actionItem) Title() string { return i.action.StageName + " → " + i.action.ActionName }

// Description returns the formatted action execution description.
func (i actionItem) Description() string { return i.action.Description() }

// FilterValue returns the combined stage and action name for list filtering.
func (i actionItem) FilterValue() string { return i.action.StageName + " " + i.action.ActionName }

// logItem wraps a LogEntry for use in the bubble tea list component.
type logItem struct {
	log types.LogEntry
}

// Title returns the full log entry formatted for single-line display.
func (i logItem) Title() string {
	return i.log.Description()
}

// Description returns empty string since we want single-line display.
func (i logItem) Description() string { return "" }

// FilterValue returns the log message and level for list filtering.
func (i logItem) FilterValue() string { return i.log.Level + " " + i.log.Message + " " + i.log.Source }
