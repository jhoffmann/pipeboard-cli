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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"

	appTypes "github.com/jhoffmann/pipeboard-cli/internal/types"
)

const (
	// AWS API call timeout
	awsAPITimeout = 30 * time.Second

	// AWS API pagination limits
	defaultMaxPipelineResults = 1
	defaultMaxLogResults      = 500
	defaultMaxLogStreams      = 50

	// Log fetching configuration
	logFetchWindowMinutes = 5

	// CloudWatch log group prefixes
	logGroupCodeBuild  = "/aws/codebuild/"
	logGroupCodeDeploy = "/aws/codedeploy-agent"
	logGroupLambda     = "/aws/lambda/"
)

// logGroupStrategy defines how to determine log group names for different action types
type logGroupStrategy struct {
	extractIdentifier func(string, string) string // extractIdentifier(externalExecutionId, externalExecutionUrl) string
	format            string
	isStatic          bool // true if log group doesn't depend on identifiers
}

// actionTypeStrategies maps action types to their log group determination strategies
var actionTypeStrategies = map[string]logGroupStrategy{
	"Build": {
		extractIdentifier: extractProjectFromBuildID,
		format:            logGroupCodeBuild + "%s",
		isStatic:          false,
	},
	"Test": {
		extractIdentifier: extractProjectFromBuildID,
		format:            logGroupCodeBuild + "%s",
		isStatic:          false,
	},
	"Invoke": {
		extractIdentifier: extractLambdaIdentifier,
		format:            logGroupLambda + "%s",
		isStatic:          false,
	},
	"Deploy": {
		extractIdentifier: nil,
		format:            logGroupCodeDeploy,
		isStatic:          true,
	},
}

// extractProjectFromBuildID extracts the project name from a CodeBuild build ID
func extractProjectFromBuildID(buildID, _ string) string {
	if parts := strings.Split(buildID, ":"); len(parts) >= 1 {
		return parts[0]
	}
	return buildID
}

// extractLambdaIdentifier extracts function name from Lambda execution URL or returns the build ID
func extractLambdaIdentifier(executionID, executionURL string) string {
	if executionURL != "" {
		if functionName := extractLambdaFunctionName(executionURL); functionName != "" {
			return functionName
		}
	}
	return executionID
}

// extractLambdaFunctionName extracts function name from Lambda execution URL
func extractLambdaFunctionName(executionUrl string) string {
	// Lambda console URLs typically contain the function name
	// Example: https://console.aws.amazon.com/lambda/home?region=us-east-1#/functions/my-function
	if strings.Contains(executionUrl, "/functions/") {
		parts := strings.Split(executionUrl, "/functions/")
		if len(parts) > 1 {
			// Extract function name, remove any query parameters
			functionPart := strings.Split(parts[1], "?")[0]
			functionPart = strings.Split(functionPart, "/")[0]
			return functionPart
		}
	}
	return ""
}

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
	client    CodePipelineAPI
	cwlClient *cloudwatchlogs.Client
	filter    string
	logger    *slog.Logger
	region    string
	profile   string
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
		client:    codepipeline.NewFromConfig(cfg),
		cwlClient: cloudwatchlogs.NewFromConfig(cfg),
		filter:    filter,
		logger:    logger,
		region:    cfg.Region,
		profile:   profile,
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

// GetFilter returns the pipeline filter being used.
func (s *CodePipelineService) GetFilter() string {
	return s.filter
}

// ListPipelines retrieves all AWS CodePipelines matching the configured filter.
// Returns pipeline information including status and last execution time.
func (s *CodePipelineService) ListPipelines(ctx context.Context) ([]appTypes.Pipeline, error) {
	ctx, cancel := context.WithTimeout(ctx, awsAPITimeout)
	defer cancel()

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
		MaxResults:   aws.Int32(defaultMaxPipelineResults),
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
	ctx, cancel := context.WithTimeout(ctx, awsAPITimeout)
	defer cancel()

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
		MaxResults:   aws.Int32(defaultMaxPipelineResults),
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

		// Extract external execution details
		var externalExecutionId, externalExecutionUrl, externalExecutionSummary string
		if actionExec.Output != nil && actionExec.Output.ExecutionResult != nil {
			externalExecutionId = aws.ToString(actionExec.Output.ExecutionResult.ExternalExecutionId)
			externalExecutionUrl = aws.ToString(actionExec.Output.ExecutionResult.ExternalExecutionUrl)
			externalExecutionSummary = aws.ToString(actionExec.Output.ExecutionResult.ExternalExecutionSummary)
		}

		actions = append(actions, appTypes.ActionExecution{
			StageName:                aws.ToString(actionExec.StageName),
			ActionName:               aws.ToString(actionExec.ActionName),
			ActionType:               string(actionExec.Input.ActionTypeId.Category),
			Status:                   string(actionExec.Status),
			StartTime:                startTime,
			LastUpdateTime:           lastUpdateTime,
			EndTime:                  endTime,
			ExternalExecutionId:      externalExecutionId,
			ExternalExecutionUrl:     externalExecutionUrl,
			ExternalExecutionSummary: externalExecutionSummary,
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

// GetActionLogs retrieves log entries for a specific action execution from AWS CloudWatch Logs.
func (s *CodePipelineService) GetActionLogs(ctx context.Context, pipelineName, stageName, actionName string) ([]appTypes.LogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, awsAPITimeout)
	defer cancel()

	s.logger.Debug("Getting action logs", "pipeline", pipelineName, "stage", stageName, "action", actionName)

	// Get action details to determine the log source
	actions, err := s.GetPipelineActions(ctx, pipelineName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline actions: %w", err)
	}

	var action *appTypes.ActionExecution
	for _, a := range actions {
		if a.StageName == stageName && a.ActionName == actionName {
			action = &a
			break
		}
	}

	if action == nil {
		return nil, fmt.Errorf("action not found: %s in stage %s", actionName, stageName)
	}

	// Determine log group name based on action type and configuration
	logGroupName, err := s.determineLogGroup(ctx, pipelineName, action)
	if err != nil {
		s.logger.Warn("Could not determine log group", "action", actionName, "error", err)
		return s.getPlaceholderLogs(action), nil
	}

	// Fetch logs from CloudWatch Logs
	logs, err := s.fetchCloudWatchLogs(ctx, logGroupName, action)
	if err != nil {
		s.logger.Warn("Failed to fetch CloudWatch logs", "logGroup", logGroupName, "error", err)
		return s.getPlaceholderLogs(action), nil
	}

	s.logger.Info("Retrieved action logs", "pipeline", pipelineName, "stage", stageName, "action", actionName, "log_count", len(logs), "logGroup", logGroupName)
	return logs, nil
}

// determineLogGroup determines the CloudWatch log group name for an action using external execution details
func (s *CodePipelineService) determineLogGroup(_ context.Context, _ string, action *appTypes.ActionExecution) (string, error) {
	strategy, exists := actionTypeStrategies[action.ActionType]
	if !exists {
		return "", fmt.Errorf("unsupported action type for log retrieval: %s", action.ActionType)
	}

	// For static log groups (like CodeDeploy), return immediately
	if strategy.isStatic {
		return strategy.format, nil
	}

	// Determine identifier for log group name
	identifier := action.ActionName // fallback to action name

	// Try to extract identifier from external execution details if available
	if action.ExternalExecutionId != "" && strategy.extractIdentifier != nil {
		identifier = strategy.extractIdentifier(action.ExternalExecutionId, action.ExternalExecutionUrl)
	}

	return fmt.Sprintf(strategy.format, identifier), nil
}

// findLogStreams finds the appropriate log streams for a given action
func (s *CodePipelineService) findLogStreams(ctx context.Context, logGroupName string, action *appTypes.ActionExecution, startTime, _ time.Time) ([]string, error) {
	// List log streams in the log group within the time range
	input := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroupName),
		OrderBy:      "LastEventTime",
		Descending:   aws.Bool(true),
		Limit:        aws.Int32(defaultMaxLogStreams),
	}

	result, err := s.cwlClient.DescribeLogStreams(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe log streams: %w", err)
	}

	var matchingStreams []string

	for _, stream := range result.LogStreams {
		streamName := aws.ToString(stream.LogStreamName)

		// Check if stream has events in our time range
		if stream.LastEventTimestamp != nil {
			streamTime := time.UnixMilli(aws.ToInt64(stream.LastEventTimestamp))
			if streamTime.Before(startTime) {
				continue // Stream is too old
			}
		}

		// For CodeBuild actions, look for streams that contain the build ID
		if action.ActionType == "Build" && action.ExternalExecutionId != "" {
			buildId := action.ExternalExecutionId
			// Extract just the build number part if it's in the format "project:build-id"
			if parts := strings.Split(buildId, ":"); len(parts) > 1 {
				buildId = parts[1]
			}

			// CodeBuild log streams typically contain the build ID
			if strings.Contains(streamName, buildId) {
				matchingStreams = append(matchingStreams, streamName)
				s.logger.Debug("Found matching log stream", "stream", streamName, "buildId", buildId)
			}
		}
	}

	// If no specific streams found and we have external execution ID,
	// try using the execution ID as the stream name directly
	if len(matchingStreams) == 0 && action.ExternalExecutionId != "" {
		buildId := action.ExternalExecutionId
		if parts := strings.Split(buildId, ":"); len(parts) > 1 {
			buildId = parts[1]
		}
		matchingStreams = []string{buildId}
		s.logger.Debug("Using build ID as stream name", "stream", buildId)
	}

	return matchingStreams, nil
}

// fetchCloudWatchLogs retrieves logs from CloudWatch Logs for the specified log group
func (s *CodePipelineService) fetchCloudWatchLogs(ctx context.Context, logGroupName string, action *appTypes.ActionExecution) ([]appTypes.LogEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, awsAPITimeout)
	defer cancel()

	// Determine the time window for log fetching
	var endTime time.Time
	if !action.EndTime.IsZero() {
		// For completed actions: use completion time
		endTime = action.EndTime
	} else {
		// For running actions: use current time
		endTime = time.Now()
	}

	startTime := endTime.Add(-logFetchWindowMinutes * time.Minute)

	// Find the correct log streams for this action
	logStreamNames, err := s.findLogStreams(ctx, logGroupName, action, startTime, endTime)
	if err != nil {
		s.logger.Warn("Failed to find log streams, fetching all logs", "error", err)
		logStreamNames = nil // Fallback to all streams
	}

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(logGroupName),
		StartTime:    aws.Int64(startTime.UnixMilli()),
		EndTime:      aws.Int64(endTime.UnixMilli()),
		Limit:        aws.Int32(defaultMaxLogResults),
	}

	// If we found specific log streams, use them
	if len(logStreamNames) > 0 {
		input.LogStreamNames = logStreamNames
	}

	result, err := s.cwlClient.FilterLogEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to filter log events: %w", err)
	}

	// Convert CloudWatch log events to our LogEntry format
	var logs []appTypes.LogEntry
	for _, event := range result.Events {
		message := aws.ToString(event.Message)
		// Clean up the message - remove carriage returns, newlines, and extra whitespace
		message = s.cleanLogMessage(message)

		logEntry := appTypes.LogEntry{
			Timestamp:  time.UnixMilli(aws.ToInt64(event.Timestamp)),
			Level:      s.extractLogLevel(message),
			Message:    message,
			Source:     s.extractLogSource(logGroupName),
			ActionName: action.ActionName,
		}
		logs = append(logs, logEntry)
	}

	s.logger.Debug("Fetched CloudWatch logs", "logGroup", logGroupName, "logCount", len(logs), "externalExecutionId", action.ExternalExecutionId)
	return logs, nil
}

// cleanLogMessage cleans up log messages by removing carriage returns, newlines, and extra whitespace
func (s *CodePipelineService) cleanLogMessage(message string) string {
	// Replace carriage returns and newlines with spaces
	message = strings.ReplaceAll(message, "\r\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")

	// Replace multiple spaces with single space
	for strings.Contains(message, "  ") {
		message = strings.ReplaceAll(message, "  ", " ")
	}

	// Trim leading and trailing whitespace
	message = strings.TrimSpace(message)

	return message
}

// extractLogLevel attempts to extract log level from a log message
func (s *CodePipelineService) extractLogLevel(message string) string {
	message = strings.ToUpper(message)

	// Look for common log level indicators
	if strings.Contains(message, "ERROR") || strings.Contains(message, "FATAL") {
		return "ERROR"
	}
	if strings.Contains(message, "WARN") {
		return "WARN"
	}
	if strings.Contains(message, "DEBUG") {
		return "DEBUG"
	}
	if strings.Contains(message, "INFO") {
		return "INFO"
	}

	// Default to INFO if no level found
	return "INFO"
}

// extractLogSource extracts a friendly source name from log group name
func (s *CodePipelineService) extractLogSource(logGroupName string) string {
	switch {
	case strings.Contains(logGroupName, "/aws/codebuild/"):
		return "CodeBuild"
	case strings.Contains(logGroupName, "/aws/codedeploy"):
		return "CodeDeploy"
	case strings.Contains(logGroupName, "/aws/lambda/"):
		return "Lambda"
	default:
		return "CloudWatch"
	}
}

// getPlaceholderLogs returns minimal placeholder logs when real logs can't be fetched
func (s *CodePipelineService) getPlaceholderLogs(action *appTypes.ActionExecution) []appTypes.LogEntry {
	baseTime := time.Now().Add(-10 * time.Minute)
	if !action.StartTime.IsZero() {
		baseTime = action.StartTime
	}

	logs := []appTypes.LogEntry{
		{
			Timestamp:  baseTime,
			Level:      "DEBUG",
			Message:    fmt.Sprintf("No logs available for %s action '%s'", action.ActionType, action.ActionName),
			Source:     "",
			ActionName: action.ActionName,
		},
	}

	// Add helpful information if we have external execution details
	if action.ExternalExecutionId != "" {
		logs = append(logs, appTypes.LogEntry{
			Timestamp:  baseTime.Add(1 * time.Second),
			Level:      "DEBUG",
			Message:    fmt.Sprintf("External execution ID: %s", action.ExternalExecutionId),
			Source:     "",
			ActionName: action.ActionName,
		})
	}

	if action.ExternalExecutionUrl != "" {
		logs = append(logs, appTypes.LogEntry{
			Timestamp:  baseTime.Add(2 * time.Second),
			Level:      "DEBUG",
			Message:    fmt.Sprintf("URL: %s", action.ExternalExecutionUrl),
			Source:     "",
			ActionName: action.ActionName,
		})
	}

	if action.ExternalExecutionSummary != "" {
		// Clean the summary message to remove newlines and carriage returns
		cleanedSummary := s.cleanLogMessage(action.ExternalExecutionSummary)
		logs = append(logs, appTypes.LogEntry{
			Timestamp:  baseTime.Add(3 * time.Second),
			Level:      "DEBUG",
			Message:    fmt.Sprintf("Summary: %s", cleanedSummary),
			Source:     "",
			ActionName: action.ActionName,
		})
	}

	return logs
}
