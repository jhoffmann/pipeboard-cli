// Package ui provides the terminal user interface components using Bubble Tea
// for displaying AWS CodePipeline information in an interactive list format.
package ui

import (
	"context"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jhoffmann/pipeboard-cli/internal/types"
)

const (
	// refreshIntervalSeconds defines how often to automatically refresh pipeline data
	refreshIntervalSeconds = 30

	// statusBarReservedLines is the number of vertical lines reserved for the status bar
	statusBarReservedLines = 2
)

var docStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2) // top, right, bottom, left

// refreshMsg is sent periodically to trigger data refresh.
type refreshMsg struct{}

// pipelinesLoadedMsg contains the result of loading pipelines from AWS.
type pipelinesLoadedMsg struct {
	pipelines []types.Pipeline
}

// actionsLoadedMsg contains the result of loading pipeline actions from AWS.
type actionsLoadedMsg struct {
	actions []types.ActionExecution
}

// logsLoadedMsg contains the result of loading action logs from AWS.
type logsLoadedMsg struct {
	logs []types.LogEntry
}

// keyMap defines the key bindings for the application.
type keyMap struct {
	refresh key.Binding
	enter   key.Binding
	back    key.Binding
}

// newKeyMap creates and configures the application key bindings.
func newKeyMap() *keyMap {
	return &keyMap{
		refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		back: key.NewBinding(
			key.WithKeys("backspace"),
			key.WithHelp("bksp", "back"),
		),
	}
}

// Model represents the application state for the Bubble Tea program.
type Model struct {
	list             list.Model
	pipelineService  types.PipelineService
	logger           *slog.Logger
	keys             *keyMap
	currentView      string // "list", "detail", or "logs"
	selectedPipeline types.Pipeline
	selectedAction   types.ActionExecution
	version          string // Application version for status bar
	width            int    // Terminal width for status bar rendering
	height           int    // Terminal height for layout
}

// NewModel creates a new Model with the provided pipeline service and logger.
// Initializes the list component and sets up key bindings.
func NewModel(pipelineService types.PipelineService, logger *slog.Logger, version string) Model {
	items := []list.Item{}
	delegate := list.NewDefaultDelegate()
	keys := newKeyMap()

	l := list.New(items, delegate, 0, 0)
	l.Title = "AWS CodePipelines"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.refresh,
			keys.enter,
			keys.back,
		}
	}

	l.SetSpinner(spinner.Dot)

	return Model{
		list:            l,
		pipelineService: pipelineService,
		logger:          logger,
		keys:            keys,
		currentView:     "list",
		version:         version,
	}
}

// Init initializes the model and returns initial commands.
// Starts pipeline loading and sets up periodic refresh.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadPipelines(),
		m.tickRefresh(),
	)
}

// Update handles incoming messages and updates the model state.
// Routes to appropriate view-specific update functions.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Route to appropriate view-specific update function based on current view
	switch m.currentView {
	case "list":
		return updateListView(msg, m)
	case "detail":
		return updateDetailView(msg, m)
	case "logs":
		return updateLogsView(msg, m)
	default:
		return updateListView(msg, m)
	}
}

// View renders the current model state as a string for display.
func (m Model) View() string {
	// Get AWS context for status bar
	profile := m.pipelineService.GetProfile()
	region := m.pipelineService.GetRegion()
	filter := m.pipelineService.GetFilter()

	// Render status bar
	statusBar := renderStatusBar(m.width, m.version, profile, filter, region)

	// Render main content with docStyle
	mainContent := docStyle.Render(m.list.View())

	// Use lipgloss to create layout with proper spacing
	// The key is to ensure the status bar is at the very bottom
	if m.height > 0 {
		// Calculate the height for main content to push status bar to bottom
		totalHeight := m.height
		statusBarHeight := lipgloss.Height(statusBar)
		gapHeight := 1
		mainContentHeight := totalHeight - statusBarHeight - gapHeight

		// Create a spacer to push status bar to bottom
		spacer := lipgloss.NewStyle().
			Height(mainContentHeight - lipgloss.Height(mainContent)).
			Render("")

		return lipgloss.JoinVertical(lipgloss.Left,
			mainContent,
			spacer,
			statusBar,
		)
	}

	// Fallback for when height is not set
	return lipgloss.JoinVertical(lipgloss.Left, mainContent, statusBar)
}

// loadPipelines returns a command that fetches pipelines from AWS.
func (m Model) loadPipelines() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		m.logger.Debug("Loading pipelines")
		pipelines, err := m.pipelineService.ListPipelines(ctx)
		if err != nil {
			m.logger.Error("Failed to load pipelines", "error", err)
			return pipelinesLoadedMsg{pipelines: []types.Pipeline{}}
		}
		return pipelinesLoadedMsg{pipelines: pipelines}
	}
}

// tickRefresh returns a command that sends refresh messages at regular intervals.
func (m Model) tickRefresh() tea.Cmd {
	return tea.Tick(refreshIntervalSeconds*time.Second, func(time.Time) tea.Msg {
		return refreshMsg{}
	})
}

// loadPipelineActions returns a command that fetches actions for the selected pipeline.
func (m Model) loadPipelineActions() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		m.logger.Debug("Loading pipeline actions", "pipeline", m.selectedPipeline.Name)
		actions, err := m.pipelineService.GetPipelineActions(ctx, m.selectedPipeline.Name)
		if err != nil {
			m.logger.Error("Failed to load pipeline actions", "pipeline", m.selectedPipeline.Name, "error", err)
			return actionsLoadedMsg{actions: []types.ActionExecution{}}
		}
		return actionsLoadedMsg{actions: actions}
	}
}

// loadActionLogs returns a command that fetches logs for the selected action.
func (m Model) loadActionLogs() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		m.logger.Debug("Loading action logs", "pipeline", m.selectedPipeline.Name, "stage", m.selectedAction.StageName, "action", m.selectedAction.ActionName)
		logs, err := m.pipelineService.GetActionLogs(ctx, m.selectedPipeline.Name, m.selectedAction.StageName, m.selectedAction.ActionName)
		if err != nil {
			m.logger.Error("Failed to load action logs", "pipeline", m.selectedPipeline.Name, "stage", m.selectedAction.StageName, "action", m.selectedAction.ActionName, "error", err)
			return logsLoadedMsg{logs: []types.LogEntry{}}
		}
		return logsLoadedMsg{logs: logs}
	}
}
