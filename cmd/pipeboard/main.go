// Package main provides the command-line interface for pipeboard,
// a terminal user interface for monitoring AWS CodePipeline status.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	awsservice "pipeboard-cli-v2/internal/aws"
	"pipeboard-cli-v2/internal/ui"
)

var (
	version   = "dev"
	logLevel  string
	logFormat string
)

func getStyledLogo() string {
	secondary := lipgloss.NewStyle().Foreground(lipgloss.Color("#9c4dcc"))

	lines := []string{}
	pipeLines := []string{
		"░█▀█░▀█▀░█▀█░█▀▀",
		"░█▀▀░░█░░█▀▀░█▀▀",
		"░▀░░░▀▀▀░▀░░░▀▀▀",
	}
	boardLines := []string{
		"░█▀▄░█▀█░█▀█░█▀▄░█▀▄",
		"░█▀▄░█░█░█▀█░█▀▄░█░█",
		"░▀▀░░▀▀▀░▀░▀░▀░▀░▀▀░",
	}

	for i := range 3 {
		line := pipeLines[i] + secondary.Render(boardLines[i])
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

var rootCmd = &cobra.Command{
	Use:   "pipeboard <filter>",
	Short: "Monitor AWS CodePipelines in a terminal UI",
	Long:  getStyledLogo() + "\n\nA terminal user interface for monitoring AWS CodePipelines",
	Args:  cobra.ExactArgs(1),
	Run:   run,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("pipeboard version %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "error", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (text, json)")
}

// run executes the main application logic, initializing AWS services and starting the TUI.
func run(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Setup structured logging
	logger := setupLogger(logLevel, logFormat)
	logger.Info("Starting pipeboard", "version", version, "filter", args[0])

	filter := args[0] // args[0] is guaranteed to exist due to cobra.ExactArgs(1)

	pipelineService, err := awsservice.NewCodePipelineService(ctx, filter, logger)
	if err != nil {
		logger.Error("Failed to initialize AWS CodePipeline service", "error", err)
		os.Exit(1)
	}

	// Log AWS configuration details
	logger.Info("AWS configuration", "region", pipelineService.GetRegion(), "profile", pipelineService.GetProfile())

	model := ui.NewModel(pipelineService, logger, version)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		logger.Error("Program execution failed", "error", err)
		os.Exit(1)
	}
}

// setupLogger configures and returns a structured logger based on CLI flags.
// Supports debug, info, warn, error levels and text/json formats.
func setupLogger(level, format string) *slog.Logger {
	var handler slog.Handler

	// Parse log level
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// main is the application entry point that executes the root command.
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
