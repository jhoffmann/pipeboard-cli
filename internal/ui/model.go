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

	"pipeboard-cli-v2/internal/types"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2, 0, 2) // top, right, bottom, left

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

// refreshMsg is sent periodically to trigger data refresh.
type refreshMsg struct{}

// pipelinesLoadedMsg contains the result of loading pipelines from AWS.
type pipelinesLoadedMsg struct {
	pipelines []types.Pipeline
}

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

// actionsLoadedMsg contains the result of loading pipeline actions from AWS.
type actionsLoadedMsg struct {
	actions []types.ActionExecution
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
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// Model represents the application state for the Bubble Tea program.
type Model struct {
	list             list.Model
	pipelineService  types.PipelineService
	logger           *slog.Logger
	keys             *keyMap
	currentView      string // "list" or "detail"
	selectedPipeline types.Pipeline
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
	l.SetShowStatusBar(false)
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
	// Route to appropriate update function
	if m.currentView == "list" {
		return updateListView(msg, m)
	}
	return updateDetailView(msg, m)
}

// View renders the current model state as a string for display.
func (m Model) View() string {
	// Get AWS context for status bar
	profile := m.pipelineService.GetProfile()
	region := m.pipelineService.GetRegion()

	// Render status bar
	statusBar := renderStatusBar(m.width, m.version, profile, region)

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

// tickRefresh returns a command that sends refresh messages every 30 seconds.
func (m Model) tickRefresh() tea.Cmd {
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg {
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

// updateListView handles messages when in pipeline list view.
// Manages pipeline selection, refresh, and navigation to detail view.
func updateListView(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update help keys for list view
	m.keys.back.SetEnabled(false)
	m.keys.enter.SetEnabled(true)

	m.list.Title = "AWS CodePipelines"

	// Start spinner if we have no items (initial load)
	if len(m.list.Items()) == 0 {
		cmds = append(cmds, m.list.StartSpinner())
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.refresh):
			return m, tea.Batch(m.loadPipelines(), m.list.StartSpinner())
		case key.Matches(msg, m.keys.enter):
			if len(m.list.Items()) > 0 {
				selected := m.list.SelectedItem().(pipelineItem)
				m.selectedPipeline = selected.pipeline
				m.currentView = "detail"
				return m, tea.Batch(m.loadPipelineActions(), m.list.StartSpinner())
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		// Reserve space for status bar (1 line + 1 for spacing)
		m.list.SetSize(msg.Width-h, msg.Height-v-2)
		m.width = msg.Width
		m.height = msg.Height

	case refreshMsg:
		if m.currentView == "list" {
			return m, tea.Batch(
				m.loadPipelines(),
				m.tickRefresh(),
				m.list.StartSpinner(),
			)
		}

	case pipelinesLoadedMsg:
		m.list.StopSpinner()
		items := make([]list.Item, len(msg.pipelines))
		for i, pipeline := range msg.pipelines {
			items[i] = pipelineItem{pipeline: pipeline}
		}
		m.list.SetItems(items)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// updateDetailView handles messages when in pipeline detail view.
// Manages action refresh and navigation back to list view.
func updateDetailView(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update help keys for detail view
	m.keys.back.SetEnabled(true)
	m.keys.enter.SetEnabled(false)

	m.list.Title = "Actions: " + m.selectedPipeline.Name

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.refresh):
			return m, tea.Batch(m.loadPipelineActions(), m.list.StartSpinner())
		case key.Matches(msg, m.keys.back):
			m.currentView = "list"
			return m, tea.Batch(m.loadPipelines(), m.list.StartSpinner())
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		// Reserve space for status bar (1 line + 1 for spacing)
		m.list.SetSize(msg.Width-h, msg.Height-v-2)
		m.width = msg.Width
		m.height = msg.Height

	case actionsLoadedMsg:
		m.list.StopSpinner()
		items := make([]list.Item, len(msg.actions))
		for i, action := range msg.actions {
			items[i] = actionItem{action: action}
		}
		m.list.SetItems(items)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
