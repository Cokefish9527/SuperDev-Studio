package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"superdevstudio/internal/agentruntime"
	"superdevstudio/internal/contextopt"
	"superdevstudio/internal/store"
)

func (m *Manager) handoffStepByStepToHuman(
	ctx context.Context,
	runID string,
	req StartRequest,
	stage string,
	progress int,
	evaluation *agentruntime.EvaluateResult,
	phasePacks []PhaseContextPack,
) {
	if evaluation == nil {
		return
	}
	reason := firstNonEmpty(strings.TrimSpace(evaluation.Reason), "Agent requested human intervention")
	nextAction := strings.TrimSpace(evaluation.NextAction)
	summary := reason
	if nextAction != "" {
		summary = fmt.Sprintf("%s Next: %s", reason, nextAction)
	}

	m.maybeGenerateReflectionArtifact(ctx, runID, req, "", "", "awaiting_human", reason)
	m.writebackRunMemory(ctx, req, runID, "awaiting_human", stage, reason, phasePacks)
	if m.agentRun != nil {
		if agentRun, err := m.agentRun.GetRunByPipelineRun(ctx, runID); err == nil {
			_ = m.store.UpdateAgentRun(ctx, agentRun.ID, "awaiting_human", "step-human-handoff", summary, nil)
		}
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 99 {
		progress = 99
	}
	_ = m.store.UpdatePipelineRun(ctx, runID, "awaiting_human", "step-human-handoff", progress, nil, nil)
	m.touchChangeBatch(ctx, req.ChangeBatchID, "awaiting_human", runID, "")
	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-human-handoff",
		Status:  "blocked",
		Message: "Human intervention required: " + reason,
	})
	if nextAction != "" {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-human-handoff",
			Status:  "log",
			Message: "Suggested next action: " + nextAction,
		})
	}
}

func (m *Manager) enrichStepByStepNeedContext(
	ctx context.Context,
	runID string,
	req StartRequest,
	task store.Task,
	attempt int,
	evaluation *agentruntime.EvaluateResult,
	qualitySummary string,
) string {
	if evaluation == nil || m.contextOpt == nil || strings.TrimSpace(req.ProjectID) == "" {
		return ""
	}
	query := firstNonEmpty(
		strings.TrimSpace(strings.Join([]string{task.Title, evaluation.Reason, evaluation.NextAction, qualitySummary}, " ")),
		strings.TrimSpace(task.Title),
		strings.TrimSpace(req.Context.Query),
		strings.TrimSpace(req.Prompt),
	)
	if query == "" {
		return ""
	}
	tokenBudget := req.Context.TokenBudget
	if tokenBudget <= 0 {
		tokenBudget = 1200
	}
	maxItems := req.Context.MaxItems
	if maxItems <= 0 {
		maxItems = 8
	}
	pack, err := m.contextOpt.BuildContextPack(ctx, contextopt.BuildRequest{
		ProjectID:   req.ProjectID,
		Query:       query,
		TokenBudget: tokenBudget,
		MaxItems:    maxItems,
	})
	if err != nil {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-context-enrichment",
			Status:  "failed",
			Message: fmt.Sprintf("Need-context enrichment failed: %v", err),
		})
		return ""
	}

	memoryIDs := make([]string, 0, len(pack.Memories))
	for _, item := range pack.Memories {
		memoryIDs = append(memoryIDs, item.ID)
	}
	knowledgeIDs := make([]string, 0, len(pack.Knowledge))
	for _, item := range pack.Knowledge {
		knowledgeIDs = append(knowledgeIDs, strconv.FormatInt(item.ID, 10))
	}
	metadata, _ := json.Marshal(map[string]any{
		"query":            query,
		"reason":           strings.TrimSpace(evaluation.Reason),
		"next_action":      strings.TrimSpace(evaluation.NextAction),
		"attempt":          attempt,
		"task_id":          strings.TrimSpace(task.ID),
		"task_title":       strings.TrimSpace(task.Title),
		"memory_ids":       memoryIDs,
		"knowledge_ids":    knowledgeIDs,
		"memory_count":     len(pack.Memories),
		"knowledge_count":  len(pack.Knowledge),
		"estimated_tokens": pack.EstimatedTokens,
	})
	_, _ = m.store.CreateAgentEvidence(ctx, store.AgentEvidence{
		AgentStepID:  evaluation.Step.ID,
		SourceType:   "context_enrichment",
		SourceID:     query,
		Title:        firstNonEmpty(strings.TrimSpace(task.Title), "need_context enrichment"),
		Snippet:      strings.TrimSpace(pack.Summary),
		Score:        1,
		MetadataJSON: string(metadata),
	})

	_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
		RunID:   runID,
		Stage:   "step-context-enrichment",
		Status:  "completed",
		Message: fmt.Sprintf("Need-context enrichment loaded memories=%d knowledge=%d", len(pack.Memories), len(pack.Knowledge)),
	})
	if strings.TrimSpace(pack.Summary) != "" {
		_, _ = m.store.AppendRunEvent(ctx, store.RunEvent{
			RunID:   runID,
			Stage:   "step-context-enrichment",
			Status:  "log",
			Message: pack.Summary,
		})
	}
	return strings.TrimSpace(pack.Summary)
}
