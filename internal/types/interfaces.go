// Package types defines the core domain types, interfaces, and utility functions
// used throughout the pipeboard application for AWS CodePipeline monitoring.
package types

import (
	"context"
)

// PipelineService defines the interface for pipeline operations
type PipelineService interface {
	// ListPipelines returns all pipelines matching the configured filter
	ListPipelines(ctx context.Context) ([]Pipeline, error)

	// GetPipelineActions returns all actions for a specific pipeline
	GetPipelineActions(ctx context.Context, pipelineName string) ([]ActionExecution, error)

	// GetRegion returns the AWS region being used
	GetRegion() string

	// GetProfile returns the AWS profile being used
	GetProfile() string
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}
