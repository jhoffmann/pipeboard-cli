package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// updateDetailView handles messages and state updates for the pipeline detail view.
// Manages action selection, refresh, and navigation between list and logs views.
func updateDetailView(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update help keys for detail view
	m.keys.back.SetEnabled(true)
	m.keys.enter.SetEnabled(true)

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
		case key.Matches(msg, m.keys.enter):
			if len(m.list.Items()) > 0 {
				selected := m.list.SelectedItem().(actionItem)
				m.selectedAction = selected.action
				m.currentView = "logs"
				// Clear filter when switching to logs view
				m.list.ResetFilter()
				return m, tea.Batch(m.loadActionLogs(), m.list.StartSpinner())
			}
		case key.Matches(msg, m.keys.back):
			m.currentView = "list"
			// Clear filter when switching back to list view
			m.list.ResetFilter()
			return m, tea.Batch(m.loadPipelines(), m.list.StartSpinner())
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		// Reserve space for status bar
		m.list.SetSize(msg.Width-h, msg.Height-v-statusBarReservedLines)
		m.width = msg.Width
		m.height = msg.Height

	case actionsLoadedMsg:
		m.list.StopSpinner()
		// Ensure default delegate for actions view
		m.list.SetDelegate(list.NewDefaultDelegate())
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
