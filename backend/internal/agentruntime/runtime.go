package agentruntime

import (
	"context"
	"time"

	"superdevstudio/internal/store"
)

type Runtime interface {
	StartRun(ctx context.Context, req StartRunRequest) (store.AgentRun, error)
	GetRunByPipelineRun(ctx context.Context, pipelineRunID string) (store.AgentRun, error)
	Plan(ctx context.Context, req PlanRequest) (PlanResult, error)
	Evaluate(ctx context.Context, req EvaluateRequest) (EvaluateResult, error)
	RecordToolCall(ctx context.Context, req ToolCallRequest) (store.AgentToolCall, error)
	FinishRun(ctx context.Context, runID, currentNode, summary string) error
}

type StartRunRequest struct {
	PipelineRunID string
	ProjectID     string
	ChangeBatchID string
	AgentName     string
	ModeName      string
	CurrentNode   string
}

type PlanRequest struct {
	AgentRunID    string
	ProjectID     string
	PipelineRunID string
	NodeName      string
	Title         string
	Goal          string
	Query         string
	ModeName      string
	MaxEvidence   int
	AllowedTools  []string
	Context       map[string]any
}

type PlanResult struct {
	Step          store.AgentStep       `json:"step"`
	Summary       string                `json:"summary"`
	SuggestedTool string                `json:"suggested_tool"`
	NextAction    string                `json:"next_action"`
	ToolArgs      map[string]any        `json:"tool_args,omitempty"`
	Evidence      []store.AgentEvidence `json:"evidence,omitempty"`
	Raw           string                `json:"raw,omitempty"`
}

type EvaluateRequest struct {
	AgentRunID          string
	NodeName            string
	Title               string
	Goal                string
	TaskTitle           string
	Attempt             int
	QualitySummary      string
	DecisionContext     map[string]any
	AllowedNextCommands []string
}

type EvaluateResult struct {
	Step        store.AgentStep       `json:"step"`
	Verdict     string                `json:"verdict"`
	Reason      string                `json:"reason"`
	NextAction  string                `json:"next_action"`
	NextCommand string                `json:"next_command"`
	Evaluation  store.AgentEvaluation `json:"evaluation"`
	Raw         string                `json:"raw,omitempty"`
}

type ToolCallRequest struct {
	AgentStepID string
	ToolName    string
	Request     map[string]any
	Response    map[string]any
	Success     bool
	Latency     time.Duration
}
