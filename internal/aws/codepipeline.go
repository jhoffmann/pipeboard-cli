// Package aws provides AWS CodePipeline service integration for fetching
// pipeline status and execution details using the AWS SDK v2.
package aws

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"

	appTypes "pipeboard-cli-v2/internal/types"
)

// CodePipelineAPI defines the interface for AWS CodePipeline operations.
// This interface enables dependency injection and testing with mock implementations.
type CodePipelineAPI interface {
	ListPipelines(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error)
	ListPipelineExecutions(ctx context.Context, params *codepipeline.ListPipelineExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelineExecutionsOutput, error)
	GetPipeline(ctx context.Context, params *codepipeline.GetPipelineInput, optFns ...func(*codepipeline.Options)) (*codepipeline.GetPipelineOutput, error)
	ListActionExecutions(ctx context.Context, params *codepipeline.ListActionExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListActionExecutionsOutput, error)
}

// CodePipelineService provides AWS CodePipeline operations with filtering and logging.
type CodePipelineService struct {
	client  CodePipelineAPI
	filter  string
	logger  *slog.Logger
	region  string
	profile string
}

// NewCodePipelineService creates a new CodePipelineService with AWS SDK configuration.
// The filter parameter is used to filter pipeline names, and logger provides structured logging.
func NewCodePipelineService(ctx context.Context, filter string, logger *slog.Logger) (*CodePipelineService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Extract profile information from ConfigSources using reflection
	var profile string
	v := reflect.ValueOf(cfg)
	configSourcesField := v.FieldByName("ConfigSources")

	if configSourcesField.IsValid() && configSourcesField.CanInterface() {
		for i := 0; i < configSourcesField.Len(); i++ {
			source := configSourcesField.Index(i)
			actualValue := reflect.ValueOf(source.Interface())

			// Check if this is config.EnvConfig (for profile)
			if actualValue.Type().String() == "config.EnvConfig" {
				profileField := actualValue.FieldByName("SharedConfigProfile")
				if profileField.IsValid() && profileField.CanInterface() {
					if profileStr, ok := profileField.Interface().(string); ok && profileStr != "" {
						profile = profileStr
					}
				}
			}
		}
	}

	service := &CodePipelineService{
		client:  codepipeline.NewFromConfig(cfg),
		filter:  filter,
		logger:  logger,
		region:  cfg.Region,
		profile: profile,
	}

	return service, nil
}

// GetRegion returns the AWS region being used.
func (s *CodePipelineService) GetRegion() string {
	return s.region
}

// GetProfile returns the AWS profile being used.
func (s *CodePipelineService) GetProfile() string {
	return s.profile
}

// ListPipelines retrieves all AWS CodePipelines matching the configured filter.
// Returns pipeline information including status and last execution time.
func (s *CodePipelineService) ListPipelines(ctx context.Context) ([]appTypes.Pipeline, error) {
	s.logger.Debug("Listing pipelines", "filter", s.filter)

	result, err := s.client.ListPipelines(ctx, &codepipeline.ListPipelinesInput{})
	if err != nil {
		s.logger.Error("Failed to list pipelines", "error", err)
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}

	var pipelines []appTypes.Pipeline
	for _, pipeline := range result.Pipelines {
		pipelineName := aws.ToString(pipeline.Name)

		// Apply filter if provided
		if s.filter != "" && !strings.Contains(pipelineName, s.filter) {
			continue
		}

		status, lastExecTime, err := s.getPipelineExecutionInfo(ctx, pipelineName)
		if err != nil {
			s.logger.Warn("Failed to get pipeline execution info", "pipeline", pipelineName, "error", err)
			status = "Unknown"
			lastExecTime = time.Time{}
		}

		pipelines = append(pipelines, appTypes.Pipeline{
			Name:              pipelineName,
			Status:            status,
			LastExecutionTime: lastExecTime,
		})
	}

	s.logger.Info("Listed pipelines", "count", len(pipelines), "filtered_from", len(result.Pipelines))
	return pipelines, nil
}

// getPipelineExecutionInfo retrieves the latest execution status and start time for a pipeline.
// Returns the status string, start time, and any error encountered.
func (s *CodePipelineService) getPipelineExecutionInfo(ctx context.Context, pipelineName string) (string, time.Time, error) {
	result, err := s.client.ListPipelineExecutions(ctx, &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(pipelineName),
		MaxResults:   aws.Int32(1),
	})
	if err != nil {
		return "", time.Time{}, err
	}

	if len(result.PipelineExecutionSummaries) == 0 {
		return "No executions", time.Time{}, nil
	}

	execution := result.PipelineExecutionSummaries[0]
	status := string(execution.Status)

	var lastExecTime time.Time
	if execution.StartTime != nil {
		lastExecTime = *execution.StartTime
	}

	return status, lastExecTime, nil
}

// GetPipelineActions retrieves all action executions for a specific pipeline.
// If no executions exist, returns actions from the pipeline definition with "Not executed" status.
func (s *CodePipelineService) GetPipelineActions(ctx context.Context, pipelineName string) ([]appTypes.ActionExecution, error) {
	s.logger.Debug("Getting pipeline actions", "pipeline", pipelineName)
	// Get pipeline definition to understand the structure
	pipelineResult, err := s.client.GetPipeline(ctx, &codepipeline.GetPipelineInput{
		Name: aws.String(pipelineName),
	})
	if err != nil {
		s.logger.Error("Failed to get pipeline definition", "pipeline", pipelineName, "error", err)
		return nil, fmt.Errorf("failed to get pipeline definition: %w", err)
	}

	// Get the latest pipeline execution
	execResult, err := s.client.ListPipelineExecutions(ctx, &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(pipelineName),
		MaxResults:   aws.Int32(1),
	})
	if err != nil || len(execResult.PipelineExecutionSummaries) == 0 {
		s.logger.Debug("No pipeline executions found, returning actions from definition", "pipeline", pipelineName)
		// If no executions, return actions from pipeline definition with no execution info
		return s.getActionsFromDefinition(pipelineResult.Pipeline.Stages), nil
	}

	executionId := execResult.PipelineExecutionSummaries[0].PipelineExecutionId

	// Get action execution details
	actionResult, err := s.client.ListActionExecutions(ctx, &codepipeline.ListActionExecutionsInput{
		PipelineName: aws.String(pipelineName),
		Filter: &types.ActionExecutionFilter{
			PipelineExecutionId: executionId,
		},
	})
	if err != nil {
		s.logger.Error("Failed to get action executions", "pipeline", pipelineName, "error", err)
		return nil, fmt.Errorf("failed to get action executions: %w", err)
	}

	var actions []appTypes.ActionExecution
	for _, actionExec := range actionResult.ActionExecutionDetails {
		var startTime time.Time
		if actionExec.StartTime != nil {
			startTime = *actionExec.StartTime
		}

		var lastUpdateTime time.Time
		if actionExec.LastUpdateTime != nil {
			lastUpdateTime = *actionExec.LastUpdateTime
		}

		var endTime time.Time
		// For completed actions, use LastUpdateTime as EndTime
		status := string(actionExec.Status)
		if status == "Succeeded" || status == "Failed" || status == "Stopped" {
			endTime = lastUpdateTime
		}

		actions = append(actions, appTypes.ActionExecution{
			StageName:      aws.ToString(actionExec.StageName),
			ActionName:     aws.ToString(actionExec.ActionName),
			ActionType:     string(actionExec.Input.ActionTypeId.Category),
			Status:         string(actionExec.Status),
			StartTime:      startTime,
			LastUpdateTime: lastUpdateTime,
			EndTime:        endTime,
		})
	}

	s.logger.Info("Retrieved pipeline actions", "pipeline", pipelineName, "action_count", len(actions))
	return actions, nil
}

// getActionsFromDefinition extracts actions from pipeline stage definitions.
// Used when no execution history is available, creating ActionExecution objects with "Not executed" status.
func (s *CodePipelineService) getActionsFromDefinition(stages []types.StageDeclaration) []appTypes.ActionExecution {
	var actions []appTypes.ActionExecution
	for _, stage := range stages {
		for _, action := range stage.Actions {
			actions = append(actions, appTypes.ActionExecution{
				StageName:  aws.ToString(stage.Name),
				ActionName: aws.ToString(action.Name),
				ActionType: string(action.ActionTypeId.Category),
				Status:     "Not executed",
			})
		}
	}
	return actions
}
