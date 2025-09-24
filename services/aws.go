package services

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type PipelineService struct {
	client           *codepipeline.Client
	codeBuildClient  *codebuild.Client
	cloudWatchClient *cloudwatchlogs.Client
	lambdaClient     *lambda.Client
	region           string
	profile          string
	ssoRoleName      string
	logLineLimit     int32
}

type Pipeline struct {
	Name                string
	Status              string
	Stages              []Stage
	Updated             string
	ExecutionStatus     string
	ExecutionStartTime  string
	ExecutionLastUpdate string
}

type Stage struct {
	Name   string
	Status string
	Order  int32
}

type ActionExecution struct {
	ActionName          string
	StageName           string
	Status              string
	StartTime           string
	EndTime             string
	Duration            string
	LastUpdateTime      string
	ErrorMessage        string
	ExternalExecutionID string
	ActionProvider      string
}

type LogEntry struct {
	Timestamp string
	Message   string
}

func NewPipelineService(ctx context.Context) (*PipelineService, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if cfg.Region == "" {
		return nil, fmt.Errorf("AWS region not configured. Set AWS_REGION environment variable or configure AWS profile")
	}

	// Extract profile and role information from ConfigSources
	var profile, ssoRoleName string

	// Use reflection to access ConfigSources field
	v := reflect.ValueOf(cfg)
	configSourcesField := v.FieldByName("ConfigSources")

	if configSourcesField.IsValid() && configSourcesField.CanInterface() {
		// Iterate through ConfigSources slice to find profile and role info
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

			// Check if this is config.SharedConfig (for RoleARN)
			if actualValue.Type().String() == "config.SharedConfig" {
				roleARNField := actualValue.FieldByName("RoleARN")
				if roleARNField.IsValid() && roleARNField.CanInterface() {
					if roleARNStr, ok := roleARNField.Interface().(string); ok && roleARNStr != "" {
						// Store the full role ARN
						ssoRoleName = roleARNStr
					}
				}
			}
		}
	}

	client := codepipeline.NewFromConfig(cfg)
	codeBuildClient := codebuild.NewFromConfig(cfg)
	cloudWatchClient := cloudwatchlogs.NewFromConfig(cfg)
	lambdaClient := lambda.NewFromConfig(cfg)

	return &PipelineService{
		client:           client,
		codeBuildClient:  codeBuildClient,
		cloudWatchClient: cloudWatchClient,
		lambdaClient:     lambdaClient,
		region:           cfg.Region,
		profile:          profile,
		ssoRoleName:      ssoRoleName,
		logLineLimit:     100, // default value
	}, nil
}

func (ps *PipelineService) GetRegion() string {
	return ps.region
}

func (ps *PipelineService) GetProfile() string {
	return ps.profile
}

func (ps *PipelineService) GetSSORoleName() string {
	return ps.ssoRoleName
}

func (ps *PipelineService) SetLogLineLimit(limit int32) {
	ps.logLineLimit = limit
}

func (ps *PipelineService) GetLatestExecutionStatus(ctx context.Context, pipelineName string) (string, string, string, error) {
	result, err := ps.client.ListPipelineExecutions(ctx, &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(pipelineName),
		MaxResults:   aws.Int32(1), // Only get the latest execution
	})
	if err != nil {
		return "Unknown", "", "", err
	}

	if len(result.PipelineExecutionSummaries) > 0 {
		latest := result.PipelineExecutionSummaries[0]
		status := string(latest.Status)

		var startTime, lastUpdate string
		if latest.StartTime != nil {
			startTime = latest.StartTime.Local().Format("2006-01-02 3:04:05 PM MST")
		}
		if latest.LastUpdateTime != nil {
			lastUpdate = latest.LastUpdateTime.Local().Format("2006-01-02 3:04:05 PM MST")
		}

		return status, startTime, lastUpdate, nil
	}

	return "No executions", "", "", nil
}

func (ps *PipelineService) ListFilteredPipelines(ctx context.Context, filter string) ([]Pipeline, error) {
	var allPipelines []Pipeline
	var nextToken *string

	filterLower := strings.ToLower(filter)

	for {
		input := &codepipeline.ListPipelinesInput{}
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := ps.client.ListPipelines(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, p := range result.Pipelines {
			// Filter on the client side
			if strings.Contains(strings.ToLower(*p.Name), filterLower) {
				pipeline := Pipeline{
					Name:    *p.Name,
					Updated: p.Updated.Format("2006-01-02 15:04:05"),
					Status:  "Unknown",
				}

				// Get latest execution status
				execStatus, startTime, lastUpdate, err := ps.GetLatestExecutionStatus(ctx, *p.Name)
				if err == nil {
					pipeline.ExecutionStatus = execStatus
					pipeline.ExecutionStartTime = startTime
					pipeline.ExecutionLastUpdate = lastUpdate
				} else {
					pipeline.ExecutionStatus = "Error"
				}

				allPipelines = append(allPipelines, pipeline)
			}
		}

		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return allPipelines, nil
}

func (ps *PipelineService) GetPipelineActionExecutions(ctx context.Context, pipelineName string) ([]ActionExecution, error) {
	// First get the latest execution
	execResult, err := ps.client.ListPipelineExecutions(ctx, &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(pipelineName),
		MaxResults:   aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline executions: %w", err)
	}

	if len(execResult.PipelineExecutionSummaries) == 0 {
		return []ActionExecution{}, nil
	}

	executionID := execResult.PipelineExecutionSummaries[0].PipelineExecutionId

	// Get action executions for this execution
	actionResult, err := ps.client.ListActionExecutions(ctx, &codepipeline.ListActionExecutionsInput{
		PipelineName: aws.String(pipelineName),
		Filter: &types.ActionExecutionFilter{
			PipelineExecutionId: executionID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get action executions: %w", err)
	}

	var actions []ActionExecution
	for _, actionExec := range actionResult.ActionExecutionDetails {
		action := ActionExecution{
			ActionName: *actionExec.ActionName,
			StageName:  *actionExec.StageName,
			Status:     string(actionExec.Status),
		}

		if actionExec.StartTime != nil {
			action.StartTime = actionExec.StartTime.Local().Format("2006-01-02 3:04:05 PM MST")
		}
		if actionExec.LastUpdateTime != nil {
			action.LastUpdateTime = actionExec.LastUpdateTime.Local().Format("2006-01-02 3:04:05 PM MST")
		}

		// Calculate duration for completed actions
		if actionExec.StartTime != nil && actionExec.LastUpdateTime != nil && actionExec.Status != types.ActionExecutionStatusInProgress {
			duration := actionExec.LastUpdateTime.Sub(*actionExec.StartTime)
			action.Duration = formatDuration(duration)
		}

		// Get error message if available
		if actionExec.Output != nil && actionExec.Output.ExecutionResult != nil && actionExec.Output.ExecutionResult.ErrorDetails != nil {
			action.ErrorMessage = *actionExec.Output.ExecutionResult.ErrorDetails.Message
		}

		// Get external execution ID if available
		if actionExec.Output != nil && actionExec.Output.ExecutionResult != nil && actionExec.Output.ExecutionResult.ExternalExecutionId != nil {
			action.ExternalExecutionID = *actionExec.Output.ExecutionResult.ExternalExecutionId
		}

		// Get action provider information
		if actionExec.Input != nil && actionExec.Input.ActionTypeId != nil && actionExec.Input.ActionTypeId.Provider != nil {
			action.ActionProvider = *actionExec.Input.ActionTypeId.Provider
		}

		actions = append(actions, action)
	}

	return actions, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

func (ps *PipelineService) GetActionLogs(ctx context.Context, action ActionExecution) ([]LogEntry, error) {
	switch action.ActionProvider {
	case "CodeBuild":
		return ps.getCodeBuildLogs(ctx, action.ExternalExecutionID)
	case "Lambda":
		return ps.getLambdaLogs(ctx, action)
	default:
		return nil, fmt.Errorf("log retrieval not supported for action provider: %s", action.ActionProvider)
	}
}

func (ps *PipelineService) getCodeBuildLogs(ctx context.Context, buildID string) ([]LogEntry, error) {
	if buildID == "" {
		return nil, fmt.Errorf("no build ID available for CodeBuild action")
	}

	builds, err := ps.codeBuildClient.BatchGetBuilds(ctx, &codebuild.BatchGetBuildsInput{
		Ids: []string{buildID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get CodeBuild details: %w", err)
	}

	if len(builds.Builds) == 0 {
		return nil, fmt.Errorf("build not found: %s", buildID)
	}

	build := builds.Builds[0]
	if build.Logs == nil || build.Logs.GroupName == nil {
		return nil, fmt.Errorf("no log group found for build: %s", buildID)
	}

	return ps.getCloudWatchLogs(ctx, *build.Logs.GroupName, build.Logs.StreamName, ps.logLineLimit)
}

func (ps *PipelineService) getLambdaLogs(ctx context.Context, action ActionExecution) ([]LogEntry, error) {
	functionName := action.ActionName
	if functionName == "" {
		return nil, fmt.Errorf("no function name available for Lambda action")
	}

	logGroupName := fmt.Sprintf("/aws/lambda/%s", functionName)

	var startTime *time.Time
	if action.StartTime != "" {
		if t, err := time.Parse("2006-01-02 3:04:05 PM MST", action.StartTime); err == nil {
			startTime = &t
		}
	}

	return ps.getCloudWatchLogsWithTime(ctx, logGroupName, nil, ps.logLineLimit, startTime)
}

func (ps *PipelineService) getCloudWatchLogs(ctx context.Context, logGroupName string, streamName *string, maxLines int32) ([]LogEntry, error) {
	return ps.getCloudWatchLogsWithTime(ctx, logGroupName, streamName, maxLines, nil)
}

func (ps *PipelineService) getCloudWatchLogsWithTime(ctx context.Context, logGroupName string, streamName *string, maxLines int32, startTime *time.Time) ([]LogEntry, error) {
	var logs []LogEntry

	if streamName != nil {
		input := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(logGroupName),
			LogStreamName: streamName,
			Limit:         aws.Int32(maxLines),
		}

		if startTime != nil {
			input.StartTime = aws.Int64(startTime.UnixMilli())
		}

		result, err := ps.cloudWatchClient.GetLogEvents(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get log events: %w", err)
		}

		for _, event := range result.Events {
			timestamp := ""
			if event.Timestamp != nil {
				timestamp = time.UnixMilli(*event.Timestamp).Format("2006-01-02 15:04:05")
			}

			message := ""
			if event.Message != nil {
				message = *event.Message
			}

			logs = append(logs, LogEntry{
				Timestamp: timestamp,
				Message:   message,
			})
		}
	} else {
		streamsInput := &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: aws.String(logGroupName),
			OrderBy:      "LastEventTime",
			Descending:   aws.Bool(true),
			Limit:        aws.Int32(5),
		}

		streamsResult, err := ps.cloudWatchClient.DescribeLogStreams(ctx, streamsInput)
		if err != nil {
			return nil, fmt.Errorf("failed to describe log streams: %w", err)
		}

		if len(streamsResult.LogStreams) == 0 {
			return nil, fmt.Errorf("no log streams found in group: %s", logGroupName)
		}

		for _, stream := range streamsResult.LogStreams {
			if stream.LogStreamName == nil {
				continue
			}

			input := &cloudwatchlogs.GetLogEventsInput{
				LogGroupName:  aws.String(logGroupName),
				LogStreamName: stream.LogStreamName,
				Limit:         aws.Int32(maxLines / int32(len(streamsResult.LogStreams))),
			}

			if startTime != nil {
				input.StartTime = aws.Int64(startTime.UnixMilli())
			}

			result, err := ps.cloudWatchClient.GetLogEvents(ctx, input)
			if err != nil {
				continue
			}

			for _, event := range result.Events {
				timestamp := ""
				if event.Timestamp != nil {
					timestamp = time.UnixMilli(*event.Timestamp).Format("2006-01-02 15:04:05")
				}

				message := ""
				if event.Message != nil {
					message = *event.Message
				}

				logs = append(logs, LogEntry{
					Timestamp: timestamp,
					Message:   message,
				})
			}

			if int32(len(logs)) >= maxLines {
				break
			}
		}
	}

	if len(logs) > int(maxLines) {
		logs = logs[len(logs)-int(maxLines):]
	}

	return logs, nil
}
