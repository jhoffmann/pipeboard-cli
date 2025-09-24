package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
	"pipeboard/services"
	"pipeboard/ui"
)

func main() {
	app := &cli.Command{
		Name:  "pipeboard",
		Usage: "Pipeline monitoring and management tool",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "log",
				Aliases: []string{"l"},
				Value:   100,
				Usage:   "Number of log lines to retrieve",
			},
		},
		ArgsUsage: "<filter>",
		Action:    runApp,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func runApp(ctx context.Context, cmd *cli.Command) error {
	logLines := cmd.Int("log")

	if cmd.Args().Len() < 1 {
		return fmt.Errorf("filter argument is required")
	}

	filter := cmd.Args().First()

	// Initialize AWS service
	service, err := services.NewPipelineService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS service: %w", err)
	}

	// Set log line limit
	service.SetLogLineLimit(int32(logLines))

	// Create and run TUI
	tui := ui.NewTUI(service)

	// Load pipelines in the background
	if err := tui.LoadPipelines(ctx, filter); err != nil {
		return fmt.Errorf("failed to load pipelines: %w", err)
	}

	// Run the TUI
	if err := tui.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
