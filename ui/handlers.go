package ui

import (
	"context"
	"fmt"
	"strings"

	"pipeboard/services"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (t *TUI) setupKeyBindings() {
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check if a modal is currently showing
		if t.pages.HasPage("filter-modal") {
			// Let the modal handle its own keys
			return event
		}

		switch event.Key() {
		case tcell.KeyEscape:
			if t.showingLogs {
				t.hideLogViewer()
				return nil
			} else if t.showingActions {
				t.showPipelinesView()
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				t.app.Stop()
				return nil
			case 'r':
				t.refresh()
				return nil
			case 'f':
				// Only allow filter modal in pipelines view
				if !t.showingActions && !t.showingLogs {
					t.showFilterModal()
					return nil
				}

			}
		}
		return event
	})
}

func (t *TUI) setupTableHandlers() {
	t.pipelinesTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(t.pipelines) {
			t.currentPipeline = &t.pipelines[row-1]
			t.showActionsView()
		}
	})

	t.actionsTable.SetSelectedFunc(func(row, column int) {
		if row > 0 && row-1 < len(t.currentActions) {
			action := t.currentActions[row-1]
			t.showLogViewer(action)
		}
	})

	// Set up log viewer keybindings
	t.logViewer.GetTable().SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			t.hideLogViewer()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				t.hideLogViewer()
				return nil
			}
		}
		return event
	})
}

func (t *TUI) showLogViewer(action services.ActionExecution) {
	// Same pattern as showActionsView - do everything here
	t.showingLogs = true

	// Update layout immediately
	t.flex.Clear()
	t.flex.AddItem(t.topGrid, 4, 0, false)
	t.flex.AddItem(t.logViewer.GetTable(), 0, 1, true)
	t.flex.AddItem(t.bottomPanel, 1, 0, false)

	// Update help text
	t.updateBottomPanel()

	// Set table title and show loading
	logTable := t.logViewer.GetTable()
	title := fmt.Sprintf("[ %s - %s: Loading logs... ]", action.StageName, action.ActionName)
	logTable.SetTitle(title)
	logTable.Clear()
	logTable.SetCell(0, 0, tview.NewTableCell("Loading logs...").
		SetTextColor(tcell.ColorYellow).
		SetAlign(tview.AlignCenter).
		SetExpansion(1))

	// Load and display logs (same pattern as actions)
	ctx := context.Background()
	logs, err := t.service.GetActionLogs(ctx, action)
	if err != nil {
		logTable.Clear()
		logTable.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error loading logs: %v", err)).
			SetTextColor(tcell.ColorRed))
		t.app.SetFocus(logTable)
		return
	}

	// Populate table with logs
	t.populateLogTable(logs, action)
	t.app.SetFocus(logTable)
}

func (t *TUI) populateLogTable(logs []services.LogEntry, action services.ActionExecution) {
	table := t.logViewer.GetTable()
	table.Clear()

	headers := []string{"Time", "Message"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetSelectable(false).
			SetAlign(tview.AlignLeft)
		if col == 1 {
			cell.SetExpansion(1)
		}
		table.SetCell(0, col, cell)
	}

	if len(logs) == 0 {
		table.SetCell(1, 0, tview.NewTableCell("No logs found").
			SetTextColor(tcell.ColorGray).
			SetAlign(tview.AlignCenter).
			SetExpansion(2))
		table.SetTitle("[ No Logs ]")
		return
	}

	for row, log := range logs {
		table.SetCell(row+1, 0, tview.NewTableCell(log.Timestamp).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft))

		message := strings.ReplaceAll(log.Message, "\n", " ")
		if len(message) > 120 {
			message = message[:117] + "..."
		}

		table.SetCell(row+1, 1, tview.NewTableCell(message).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignLeft).
			SetExpansion(1))
	}

	logCount := len(logs)
	title := fmt.Sprintf("Logs (%d) - ESC to close", logCount)
	table.SetTitle(fmt.Sprintf("[ %s - %s: %s ]", action.StageName, action.ActionName, title))

	if len(logs) > 0 {
		table.Select(len(logs), 0)
	}
}

func (t *TUI) hideLogViewer() {
	t.showingLogs = false
	t.showActionsView()
}

func (t *TUI) refresh() {
	ctx := context.Background()

	if t.showingActions && t.currentPipeline != nil {
		// Refresh actions view
		actions, err := t.service.GetPipelineActionExecutions(ctx, t.currentPipeline.Name)
		if err != nil {
			t.actionsTable.Clear()
			t.actionsTable.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error refreshing actions: %v", err)).
				SetTextColor(tcell.ColorRed))
			return
		}

		t.populateActionsTable(actions)
	} else {
		// Refresh pipelines view
		pipelines, err := t.service.ListFilteredPipelines(ctx, t.filter)
		if err != nil {
			t.pipelinesTable.Clear()
			t.pipelinesTable.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error refreshing pipelines: %v", err)).
				SetTextColor(tcell.ColorRed))
			return
		}

		t.pipelines = pipelines
		t.populatePipelinesTable()
		t.updateTopPanel()
	}
}
