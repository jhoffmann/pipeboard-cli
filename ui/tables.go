package ui

import (
	"context"
	"fmt"

	"pipeboard/services"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (t *TUI) populatePipelinesTable() {
	t.pipelinesTable.Clear()

	headers := []string{"Name", "Last Execution", "Execution Status", "Updated"}
	for col, header := range headers {
		t.pipelinesTable.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetExpansion(1).
			SetAlign(tview.AlignLeft).
			SetSelectable(false))
	}

	for row, pipeline := range t.pipelines {
		t.pipelinesTable.SetCell(row+1, 0, tview.NewTableCell(pipeline.Name).SetTextColor(tcell.ColorWhite))

		// Format relative time for last execution
		relativeTime := formatRelativeTime(pipeline.ExecutionLastUpdate)
		t.pipelinesTable.SetCell(row+1, 1, tview.NewTableCell(relativeTime).SetTextColor(tcell.ColorWhite))

		statusColor := tcell.ColorGray
		switch pipeline.ExecutionStatus {
		case "Succeeded":
			statusColor = tcell.ColorGreen
		case "Failed":
			statusColor = tcell.ColorRed
		case "InProgress":
			statusColor = tcell.ColorYellow
		case "Stopped", "Stopping":
			statusColor = tcell.ColorOrange
		}

		t.pipelinesTable.SetCell(row+1, 2, tview.NewTableCell(pipeline.ExecutionStatus).SetTextColor(statusColor))
		t.pipelinesTable.SetCell(row+1, 3, tview.NewTableCell(pipeline.Updated).SetTextColor(tcell.ColorWhite))
	}
}

func (t *TUI) populateActionsTable(actions []services.ActionExecution) {
	t.currentActions = actions
	t.actionsTable.Clear()

	headers := []string{"Stage", "Action", "Status", "Duration", "Start Time", "Error"}
	for col, header := range headers {
		t.actionsTable.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetExpansion(1).
			SetAlign(tview.AlignLeft).
			SetSelectable(false))
	}

	for row, action := range actions {
		statusColor := tcell.ColorGreen
		switch action.Status {
		case "Failed":
			statusColor = tcell.ColorRed
		case "InProgress":
			statusColor = tcell.ColorYellow
		case "Stopped":
			statusColor = tcell.ColorOrange
		}

		t.actionsTable.SetCell(row+1, 0, tview.NewTableCell(action.StageName).SetTextColor(tcell.ColorWhite))
		t.actionsTable.SetCell(row+1, 1, tview.NewTableCell(action.ActionName).SetTextColor(tcell.ColorWhite))
		t.actionsTable.SetCell(row+1, 2, tview.NewTableCell(action.Status).SetTextColor(statusColor))
		t.actionsTable.SetCell(row+1, 3, tview.NewTableCell(action.Duration).SetTextColor(tcell.ColorWhite))
		t.actionsTable.SetCell(row+1, 4, tview.NewTableCell(action.StartTime).SetTextColor(tcell.ColorWhite))

		errorText := ""
		if action.ErrorMessage != "" {
			errorText = action.ErrorMessage
			if len(errorText) > 50 {
				errorText = errorText[:47] + "..."
			}
		}
		t.actionsTable.SetCell(row+1, 5, tview.NewTableCell(errorText).SetTextColor(tcell.ColorRed))
	}
}

func (t *TUI) showActionsView() {
	if t.currentPipeline == nil {
		return
	}

	t.showingActions = true

	t.actionsTable.SetTitle(fmt.Sprintf("[ Actions: %s ]", t.currentPipeline.Name))

	// Clear and rebuild flex layout for actions view
	t.flex.Clear()
	t.flex.AddItem(t.topGrid, 4, 0, false)
	t.flex.AddItem(t.actionsTable, 0, 1, true)
	t.flex.AddItem(t.bottomPanel, 1, 0, false)

	// Update help text for actions view
	t.updateBottomPanel()

	// Load and display actions
	ctx := context.Background()
	actions, err := t.service.GetPipelineActionExecutions(ctx, t.currentPipeline.Name)
	if err != nil {
		t.actionsTable.Clear()
		t.actionsTable.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("Error loading actions: %v", err)).
			SetTextColor(tcell.ColorRed))
		return
	}

	t.populateActionsTable(actions)
	t.app.SetFocus(t.actionsTable)
}

func (t *TUI) showPipelinesView() {
	t.showingActions = false

	// Clear and rebuild flex layout for pipelines view
	t.flex.Clear()
	t.flex.AddItem(t.topGrid, 4, 0, false)
	t.flex.AddItem(t.pipelinesTable, 0, 1, true)
	t.flex.AddItem(t.bottomPanel, 1, 0, false)

	// Update help text for pipelines view
	t.updateBottomPanel()

	t.app.SetFocus(t.pipelinesTable)
}
