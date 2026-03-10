//go:build windows && amd64 && go1.24

package eino

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"superdevstudio/internal/agentruntime"
	"superdevstudio/internal/retrieval"
	"superdevstudio/internal/store"
)

type Config struct {
	APIKey  string
	Model   string
	BaseURL string
}

type Runtime struct {
	store     *store.Store
	retrieval *retrieval.Service
}

type planPayload struct {
	Summary       string         `json:"summary"`
	SuggestedTool string         `json:"suggested_tool"`
	NextAction    string         `json:"next_action"`
	ToolArgs      map[string]any `json:"tool_args"`
}

type evaluationPayload struct {
	Verdict         string   `json:"verdict"`
	Reason          string   `json:"reason"`
	NextAction      string   `json:"next_action"`
	MissingItems    []string `json:"missing_items"`
	AcceptanceDelta string   `json:"acceptance_delta"`
}

func New(_ context.Context, s *store.Store, retrievalService *retrieval.Service, _ Config) (*Runtime, error) {
	return &Runtime{store: s, retrieval: retrievalService}, nil
}

func (r *Runtime) StartRun(ctx context.Context, req agentruntime.StartRunRequest) (store.AgentRun, error) {
	return r.store.CreateAgentRun(ctx, store.AgentRun{
		PipelineRunID: req.PipelineRunID,
		ProjectID:     req.ProjectID,
		ChangeBatchID: req.ChangeBatchID,
		AgentName:     firstNonEmpty(req.AgentName, "delivery-agent"),
		ModeName:      firstNonEmpty(req.ModeName, "step_by_step"),
		Status:        "running",
		CurrentNode:   firstNonEmpty(req.CurrentNode, "bootstrap"),
	})
}

func (r *Runtime) GetRunByPipelineRun(ctx context.Context, pipelineRunID string) (store.AgentRun, error) {
	return r.store.GetAgentRunByPipelineRun(ctx, pipelineRunID)
}

func (r *Runtime) FinishRun(ctx context.Context, runID, currentNode, summary string) error {
	finished := time.Now().UTC()
	return r.store.UpdateAgentRun(ctx, runID, "completed", currentNode, summary, &finished)
}

func (r *Runtime) Plan(ctx context.Context, req agentruntime.PlanRequest) (agentruntime.PlanResult, error) {
	inputJSON := mustJSON(map[string]any{
		"goal":          req.Goal,
		"query":         req.Query,
		"mode":          req.ModeName,
		"allowed_tools": req.AllowedTools,
		"context":       req.Context,
	})
	step, err := r.store.CreateAgentStep(ctx, store.AgentStep{
		AgentRunID: req.AgentRunID,
		NodeName:   strings.TrimSpace(req.NodeName),
		Title:      firstNonEmpty(req.Title, "Agent planning"),
		InputJSON:  inputJSON,
		Status:     "running",
	})
	if err != nil {
		return agentruntime.PlanResult{}, err
	}

	persistedEvidence := make([]store.AgentEvidence, 0, req.MaxEvidence)
	if r.retrieval != nil {
		evidenceItems, retrieveErr := r.retrieval.Retrieve(ctx, retrieval.Request{ProjectID: req.ProjectID, Query: firstNonEmpty(req.Query, req.Goal), MaxItems: req.MaxEvidence})
		if retrieveErr == nil {
			for _, item := range evidenceItems {
				record, createErr := r.store.CreateAgentEvidence(ctx, store.AgentEvidence{
					AgentStepID:  step.ID,
					SourceType:   item.SourceType,
					SourceID:     item.SourceID,
					Title:        item.Title,
					Snippet:      item.Snippet,
					Score:        item.Score,
					MetadataJSON: retrieval.EncodeMetadata(item.Metadata),
				})
				if createErr == nil {
					persistedEvidence = append(persistedEvidence, record)
				}
			}
		}
	}

	plan := fallbackPlan(req, persistedEvidence)
	outputJSON := mustJSON(map[string]any{
		"summary":        plan.Summary,
		"suggested_tool": plan.SuggestedTool,
		"next_action":    plan.NextAction,
		"tool_args":      plan.ToolArgs,
		"raw":            plan.Summary,
	})
	finished := time.Now().UTC()
	if err := r.store.UpdateAgentStep(ctx, step.ID, "completed", outputJSON, plan.Summary, &finished); err != nil {
		return agentruntime.PlanResult{}, err
	}
	step.Status = "completed"
	step.OutputJSON = outputJSON
	step.DecisionSummary = plan.Summary
	step.FinishedAt = &finished
	_ = r.store.UpdateAgentRun(ctx, req.AgentRunID, "running", req.NodeName, plan.Summary, nil)
	return agentruntime.PlanResult{Step: step, Summary: plan.Summary, SuggestedTool: plan.SuggestedTool, NextAction: plan.NextAction, ToolArgs: plan.ToolArgs, Evidence: persistedEvidence, Raw: plan.Summary}, nil
}

func (r *Runtime) Evaluate(ctx context.Context, req agentruntime.EvaluateRequest) (agentruntime.EvaluateResult, error) {
	inputJSON := mustJSON(map[string]any{
		"goal":             req.Goal,
		"task_title":       req.TaskTitle,
		"attempt":          req.Attempt,
		"quality_summary":  req.QualitySummary,
		"decision_context": req.DecisionContext,
	})
	step, err := r.store.CreateAgentStep(ctx, store.AgentStep{
		AgentRunID: req.AgentRunID,
		NodeName:   strings.TrimSpace(req.NodeName),
		Title:      firstNonEmpty(req.Title, "Agent evaluation"),
		InputJSON:  inputJSON,
		Status:     "running",
	})
	if err != nil {
		return agentruntime.EvaluateResult{}, err
	}

	evaluation := fallbackEvaluation(req)
	record, err := r.store.CreateAgentEvaluation(ctx, store.AgentEvaluation{
		AgentStepID:     step.ID,
		EvaluationType:  "step-outcome",
		Verdict:         evaluation.Verdict,
		Reason:          evaluation.Reason,
		NextAction:      evaluation.NextAction,
		MissingItems:    evaluation.MissingItems,
		AcceptanceDelta: evaluation.AcceptanceDelta,
	})
	if err != nil {
		return agentruntime.EvaluateResult{}, err
	}
	outputJSON := mustJSON(map[string]any{
		"verdict":          evaluation.Verdict,
		"reason":           evaluation.Reason,
		"next_action":      evaluation.NextAction,
		"missing_items":    evaluation.MissingItems,
		"acceptance_delta": evaluation.AcceptanceDelta,
		"raw":              evaluation.Reason,
	})
	finished := time.Now().UTC()
	if err := r.store.UpdateAgentStep(ctx, step.ID, "completed", outputJSON, evaluation.Reason, &finished); err != nil {
		return agentruntime.EvaluateResult{}, err
	}
	step.Status = "completed"
	step.OutputJSON = outputJSON
	step.DecisionSummary = evaluation.Reason
	step.FinishedAt = &finished
	_ = r.store.UpdateAgentRun(ctx, req.AgentRunID, "running", req.NodeName, evaluation.Reason, nil)
	return agentruntime.EvaluateResult{Step: step, Verdict: evaluation.Verdict, Reason: evaluation.Reason, NextAction: evaluation.NextAction, Evaluation: record, Raw: evaluation.Reason}, nil
}

func (r *Runtime) RecordToolCall(ctx context.Context, req agentruntime.ToolCallRequest) (store.AgentToolCall, error) {
	return r.store.CreateAgentToolCall(ctx, store.AgentToolCall{
		AgentStepID:  req.AgentStepID,
		ToolName:     req.ToolName,
		RequestJSON:  mustJSON(req.Request),
		ResponseJSON: mustJSON(req.Response),
		Success:      req.Success,
		LatencyMS:    int(req.Latency / time.Millisecond),
	})
}

func fallbackPlan(req agentruntime.PlanRequest, evidence []store.AgentEvidence) planPayload {
	suggestedTool := "run_superdev_task_status"
	if len(req.AllowedTools) > 0 {
		suggestedTool = req.AllowedTools[0]
	}
	if len(evidence) == 0 {
		return planPayload{
			Summary:       "No strong evidence found; start with task status and rebuild context as needed.",
			SuggestedTool: suggestedTool,
			NextAction:    "Inspect task status, then execute the next repairable step.",
			ToolArgs:      map[string]any{"query": req.Query},
		}
	}
	return planPayload{
		Summary:       "Evidence collected; continue with the next executable delivery step.",
		SuggestedTool: suggestedTool,
		NextAction:    "Use the strongest evidence to pick the next command and verify the outcome.",
		ToolArgs:      map[string]any{"query": req.Query, "evidence_count": len(evidence)},
	}
}

func fallbackEvaluation(req agentruntime.EvaluateRequest) evaluationPayload {
	summary := strings.ToLower(strings.TrimSpace(req.QualitySummary))
	switch {
	case strings.Contains(summary, "passed") || strings.Contains(summary, "通过"):
		return evaluationPayload{Verdict: "pass", Reason: "Quality summary indicates the current step passed.", NextAction: "Advance to the next step.", AcceptanceDelta: "No blocking acceptance gap detected."}
	case strings.Contains(summary, "missing") || strings.Contains(summary, "context"):
		return evaluationPayload{Verdict: "need_context", Reason: "Current evidence appears insufficient for a safe decision.", NextAction: "Retrieve more context and retry planning.", MissingItems: []string{"补充上下文证据", "补充需求边界说明"}, AcceptanceDelta: "当前验收证据不足，无法确认方案是否满足需求。"}
	default:
		return evaluationPayload{Verdict: "retry", Reason: "Quality summary still shows unresolved issues.", NextAction: "Prepare a repair action and retry the task.", MissingItems: []string{"修复质量门禁暴露的问题"}, AcceptanceDelta: "尚未达到当前阶段的质量验收标准。"}
	}
}

func mustJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(payload)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
