package aws

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
)

type mockCodePipelineClient struct {
	listPipelinesFunc          func(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error)
	listPipelineExecutionsFunc func(ctx context.Context, params *codepipeline.ListPipelineExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelineExecutionsOutput, error)
	getPipelineFunc            func(ctx context.Context, params *codepipeline.GetPipelineInput, optFns ...func(*codepipeline.Options)) (*codepipeline.GetPipelineOutput, error)
	listActionExecutionsFunc   func(ctx context.Context, params *codepipeline.ListActionExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListActionExecutionsOutput, error)
}

func (m *mockCodePipelineClient) ListPipelines(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error) {
	if m.listPipelinesFunc != nil {
		return m.listPipelinesFunc(ctx, params, optFns...)
	}
	return &codepipeline.ListPipelinesOutput{}, nil
}

func (m *mockCodePipelineClient) ListPipelineExecutions(ctx context.Context, params *codepipeline.ListPipelineExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelineExecutionsOutput, error) {
	if m.listPipelineExecutionsFunc != nil {
		return m.listPipelineExecutionsFunc(ctx, params, optFns...)
	}
	return &codepipeline.ListPipelineExecutionsOutput{}, nil
}

func (m *mockCodePipelineClient) GetPipeline(ctx context.Context, params *codepipeline.GetPipelineInput, optFns ...func(*codepipeline.Options)) (*codepipeline.GetPipelineOutput, error) {
	if m.getPipelineFunc != nil {
		return m.getPipelineFunc(ctx, params, optFns...)
	}
	return &codepipeline.GetPipelineOutput{}, nil
}

func (m *mockCodePipelineClient) ListActionExecutions(ctx context.Context, params *codepipeline.ListActionExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListActionExecutionsOutput, error) {
	if m.listActionExecutionsFunc != nil {
		return m.listActionExecutionsFunc(ctx, params, optFns...)
	}
	return &codepipeline.ListActionExecutionsOutput{}, nil
}

func createTestService(client CodePipelineAPI) *CodePipelineService {
	logger := slog.New(slog.NewTextHandler(&mockWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
	return &CodePipelineService{
		client: client,
		filter: "",
		logger: logger,
	}
}

type mockWriter struct{}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func TestListPipelines(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		mockResponse  *codepipeline.ListPipelinesOutput
		execResponse  *codepipeline.ListPipelineExecutionsOutput
		expectedCount int
		expectedNames []string
	}{
		{
			name:   "successful pipeline listing",
			filter: "",
			mockResponse: &codepipeline.ListPipelinesOutput{
				Pipelines: []types.PipelineSummary{
					{Name: aws.String("test-pipeline-1")},
					{Name: aws.String("test-pipeline-2")},
				},
			},
			execResponse: &codepipeline.ListPipelineExecutionsOutput{
				PipelineExecutionSummaries: []types.PipelineExecutionSummary{
					{
						Status:    types.PipelineExecutionStatusSucceeded,
						StartTime: aws.Time(time.Now().Add(-time.Hour)),
					},
				},
			},
			expectedCount: 2,
			expectedNames: []string{"test-pipeline-1", "test-pipeline-2"},
		},
		{
			name:   "filtered pipeline listing",
			filter: "test-pipeline",
			mockResponse: &codepipeline.ListPipelinesOutput{
				Pipelines: []types.PipelineSummary{
					{Name: aws.String("test-pipeline-1")},
					{Name: aws.String("prod-pipeline-2")},
				},
			},
			execResponse: &codepipeline.ListPipelineExecutionsOutput{
				PipelineExecutionSummaries: []types.PipelineExecutionSummary{
					{
						Status:    types.PipelineExecutionStatusSucceeded,
						StartTime: aws.Time(time.Now().Add(-time.Hour)),
					},
				},
			},
			expectedCount: 1,
			expectedNames: []string{"test-pipeline-1"},
		},
		{
			name:          "empty pipeline list",
			filter:        "",
			mockResponse:  &codepipeline.ListPipelinesOutput{Pipelines: []types.PipelineSummary{}},
			execResponse:  &codepipeline.ListPipelineExecutionsOutput{},
			expectedCount: 0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockCodePipelineClient{
				listPipelinesFunc: func(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error) {
					return tt.mockResponse, nil
				},
				listPipelineExecutionsFunc: func(ctx context.Context, params *codepipeline.ListPipelineExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelineExecutionsOutput, error) {
					return tt.execResponse, nil
				},
			}

			service := createTestService(mockClient)
			service.filter = tt.filter

			pipelines, err := service.ListPipelines(context.Background())
			if err != nil {
				t.Errorf("ListPipelines() error = %v", err)
				return
			}

			if len(pipelines) != tt.expectedCount {
				t.Errorf("ListPipelines() got %d pipelines, want %d", len(pipelines), tt.expectedCount)
			}

			for i, expectedName := range tt.expectedNames {
				if i >= len(pipelines) || pipelines[i].Name != expectedName {
					t.Errorf("ListPipelines() pipeline %d name = %v, want %v", i, pipelines[i].Name, expectedName)
				}
			}
		})
	}
}

func TestGetPipelineActions(t *testing.T) {
	tests := []struct {
		name                string
		pipelineName        string
		getPipelineResp     *codepipeline.GetPipelineOutput
		execResponse        *codepipeline.ListPipelineExecutionsOutput
		actionResponse      *codepipeline.ListActionExecutionsOutput
		expectedActionCount int
	}{
		{
			name:         "successful action retrieval",
			pipelineName: "test-pipeline",
			getPipelineResp: &codepipeline.GetPipelineOutput{
				Pipeline: &types.PipelineDeclaration{
					Stages: []types.StageDeclaration{
						{
							Name: aws.String("Source"),
							Actions: []types.ActionDeclaration{
								{
									Name: aws.String("SourceAction"),
									ActionTypeId: &types.ActionTypeId{
										Category: types.ActionCategorySource,
									},
								},
							},
						},
					},
				},
			},
			execResponse: &codepipeline.ListPipelineExecutionsOutput{
				PipelineExecutionSummaries: []types.PipelineExecutionSummary{
					{PipelineExecutionId: aws.String("exec-123")},
				},
			},
			actionResponse: &codepipeline.ListActionExecutionsOutput{
				ActionExecutionDetails: []types.ActionExecutionDetail{
					{
						StageName:  aws.String("Source"),
						ActionName: aws.String("SourceAction"),
						Status:     types.ActionExecutionStatusSucceeded,
						StartTime:  aws.Time(time.Now().Add(-time.Hour)),
						Input: &types.ActionExecutionInput{
							ActionTypeId: &types.ActionTypeId{
								Category: types.ActionCategorySource,
							},
						},
					},
				},
			},
			expectedActionCount: 1,
		},
		{
			name:         "no executions",
			pipelineName: "test-pipeline",
			getPipelineResp: &codepipeline.GetPipelineOutput{
				Pipeline: &types.PipelineDeclaration{
					Stages: []types.StageDeclaration{
						{
							Name: aws.String("Source"),
							Actions: []types.ActionDeclaration{
								{
									Name: aws.String("SourceAction"),
									ActionTypeId: &types.ActionTypeId{
										Category: types.ActionCategorySource,
									},
								},
							},
						},
					},
				},
			},
			execResponse: &codepipeline.ListPipelineExecutionsOutput{
				PipelineExecutionSummaries: []types.PipelineExecutionSummary{},
			},
			actionResponse:      &codepipeline.ListActionExecutionsOutput{},
			expectedActionCount: 1, // Should return actions from definition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockCodePipelineClient{
				getPipelineFunc: func(ctx context.Context, params *codepipeline.GetPipelineInput, optFns ...func(*codepipeline.Options)) (*codepipeline.GetPipelineOutput, error) {
					return tt.getPipelineResp, nil
				},
				listPipelineExecutionsFunc: func(ctx context.Context, params *codepipeline.ListPipelineExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelineExecutionsOutput, error) {
					return tt.execResponse, nil
				},
				listActionExecutionsFunc: func(ctx context.Context, params *codepipeline.ListActionExecutionsInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListActionExecutionsOutput, error) {
					return tt.actionResponse, nil
				},
			}

			service := createTestService(mockClient)

			actions, err := service.GetPipelineActions(context.Background(), tt.pipelineName)
			if err != nil {
				t.Errorf("GetPipelineActions() error = %v", err)
				return
			}

			if len(actions) != tt.expectedActionCount {
				t.Errorf("GetPipelineActions() got %d actions, want %d", len(actions), tt.expectedActionCount)
			}
		})
	}
}

func TestGetActionsFromDefinition(t *testing.T) {
	stages := []types.StageDeclaration{
		{
			Name: aws.String("Source"),
			Actions: []types.ActionDeclaration{
				{
					Name: aws.String("SourceAction"),
					ActionTypeId: &types.ActionTypeId{
						Category: types.ActionCategorySource,
					},
				},
			},
		},
		{
			Name: aws.String("Build"),
			Actions: []types.ActionDeclaration{
				{
					Name: aws.String("BuildAction"),
					ActionTypeId: &types.ActionTypeId{
						Category: types.ActionCategoryBuild,
					},
				},
			},
		},
	}

	service := createTestService(&mockCodePipelineClient{})
	actions := service.getActionsFromDefinition(stages)

	if len(actions) != 2 {
		t.Errorf("getActionsFromDefinition() got %d actions, want 2", len(actions))
	}

	expectedActions := []struct {
		stageName  string
		actionName string
		status     string
	}{
		{"Source", "SourceAction", "Not executed"},
		{"Build", "BuildAction", "Not executed"},
	}

	for i, expected := range expectedActions {
		if i >= len(actions) {
			t.Errorf("Missing action at index %d", i)
			continue
		}

		action := actions[i]
		if action.StageName != expected.stageName {
			t.Errorf("Action %d StageName = %v, want %v", i, action.StageName, expected.stageName)
		}
		if action.ActionName != expected.actionName {
			t.Errorf("Action %d ActionName = %v, want %v", i, action.ActionName, expected.actionName)
		}
		if action.Status != expected.status {
			t.Errorf("Action %d Status = %v, want %v", i, action.Status, expected.status)
		}
	}
}
