package pipeline

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"superdevstudio/internal/agentruntime"
	"superdevstudio/internal/store"
)

const (
	fullCycleReleaseApprovalStage = "lifecycle-release-approval"
	highRiskDeployToolName        = "run_superdev_deploy"
)

func (m *Manager) requiresToolApproval(session *stepAgentSession, toolName string) bool {
	if session == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(toolName), highRiskDeployToolName) {
		return false
	}
	return session.SelectedMode.RequireApproval
}

func (m *Manager) pauseFullCycleForToolApproval(
	ctx context.Context,
	runID string,
	req StartRequest,
	progress int,
	session *stepAgentSession,
	plan *agentruntime.PlanResult,
	toolName string,
	commandArgs []string,
) {
	reason := "High-risk deploy action requires human confirmation before continuing."
	nextAction := "Confirm deploy and continue execution."
	_ = m.store.UpdatePipelineRun(ctx, runID, "awaiting_human", fullCycleReleaseApprovalStage, progress, nil, nil)
	m.touchChangeBatch(ctx, req.ChangeBatchID, "awaiting_human", runID, "")
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   fullCycleReleaseApprovalStage,
		Status:  "warning",
		Message: reason,
	})
	if session == nil {
		return
	}
	_ = m.store.UpdateAgentRun(ctx, session.Run.ID, "awaiting_human", fullCycleReleaseApprovalStage, reason, nil)
	if plan == nil {
		return
	}
	_, _ = m.store.CreateAgentEvaluation(ctx, store.AgentEvaluation{
		AgentStepID:    plan.Step.ID,
		EvaluationType: "tool_governance",
		Verdict:        "need_human",
		Reason:         reason,
		NextAction:     nextAction,
		NextCommand:    "await_human",
	})
	if m.agentRun != nil {
		_, _ = m.agentRun.RecordToolCall(ctx, agentruntime.ToolCallRequest{
			AgentStepID: plan.Step.ID,
			ToolName:    toolName,
			Request: map[string]any{
				"stage": fullCycleReleaseApprovalStage,
				"args":  commandArgs,
			},
			Response: map[string]any{
				"status":                "awaiting_approval",
				"risk_level":            "high",
				"requires_confirmation": true,
				"approved":              false,
				"reason":                reason,
			},
			Success: false,
			Latency: 0,
		})
	}
}

func (m *Manager) ApprovePendingTool(ctx context.Context, runID, toolName string) (store.PipelineRun, error) {
	resolvedTool := firstNonEmpty(strings.TrimSpace(toolName), highRiskDeployToolName)
	if !strings.EqualFold(resolvedTool, highRiskDeployToolName) {
		return store.PipelineRun{}, errors.New("unsupported pending tool")
	}
	run, err := m.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return store.PipelineRun{}, err
	}
	if !run.FullCycle {
		return store.PipelineRun{}, errors.New("run is not a full_cycle pipeline")
	}
	if run.Status != "awaiting_human" || run.Stage != fullCycleReleaseApprovalStage {
		return store.PipelineRun{}, errors.New("run is not waiting for high-risk tool approval")
	}
	req := m.startRequestFromPipelineRun(ctx, run)
	_ = m.store.UpdatePipelineRun(ctx, runID, "running", "lifecycle-release", 88, nil, nil)
	m.touchChangeBatch(ctx, run.ChangeBatchID, "running", runID, run.ExternalChangeID)
	if agentRun, agentErr := m.store.GetAgentRunByPipelineRun(ctx, runID); agentErr == nil {
		_ = m.store.UpdateAgentRun(ctx, agentRun.ID, "running", "agent-plan-fullcycle-release", "Human approved high-risk deploy action", nil)
	}
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   fullCycleReleaseApprovalStage,
		Status:  "completed",
		Message: "Human approved high-risk deploy action; continue execution.",
	})
	updated, getErr := m.store.GetPipelineRun(ctx, runID)
	if getErr != nil {
		return store.PipelineRun{}, getErr
	}
	go m.resumeApprovedFullCycleRelease(runID, req, resolvedTool)
	return updated, nil
}

func (m *Manager) resumeApprovedFullCycleRelease(runID string, req StartRequest, toolName string) {
	ctx := context.Background()
	run, err := m.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return
	}
	agentSession := m.bootstrapStepAgent(ctx, runID, req)
	changeID := strings.TrimSpace(run.ExternalChangeID)
	qualitySummary := loadLatestQualitySummary(req.Options.ProjectDir)
	releasePlan := m.planStepAgent(ctx, agentSession, req, "agent-plan-fullcycle-release-approved", "Continue approved release", req.Prompt, firstNonEmpty(changeID, req.Prompt), map[string]any{
		"phase":     "release",
		"approval":  "approved",
		"tool_name": toolName,
	})
	if err := m.runAgentCommandStage(
		ctx,
		runID,
		req.Options,
		"lifecycle-release",
		toolName,
		[]string{"deploy", "--docker", "--cicd", "all"},
		90,
		agentSession,
		releasePlan,
	); err != nil {
		m.failRun(ctx, runID, req, "lifecycle-release", err, nil)
		return
	}
	previewPlan := m.planStepAgent(ctx, agentSession, req, "agent-plan-fullcycle-preview", "Prepare preview artifact", req.Prompt, firstNonEmpty(changeID, req.Prompt), map[string]any{"phase": "preview", "approval": "approved"})
	if err := m.runAgentCommandStage(
		ctx,
		runID,
		req.Options,
		"lifecycle-preview",
		"run_superdev_preview",
		[]string{"preview", "--output", "output/preview.html"},
		95,
		agentSession,
		previewPlan,
	); err != nil {
		m.failRun(ctx, runID, req, "lifecycle-preview", err, nil)
		return
	}
	m.maybeGenerateReflectionArtifact(ctx, runID, req, changeID, qualitySummary, "completed", "")
	m.writebackRunMemory(ctx, req, runID, "completed", "done", "", nil)
	if agentSession != nil {
		m.finishStepAgent(ctx, agentSession, "done", "Full-cycle lifecycle finished after human approval")
	}
	finished := time.Now().UTC()
	_ = m.store.UpdatePipelineRun(ctx, runID, "completed", "done", 100, nil, &finished)
	m.touchChangeBatch(ctx, req.ChangeBatchID, "completed", runID, changeID)
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "done",
		Status:  "completed",
		Message: "Full-cycle lifecycle finished after human approval",
	})
}

func (m *Manager) startRequestFromPipelineRun(ctx context.Context, run store.PipelineRun) StartRequest {
	agentName := ""
	agentMode := ""
	if agentRun, err := m.store.GetAgentRunByPipelineRun(ctx, run.ID); err == nil {
		agentName = agentRun.AgentName
		agentMode = agentRun.ModeName
	}
	if agentMode == "" && run.FullCycle {
		agentMode = "full_cycle"
	}
	iterationLimit := run.IterationLimit
	if run.FullCycle && iterationLimit <= 0 {
		iterationLimit = 3
	}
	contextMode := ContextMode(strings.TrimSpace(run.ContextMode))
	if contextMode == "" {
		contextMode = ContextModeOff
	}
	return StartRequest{
		ProjectID:     run.ProjectID,
		ChangeBatchID: run.ChangeBatchID,
		Prompt:        run.Prompt,
		Simulate:      run.Simulate,
		RetryOf:       run.RetryOf,
		LLM: LLMOptions{
			EnhancedLoop:     run.LLMEnhancedLoop,
			MultimodalAssets: run.MultimodalAssets,
		},
		Agent: AgentOptions{
			Name: agentName,
			Mode: agentMode,
		},
		Context: ContextOptions{
			Mode:            contextMode,
			Query:           strings.TrimSpace(run.ContextQuery),
			TokenBudget:     run.ContextTokenBudget,
			MaxItems:        run.ContextMaxItems,
			DynamicByPhase:  run.ContextDynamic,
			MemoryWriteback: run.MemoryWriteback,
		},
		Lifecycle: LifecycleOptions{
			OneClickDelivery: run.FullCycle,
			StepByStep:       run.StepByStep,
			IterationLimit:   iterationLimit,
		},
		Options: RunRequest{
			Prompt:     run.Prompt,
			ProjectDir: strings.TrimSpace(run.ProjectDir),
			Platform:   strings.TrimSpace(run.Platform),
			Frontend:   strings.TrimSpace(run.Frontend),
			Backend:    strings.TrimSpace(run.Backend),
			Domain:     strings.TrimSpace(run.Domain),
		},
	}
}

func loadLatestQualitySummary(projectDir string) string {
	outputDir := filepath.Join(resolveProjectDir(projectDir), "output")
	reportPath, err := findLatestQualityGateReport(outputDir)
	if err != nil {
		return ""
	}
	content, readErr := os.ReadFile(reportPath)
	if readErr != nil {
		return ""
	}
	lines := strings.Split(string(content), "\n")
	compact := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		compact = append(compact, trimmed)
	}
	if len(compact) > 6 {
		compact = compact[len(compact)-6:]
	}
	return summarizeQualityOutput(compact)
}
