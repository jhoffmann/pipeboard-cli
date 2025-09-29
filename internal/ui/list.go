package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// updateListView handles messages and state updates for the pipeline list view.
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
				// Clear filter when switching to detail view
				m.list.ResetFilter()
				return m, tea.Batch(m.loadPipelineActions(), m.list.StartSpinner())
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		// Reserve space for status bar
		m.list.SetSize(msg.Width-h, msg.Height-v-statusBarReservedLines)
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
		// Ensure default delegate for pipelines view
		m.list.SetDelegate(list.NewDefaultDelegate())
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
