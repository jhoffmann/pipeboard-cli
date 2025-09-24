package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LogViewer struct {
	table *tview.Table
}

func NewLogViewer() *LogViewer {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	table.SetBorder(true).
		SetBorderPadding(1, 1, 2, 2).
		SetBackgroundColor(tcell.ColorDefault)

	return &LogViewer{table: table}
}

func (lv *LogViewer) GetTable() *tview.Table {
	return lv.table
}
