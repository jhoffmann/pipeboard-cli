package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (t *TUI) showFilterModal() {
	// Create input field with current filter
	form := tview.NewForm()
	form.AddInputField("Filter Text", t.filter, 40, nil, nil)

	form.AddButton("Apply", func() {
		// Get the new filter value
		newFilter := form.GetFormItemByLabel("Filter Text").(*tview.InputField).GetText()
		t.applyNewFilter(newFilter)
		t.pages.RemovePage("filter-modal")
		t.app.SetFocus(t.pipelinesTable)
	})

	form.AddButton("Clear", func() {
		t.applyNewFilter("")
		t.pages.RemovePage("filter-modal")
		t.app.SetFocus(t.pipelinesTable)
	})

	form.AddButton("Cancel", func() {
		t.pages.RemovePage("filter-modal")
		t.app.SetFocus(t.pipelinesTable)
	})

	// Style the form
	form.SetBorder(true).SetTitle("[ Pipeline Filter ]")
	form.SetBackgroundColor(tcell.ColorDefault)
	form.SetFieldBackgroundColor(tcell.ColorBlack)

	// Handle Escape key to close modal
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			t.pages.RemovePage("filter-modal")
			t.app.SetFocus(t.pipelinesTable)
			return nil
		}
		// Let the form handle all other keys normally
		return event
	})

	// Center the modal
	flex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 9, 1, true).
			AddItem(nil, 0, 1, false), 60, 1, true).
		AddItem(nil, 0, 1, false)

	t.pages.AddPage("filter-modal", flex, true, true)
	t.app.SetFocus(form)
}

func (t *TUI) applyNewFilter(newFilter string) {
	ctx := context.Background()

	// Update the filter
	t.filter = newFilter

	// Fetch new pipelines with the new filter
	pipelines, err := t.service.ListFilteredPipelines(ctx, t.filter)
	if err != nil {
		// Show error in the table
		t.pipelinesTable.Clear()
		t.pipelinesTable.SetCell(0, 0, tview.NewTableCell("Error loading pipelines: "+err.Error()).
			SetTextColor(tcell.ColorRed))
		return
	}

	// Update pipelines and refresh the display
	t.pipelines = pipelines
	t.populatePipelinesTable()
	t.updateTopPanel()
}
