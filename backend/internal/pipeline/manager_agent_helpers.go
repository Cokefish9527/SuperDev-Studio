package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"superdevstudio/internal/agentconfig"
	"superdevstudio/internal/agentruntime"
	"superdevstudio/internal/store"
)

type stepAgentSession struct {
	Run           store.AgentRun
	Config        agentconfig.Bundle
	SelectedAgent agentconfig.AgentConfig
	SelectedMode  agentconfig.ModeConfig
	AllowedTools  []string
}

func (m *Manager) bootstrapStepAgent(ctx context.Context, runID string, req StartRequest) *stepAgentSession {
	if m.agentRun == nil {
		return nil
	}
	bundle, _ := agentconfig.LoadProjectBundle(req.Options.ProjectDir)
	selectedAgent := bundle.ResolveAgent(req.Agent.Name)
	selectedModeName := strings.TrimSpace(req.Agent.Mode)
	if selectedModeName == "" && req.Lifecycle.OneClickDelivery {
		selectedModeName = "full_cycle"
	}
	selectedMode := bundle.ResolveMode(selectedModeName)
	allowedTools := resolveAllowedTools(bundle, selectedAgent.Name)
	if existing, err := m.agentRun.GetRunByPipelineRun(ctx, runID); err == nil {
		selectedAgent = bundle.ResolveAgent(existing.AgentName)
		selectedMode = bundle.ResolveMode(existing.ModeName)
		return &stepAgentSession{Run: existing, Config: bundle, SelectedAgent: selectedAgent, SelectedMode: selectedMode, AllowedTools: resolveAllowedTools(bundle, existing.AgentName)}
	}
	run, err := m.agentRun.StartRun(ctx, agentruntime.StartRunRequest{
		PipelineRunID: runID,
		ProjectID:     req.ProjectID,
		ChangeBatchID: req.ChangeBatchID,
		AgentName:     selectedAgent.Name,
		ModeName:      selectedMode.Name,
		CurrentNode:   "bootstrap",
	})
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: runID, Stage: "step-agent", Status: "failed", Message: fmt.Sprintf("Agent bootstrap failed: %v", err)})
		return nil
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: runID, Stage: "step-agent", Status: "running", Message: fmt.Sprintf("Agent runtime bootstrapped: %s (%s)", run.AgentName, run.ModeName)})
	return &stepAgentSession{Run: run, Config: bundle, SelectedAgent: selectedAgent, SelectedMode: selectedMode, AllowedTools: allowedTools}
}

func (m *Manager) finishStepAgent(ctx context.Context, session *stepAgentSession, currentNode, summary string) {
	if session == nil || m.agentRun == nil {
		return
	}
	_ = m.agentRun.FinishRun(ctx, session.Run.ID, currentNode, summary)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: session.Run.PipelineRunID, Stage: "step-agent", Status: "completed", Message: summary})
}

func (m *Manager) planStepAgent(ctx context.Context, session *stepAgentSession, req StartRequest, nodeName, title, goal, query string, contextPayload map[string]any) *agentruntime.PlanResult {
	if session == nil || m.agentRun == nil {
		return nil
	}
	result, err := m.agentRun.Plan(ctx, agentruntime.PlanRequest{
		AgentRunID:    session.Run.ID,
		ProjectID:     req.ProjectID,
		PipelineRunID: session.Run.PipelineRunID,
		NodeName:      nodeName,
		Title:         title,
		Goal:          goal,
		Query:         query,
		ModeName:      session.Run.ModeName,
		MaxEvidence:   req.Context.MaxItems,
		AllowedTools:  session.AllowedTools,
		Context:       contextPayload,
	})
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: session.Run.PipelineRunID, Stage: "step-agent", Status: "failed", Message: fmt.Sprintf("Agent plan failed for %s: %v", nodeName, err)})
		return nil
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: session.Run.PipelineRunID, Stage: "step-agent", Status: "log", Message: fmt.Sprintf("%s: %s", title, result.Summary)})
	return &result
}

func (m *Manager) evaluateStepAgent(ctx context.Context, session *stepAgentSession, req StartRequest, nodeName, title, taskTitle, qualitySummary string, attempt int, decisionContext map[string]any) *agentruntime.EvaluateResult {
	if session == nil || m.agentRun == nil {
		return nil
	}
	result, err := m.agentRun.Evaluate(ctx, agentruntime.EvaluateRequest{
		AgentRunID:          session.Run.ID,
		NodeName:            nodeName,
		Title:               title,
		Goal:                req.Prompt,
		TaskTitle:           taskTitle,
		Attempt:             attempt,
		QualitySummary:      qualitySummary,
		DecisionContext:     decisionContext,
		AllowedNextCommands: resolveAllowedNextCommands(session, nodeName, decisionContext),
	})
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: session.Run.PipelineRunID, Stage: "step-agent", Status: "failed", Message: fmt.Sprintf("Agent evaluate failed for %s: %v", nodeName, err)})
		return nil
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{RunID: session.Run.PipelineRunID, Stage: "step-agent", Status: "log", Message: fmt.Sprintf("%s: %s -> %s", title, result.Verdict, result.Reason)})
	return &result
}

func (m *Manager) runAgentCommandStageWithLines(ctx context.Context, runID string, options RunRequest, stage, toolName string, commandArgs []string, progress int, session *stepAgentSession, plan *agentruntime.PlanResult) ([]string, error) {
	started := time.Now()
	lines, err := m.runCommandStageWithLines(ctx, runID, stage, options, commandArgs, progress)
	if session != nil && plan != nil && m.agentRun != nil {
		response := map[string]any{"lines": lines}
		if err != nil {
			response["error"] = err.Error()
		}
		_, _ = m.agentRun.RecordToolCall(ctx, agentruntime.ToolCallRequest{
			AgentStepID: plan.Step.ID,
			ToolName:    toolName,
			Request: map[string]any{
				"stage": stage,
				"args":  commandArgs,
			},
			Response: response,
			Success:  err == nil,
			Latency:  time.Since(started),
		})
	}
	return lines, err
}

func (m *Manager) runAgentCommandStage(ctx context.Context, runID string, options RunRequest, stage, toolName string, commandArgs []string, progress int, session *stepAgentSession, plan *agentruntime.PlanResult) error {
	_, err := m.runAgentCommandStageWithLines(ctx, runID, options, stage, toolName, commandArgs, progress, session, plan)
	return err
}

func resolveAllowedTools(bundle agentconfig.Bundle, agentName string) []string {
	if item, ok := bundle.FindAgent(agentName); ok && len(item.AllowedTools) > 0 {
		return item.AllowedTools
	}
	if len(bundle.Agents) > 0 && len(bundle.Agents[0].AllowedTools) > 0 {
		return bundle.Agents[0].AllowedTools
	}
	return []string{"search_context", "run_superdev_create", "run_superdev_spec_validate", "run_superdev_task_status", "run_superdev_task_run", "run_superdev_quality", "run_superdev_preview", "run_superdev_deploy", "read_artifact", "append_run_event"}
}

func resolveAllowedNextCommands(_ *stepAgentSession, _ string, _ map[string]any) []string {
	return []string{"rerun_delivery", "await_human", "review_preview", "complete_delivery"}
}

func agentVerdictAllowsAdvance(verdict string) bool {
	return strings.EqualFold(strings.TrimSpace(verdict), "pass")
}

func agentVerdictNeedsContext(verdict string) bool {
	return strings.EqualFold(strings.TrimSpace(verdict), "need_context")
}

func agentVerdictNeedsHuman(verdict string) bool {
	return strings.EqualFold(strings.TrimSpace(verdict), "need_human")
}
