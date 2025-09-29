package ui

import (
	"context"
	"log/slog"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jhoffmann/pipeboard-cli/internal/types"
)

type mockPipelineService struct {
	pipelines  []types.Pipeline
	actions    []types.ActionExecution
	listErr    error
	actionsErr error
}

func (m *mockPipelineService) ListPipelines(ctx context.Context) ([]types.Pipeline, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.pipelines, nil
}

func (m *mockPipelineService) GetPipelineActions(ctx context.Context, pipelineName string) ([]types.ActionExecution, error) {
	if m.actionsErr != nil {
		return nil, m.actionsErr
	}
	return m.actions, nil
}

func (m *mockPipelineService) GetActionLogs(ctx context.Context, pipelineName, stageName, actionName string) ([]types.LogEntry, error) {
	// Return mock logs for testing
	return []types.LogEntry{
		{
			Timestamp:  time.Now().Add(-5 * time.Minute),
			Level:      "INFO",
			Message:    "Test log message",
			Source:     "TestSource",
			ActionName: actionName,
		},
	}, nil
}

func (m *mockPipelineService) GetRegion() string {
	return "us-east-1"
}

func (m *mockPipelineService) GetProfile() string {
	return "default"
}

func (m *mockPipelineService) GetFilter() string {
	return "test-filter"
}

type mockWriter struct{}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func createTestModel() Model {
	service := &mockPipelineService{
		pipelines: []types.Pipeline{
			{Name: "test-pipeline-1", Status: "Succeeded", LastExecutionTime: time.Now().Add(-time.Hour)},
			{Name: "test-pipeline-2", Status: "InProgress", LastExecutionTime: time.Now().Add(-30 * time.Minute)},
		},
		actions: []types.ActionExecution{
			{StageName: "Source", ActionName: "SourceAction", ActionType: "Source", Status: "Succeeded", StartTime: time.Now().Add(-time.Hour)},
			{StageName: "Build", ActionName: "BuildAction", ActionType: "Build", Status: "InProgress", StartTime: time.Now().Add(-30 * time.Minute)},
		},
	}

	logger := slog.New(slog.NewTextHandler(&mockWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewModel(service, logger, "test-version")
}

func TestNewModel(t *testing.T) {
	model := createTestModel()

	if model.currentView != "list" {
		t.Errorf("NewModel() currentView = %v, want %v", model.currentView, "list")
	}

	if model.list.Title != "AWS CodePipelines" {
		t.Errorf("NewModel() list title = %v, want %v", model.list.Title, "AWS CodePipelines")
	}

	if model.pipelineService == nil {
		t.Error("NewModel() pipelineService should not be nil")
	}

	if model.logger == nil {
		t.Error("NewModel() logger should not be nil")
	}
}

func TestPipelineItem(t *testing.T) {
	pipeline := types.Pipeline{
		Name:              "test-pipeline",
		Status:            "Succeeded",
		LastExecutionTime: time.Now().Add(-time.Hour),
	}

	item := pipelineItem{pipeline: pipeline}

	if item.Title() != "test-pipeline" {
		t.Errorf("pipelineItem.Title() = %v, want %v", item.Title(), "test-pipeline")
	}

	if item.FilterValue() != "test-pipeline" {
		t.Errorf("pipelineItem.FilterValue() = %v, want %v", item.FilterValue(), "test-pipeline")
	}

	description := item.Description()
	if description == "" {
		t.Error("pipelineItem.Description() should not be empty")
	}
}

func TestActionItem(t *testing.T) {
	action := types.ActionExecution{
		StageName:  "Build",
		ActionName: "BuildAction",
		ActionType: "Build",
		Status:     "Succeeded",
		StartTime:  time.Now().Add(-time.Hour),
	}

	item := actionItem{action: action}

	expectedTitle := "Build → BuildAction"
	if item.Title() != expectedTitle {
		t.Errorf("actionItem.Title() = %v, want %v", item.Title(), expectedTitle)
	}

	expectedFilterValue := "Build BuildAction"
	if item.FilterValue() != expectedFilterValue {
		t.Errorf("actionItem.FilterValue() = %v, want %v", item.FilterValue(), expectedFilterValue)
	}

	description := item.Description()
	if description == "" {
		t.Error("actionItem.Description() should not be empty")
	}
}

func TestModelInit(t *testing.T) {
	model := createTestModel()
	cmd := model.Init()

	if cmd == nil {
		t.Error("Model.Init() should return a command")
	}
}

func TestModelView(t *testing.T) {
	model := createTestModel()
	view := model.View()

	if view == "" {
		t.Error("Model.View() should not return empty string")
	}
}

func TestUpdateListView(t *testing.T) {
	model := createTestModel()

	// Test window size message
	windowMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, cmd := model.Update(windowMsg)

	if cmd == nil {
		t.Error("Update with WindowSizeMsg should return a command")
	}

	// Verify model is returned
	if newModel == nil {
		t.Error("Update should return a model")
	}
}

func TestUpdateWithPipelinesLoaded(t *testing.T) {
	model := createTestModel()

	// Test pipelines loaded message
	pipelines := []types.Pipeline{
		{Name: "pipeline-1", Status: "Succeeded"},
		{Name: "pipeline-2", Status: "Failed"},
	}

	msg := pipelinesLoadedMsg{pipelines: pipelines}
	newModel, cmd := model.Update(msg)

	// Check that pipelines were loaded into the list
	m := newModel.(Model)
	if len(m.list.Items()) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(m.list.Items()))
	}

	// Check that spinner is stopped (cmd should be nil)
	if cmd != nil {
		t.Error("Update with pipelinesLoadedMsg should return nil command (spinner stopped)")
	}
}

func TestUpdateWithActionsLoaded(t *testing.T) {
	model := createTestModel()
	model.currentView = "detail"

	// Test actions loaded message
	actions := []types.ActionExecution{
		{StageName: "Source", ActionName: "SourceAction", Status: "Succeeded"},
		{StageName: "Build", ActionName: "BuildAction", Status: "InProgress"},
	}

	msg := actionsLoadedMsg{actions: actions}
	newModel, cmd := model.Update(msg)

	// Check that actions were loaded into the list
	m := newModel.(Model)
	if len(m.list.Items()) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(m.list.Items()))
	}

	// Check that spinner is stopped (cmd should be nil)
	if cmd != nil {
		t.Error("Update with actionsLoadedMsg should return nil command (spinner stopped)")
	}
}

func TestLogItem(t *testing.T) {
	logEntry := types.LogEntry{
		Timestamp:  time.Now(),
		Level:      "INFO",
		Message:    "Test log message",
		Source:     "TestSource",
		ActionName: "TestAction",
	}

	item := logItem{log: logEntry}

	title := item.Title()
	if title == "" {
		t.Error("logItem.Title() should not be empty")
	}

	expectedFilterValue := "INFO Test log message TestSource"
	if item.FilterValue() != expectedFilterValue {
		t.Errorf("logItem.FilterValue() = %v, want %v", item.FilterValue(), expectedFilterValue)
	}

	description := item.Description()
	if description != "" {
		t.Error("logItem.Description() should be empty for single-line display")
	}
}

func TestUpdateWithLogsLoaded(t *testing.T) {
	model := createTestModel()
	model.currentView = "logs"

	// Test logs loaded message
	logs := []types.LogEntry{
		{
			Timestamp:  time.Now(),
			Level:      "INFO",
			Message:    "Test log 1",
			Source:     "TestSource",
			ActionName: "TestAction",
		},
		{
			Timestamp:  time.Now().Add(time.Second),
			Level:      "ERROR",
			Message:    "Test log 2",
			Source:     "TestSource",
			ActionName: "TestAction",
		},
	}

	msg := logsLoadedMsg{logs: logs}
	newModel, cmd := model.Update(msg)

	// Check that logs were loaded into the list
	m := newModel.(Model)
	if len(m.list.Items()) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(m.list.Items()))
	}

	// Check that spinner is stopped (cmd should be nil)
	if cmd != nil {
		t.Error("Update with logsLoadedMsg should return nil command (spinner stopped)")
	}
}

func TestKeyMap(t *testing.T) {
	keys := newKeyMap()

	if keys.refresh.Keys()[0] != "r" {
		t.Errorf("refresh key should be 'r', got %v", keys.refresh.Keys()[0])
	}

	if keys.enter.Keys()[0] != "enter" {
		t.Errorf("enter key should be 'enter', got %v", keys.enter.Keys()[0])
	}

	if keys.back.Keys()[0] != "backspace" {
		t.Errorf("back key should be 'backspace', got %v", keys.back.Keys()[0])
	}
}
