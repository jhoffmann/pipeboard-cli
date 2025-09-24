package ui

import (
	"context"
	"fmt"

	"pipeboard/services"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type TUI struct {
	app             *tview.Application
	pages           *tview.Pages
	flex            *tview.Flex
	topGrid         *tview.Grid
	filterView      *tview.TextView
	profileView     *tview.TextView
	regionView      *tview.TextView
	roleView        *tview.TextView
	bottomPanel     *tview.TextView
	pipelinesTable  *tview.Table
	actionsTable    *tview.Table
	logViewer       *LogViewer
	service         *services.PipelineService
	pipelines       []services.Pipeline
	currentPipeline *services.Pipeline
	currentActions  []services.ActionExecution
	showingActions  bool
	showingLogs     bool
	filter          string
	region          string
	profile         string
	ssoRoleName     string
}

func NewTUI(service *services.PipelineService) *TUI {
	app := tview.NewApplication()

	tui := &TUI{
		app:         app,
		service:     service,
		region:      service.GetRegion(),
		profile:     service.GetProfile(),
		ssoRoleName: service.GetSSORoleName(),
	}

	tui.logViewer = NewLogViewer()
	tui.setupUI()
	return tui
}

func (t *TUI) setupUI() {
	// Create individual text views for the grid
	t.filterView = tview.NewTextView().SetText("Filter: Loading...")
	t.filterView.SetBackgroundColor(tcell.ColorDefault)

	t.profileView = tview.NewTextView().SetText("Profile: Loading...")
	t.profileView.SetBackgroundColor(tcell.ColorDefault)

	t.regionView = tview.NewTextView().SetText("Region: Loading...")
	t.regionView.SetBackgroundColor(tcell.ColorDefault)

	t.roleView = tview.NewTextView().SetText("Role: Loading...")
	t.roleView.SetBackgroundColor(tcell.ColorDefault)

	// Create the grid
	t.topGrid = tview.NewGrid().
		SetRows(1, 1).
		SetColumns(0, 0).
		SetBorders(false).
		AddItem(t.filterView, 0, 0, 1, 1, 0, 0, false).
		AddItem(t.profileView, 0, 1, 1, 1, 0, 0, false).
		AddItem(t.regionView, 1, 0, 1, 1, 0, 0, false).
		AddItem(t.roleView, 1, 1, 1, 1, 0, 0, false)

	t.topGrid.SetBorder(true).SetTitle("[ AWS Environment ]").SetBackgroundColor(tcell.ColorDefault)

	t.bottomPanel = tview.NewTextView()
	t.bottomPanel.SetTextAlign(tview.AlignCenter).
		SetBackgroundColor(tcell.ColorDefault)
	t.bottomPanel.SetBorder(false)

	t.pipelinesTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	t.pipelinesTable.SetBorder(false).
		SetBorderPadding(1, 1, 2, 2).
		SetBackgroundColor(tcell.ColorDefault)

	t.actionsTable = tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	t.actionsTable.SetBorder(true).
		SetBorderPadding(1, 1, 2, 2).
		SetBackgroundColor(tcell.ColorDefault)

	t.pages = tview.NewPages()

	t.flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.topGrid, 4, 0, false).
		AddItem(t.pipelinesTable, 0, 1, true).
		AddItem(t.bottomPanel, 1, 0, false)
	t.flex.SetBackgroundColor(tcell.ColorDefault)

	t.pages.AddPage("main", t.flex, true, true)
	t.pages.SetBackgroundColor(tcell.ColorDefault)

	// Set initial help text for pipelines view
	t.updateBottomPanel()

	t.setupKeyBindings()
	t.setupTableHandlers()

	t.app.SetRoot(t.pages, true)
}

func (t *TUI) LoadPipelines(ctx context.Context, filter string) error {
	t.filter = filter

	pipelines, err := t.service.ListFilteredPipelines(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to load pipelines: %w", err)
	}

	t.pipelines = pipelines
	t.populatePipelinesTable()
	t.updateTopPanel()
	return nil
}

func (t *TUI) updateTopPanel() {
	// Update each text view directly
	t.filterView.SetText(fmt.Sprintf("Filter: %s", t.filter))
	t.profileView.SetText(fmt.Sprintf("Profile: %s", t.profile))
	t.regionView.SetText(fmt.Sprintf("Region: %s", t.region))
	t.roleView.SetText(fmt.Sprintf("Role: %s", t.ssoRoleName))
}

func (t *TUI) updateBottomPanel() {
	if t.showingLogs {
		// Log viewer: show how to close
		t.bottomPanel.SetText("↑↓: navigate | esc: back | q: quit")
	} else if t.showingActions {
		// Actions view: show log viewing and navigation options
		t.bottomPanel.SetText("enter: logs | r: refresh | esc: back | q: quit")
	} else {
		// Pipelines view: show Filter option, no Esc needed
		t.bottomPanel.SetText("enter: select | r: refresh | f: filter | q: quit")
	}
}

func (t *TUI) Run() error {
	return t.app.Run()
}

func (t *TUI) Stop() {
	t.app.Stop()
}
