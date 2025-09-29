package ui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// updateLogsView handles messages and state updates for the action logs view.
// Manages log refresh and navigation back to detail view.
func updateLogsView(msg tea.Msg, m Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Update help keys for logs view
	m.keys.back.SetEnabled(true)
	m.keys.enter.SetEnabled(false)

	m.list.Title = "Logs: " + m.selectedAction.StageName + " → " + m.selectedAction.ActionName

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.refresh):
			return m, tea.Batch(m.loadActionLogs(), m.list.StartSpinner())
		case key.Matches(msg, m.keys.back):
			m.currentView = "detail"
			// Clear filter when switching back to detail view
			m.list.ResetFilter()
			// Restore default delegate when leaving logs view
			m.list.SetDelegate(list.NewDefaultDelegate())
			// Re-enable pagination and status bar when leaving logs view
			m.list.SetShowPagination(true)
			m.list.SetShowStatusBar(true)
			return m, tea.Batch(m.loadPipelineActions(), m.list.StartSpinner())
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		// Reserve space for status bar
		m.list.SetSize(msg.Width-h, msg.Height-v-statusBarReservedLines)
		m.width = msg.Width
		m.height = msg.Height

	case logsLoadedMsg:
		m.list.StopSpinner()
		// Switch to compact delegate for logs view
		m.list.SetDelegate(compactDelegate{})
		// Disable pagination and status bar for logs view to maximize content space
		m.list.SetShowPagination(false)
		m.list.SetShowStatusBar(false)
		items := make([]list.Item, len(msg.logs))
		for i, log := range msg.logs {
			items[i] = logItem{log: log}
		}
		m.list.SetItems(items)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}
