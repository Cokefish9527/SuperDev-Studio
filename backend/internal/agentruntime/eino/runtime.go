//go:build !(windows && amd64 && go1.24)

package eino

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

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
	chatModel model.BaseChatModel
}

func New(ctx context.Context, s *store.Store, retrievalService *retrieval.Service, cfg Config) (*Runtime, error) {
	runtime := &Runtime{store: s, retrieval: retrievalService}
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return runtime, nil
	}
	timeout := 60 * time.Second
	temperature := float32(0.2)
	chatModel, err := arkmodel.NewChatModel(ctx, &arkmodel.ChatModelConfig{
		APIKey:      strings.TrimSpace(cfg.APIKey),
		Model:       strings.TrimSpace(cfg.Model),
		BaseURL:     strings.TrimSpace(cfg.BaseURL),
		Timeout:     &timeout,
		Temperature: &temperature,
	})
	if err != nil {
		return nil, err
	}
	runtime.chatModel = chatModel
	return runtime, nil
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
	inputPayload := map[string]any{
		"goal":          req.Goal,
		"query":         req.Query,
		"mode":          req.ModeName,
		"allowed_tools": req.AllowedTools,
		"context":       req.Context,
	}
	inputJSON := mustJSON(inputPayload)
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

	evidenceItems, err := r.retrieval.Retrieve(ctx, retrieval.Request{ProjectID: req.ProjectID, Query: firstNonEmpty(req.Query, req.Goal), MaxItems: req.MaxEvidence})
	if err != nil {
		return agentruntime.PlanResult{}, err
	}
	persistedEvidence := make([]store.AgentEvidence, 0, len(evidenceItems))
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

	plan := fallbackPlan(req, persistedEvidence)
	raw := plan.Summary
	if r.chatModel != nil {
		if answer, genErr := r.generate(ctx, buildPlanPrompt(req, persistedEvidence)); genErr == nil && strings.TrimSpace(answer) != "" {
			raw = answer
			plan = parsePlanAnswer(answer, plan)
		}
	}

	outputJSON := mustJSON(map[string]any{
		"summary":        plan.Summary,
		"suggested_tool": plan.SuggestedTool,
		"next_action":    plan.NextAction,
		"tool_args":      plan.ToolArgs,
		"raw":            raw,
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
	return agentruntime.PlanResult{Step: step, Summary: plan.Summary, SuggestedTool: plan.SuggestedTool, NextAction: plan.NextAction, ToolArgs: plan.ToolArgs, Evidence: persistedEvidence, Raw: raw}, nil
}

func (r *Runtime) Evaluate(ctx context.Context, req agentruntime.EvaluateRequest) (agentruntime.EvaluateResult, error) {
	inputJSON := mustJSON(map[string]any{
		"goal":                  req.Goal,
		"task_title":            req.TaskTitle,
		"attempt":               req.Attempt,
		"quality_summary":       req.QualitySummary,
		"decision_context":      req.DecisionContext,
		"allowed_next_commands": req.AllowedNextCommands,
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
	raw := evaluation.Reason
	if r.chatModel != nil {
		if answer, genErr := r.generate(ctx, buildEvaluationPrompt(req)); genErr == nil && strings.TrimSpace(answer) != "" {
			raw = answer
			evaluation = parseEvaluationAnswer(answer, req, evaluation)
		}
	}
	record, err := r.store.CreateAgentEvaluation(ctx, store.AgentEvaluation{
		AgentStepID:     step.ID,
		EvaluationType:  "step-outcome",
		Verdict:         evaluation.Verdict,
		Reason:          evaluation.Reason,
		NextAction:      evaluation.NextAction,
		NextCommand:     evaluation.NextCommand,
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
		"next_command":     evaluation.NextCommand,
		"missing_items":    evaluation.MissingItems,
		"acceptance_delta": evaluation.AcceptanceDelta,
		"raw":              raw,
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
	return agentruntime.EvaluateResult{Step: step, Verdict: evaluation.Verdict, Reason: evaluation.Reason, NextAction: evaluation.NextAction, NextCommand: evaluation.NextCommand, Evaluation: record, Raw: raw}, nil
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
	NextCommand     string   `json:"next_command"`
	MissingItems    []string `json:"missing_items"`
	AcceptanceDelta string   `json:"acceptance_delta"`
}

func (r *Runtime) generate(ctx context.Context, prompt string) (string, error) {
	message, err := r.chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是 SuperDev Studio 的交付智能体，请输出简洁、可执行、可验证的工程决策。"),
		schema.UserMessage(prompt),
	}, model.WithTemperature(0.2))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(message.Content), nil
}

func buildPlanPrompt(req agentruntime.PlanRequest, evidence []store.AgentEvidence) string {
	sections := make([]string, 0, len(evidence)+4)
	sections = append(sections,
		fmt.Sprintf("目标：%s", strings.TrimSpace(req.Goal)),
		fmt.Sprintf("查询：%s", strings.TrimSpace(req.Query)),
		fmt.Sprintf("模式：%s", strings.TrimSpace(req.ModeName)),
		fmt.Sprintf("允许工具：%s", strings.Join(req.AllowedTools, ", ")),
	)
	if len(evidence) > 0 {
		sections = append(sections, "证据：")
		for _, item := range evidence {
			sections = append(sections, fmt.Sprintf("- [%s] %s :: %s", item.SourceType, item.Title, item.Snippet))
		}
	}
	sections = append(sections, "请仅输出 JSON 对象，字段：summary, suggested_tool, next_action, tool_args。")
	return strings.Join(sections, "\n")
}

func buildEvaluationPrompt(req agentruntime.EvaluateRequest) string {
	return strings.Join([]string{
		fmt.Sprintf("Goal: %s", strings.TrimSpace(req.Goal)),
		fmt.Sprintf("Task: %s", strings.TrimSpace(req.TaskTitle)),
		fmt.Sprintf("Attempt: %d", req.Attempt),
		fmt.Sprintf("Quality summary: %s", strings.TrimSpace(req.QualitySummary)),
		fmt.Sprintf("Allowed next_command values: %s", strings.Join(req.AllowedNextCommands, ", ")),
		"Return JSON only with fields: verdict(pass|retry|need_context|need_human|fail), reason, next_action, next_command, missing_items(array), acceptance_delta.",
	}, "\n")
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
	allowed := sanitizeAllowedNextCommands(req.AllowedNextCommands)
	switch {
	case strings.Contains(summary, "passed") || strings.Contains(summary, "pass"):
		return evaluationPayload{Verdict: "pass", Reason: "Quality summary indicates the current step passed.", NextAction: "Advance to the next step.", NextCommand: fallbackNextCommand("pass", allowed), AcceptanceDelta: "No blocking acceptance gap detected."}
	case strings.Contains(summary, "missing") || strings.Contains(summary, "context"):
		return evaluationPayload{Verdict: "need_context", Reason: "Current evidence appears insufficient for a safe decision.", NextAction: "Retrieve more context and retry planning.", NextCommand: fallbackNextCommand("need_context", allowed), MissingItems: []string{"Add more context evidence", "Clarify requirement boundaries"}, AcceptanceDelta: "Acceptance evidence is insufficient for a safe decision."}
	default:
		return evaluationPayload{Verdict: "retry", Reason: "Quality summary still shows unresolved issues.", NextAction: "Prepare a repair action and retry the task.", NextCommand: fallbackNextCommand("retry", allowed), MissingItems: []string{"Fix the remaining quality issues"}, AcceptanceDelta: "The current output does not yet meet the acceptance bar."}
	}
}

func parsePlanAnswer(raw string, fallback planPayload) planPayload {
	payload := extractJSONObject(raw)
	if payload == "" {
		return fallback
	}
	parsed := planPayload{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return fallback
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		parsed.Summary = fallback.Summary
	}
	if strings.TrimSpace(parsed.SuggestedTool) == "" {
		parsed.SuggestedTool = fallback.SuggestedTool
	}
	if strings.TrimSpace(parsed.NextAction) == "" {
		parsed.NextAction = fallback.NextAction
	}
	if parsed.ToolArgs == nil {
		parsed.ToolArgs = fallback.ToolArgs
	}
	return parsed
}

func parseEvaluationAnswer(raw string, req agentruntime.EvaluateRequest, fallback evaluationPayload) evaluationPayload {
	payload := extractJSONObject(raw)
	if payload == "" {
		return fallback
	}
	parsed := evaluationPayload{}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return fallback
	}
	if strings.TrimSpace(parsed.Verdict) == "" {
		parsed.Verdict = fallback.Verdict
	}
	if strings.TrimSpace(parsed.Reason) == "" {
		parsed.Reason = fallback.Reason
	}
	if strings.TrimSpace(parsed.NextAction) == "" {
		parsed.NextAction = fallback.NextAction
	}
	parsed.NextCommand = sanitizeNextCommand(parsed.NextCommand, sanitizeAllowedNextCommands(req.AllowedNextCommands), fallback.NextCommand)
	if len(parsed.MissingItems) == 0 {
		parsed.MissingItems = fallback.MissingItems
	}
	if strings.TrimSpace(parsed.AcceptanceDelta) == "" {
		parsed.AcceptanceDelta = fallback.AcceptanceDelta
	}
	return parsed
}

func sanitizeAllowedNextCommands(commands []string) []string {
	if len(commands) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(commands))
	result := make([]string, 0, len(commands))
	for _, command := range commands {
		normalized := strings.ToLower(strings.TrimSpace(command))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func sanitizeNextCommand(candidate string, allowed []string, fallback string) string {
	normalized := strings.ToLower(strings.TrimSpace(candidate))
	if normalized == "" {
		return strings.ToLower(strings.TrimSpace(fallback))
	}
	if len(allowed) == 0 {
		return normalized
	}
	for _, item := range allowed {
		if normalized == item {
			return normalized
		}
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func fallbackNextCommand(verdict string, allowed []string) string {
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case "pass":
		for _, candidate := range []string{"complete_delivery", "review_preview"} {
			if containsString(allowed, candidate) {
				return candidate
			}
		}
	case "need_human":
		if containsString(allowed, "await_human") {
			return "await_human"
		}
	default:
		if containsString(allowed, "rerun_delivery") {
			return "rerun_delivery"
		}
	}
	if len(allowed) > 0 {
		return allowed[0]
	}
	switch strings.ToLower(strings.TrimSpace(verdict)) {
	case "pass":
		return "complete_delivery"
	case "need_human":
		return "await_human"
	default:
		return "rerun_delivery"
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func extractJSONObject(raw string) string {
	re := regexp.MustCompile(`\{[\s\S]*\}`)
	return strings.TrimSpace(re.FindString(raw))
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
