package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"superdevstudio/internal/store"
)

const (
	syncResidualPrefix = "sync:run:"
	syncApprovalPrefix = "sync:approval:"
)

type updateResidualItemRequest struct {
	Status         string `json:"status"`
	ResolutionNote string `json:"resolution_note"`
}

type toolApprovalSnapshot struct {
	Status               string `json:"status"`
	RiskLevel            string `json:"risk_level"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
	Approved             bool   `json:"approved"`
	Reason               string `json:"reason"`
}

func (s *Server) handleListProjectResidualItems(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	items, err := s.store.ListResidualItems(r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("run_id")), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListProjectApprovalGates(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	limit := parseLimit(r.URL.Query().Get("limit"), 100)
	items, err := s.store.ListApprovalGates(r.Context(), projectID, strings.TrimSpace(r.URL.Query().Get("run_id")), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListRunResidualItems(w http.ResponseWriter, r *http.Request) {
	run, err := s.resolveRunForFollowups(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	if err := s.syncRunFollowups(r.Context(), run); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	items, err := s.store.ListResidualItems(r.Context(), run.ProjectID, run.ID, parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleListRunApprovalGates(w http.ResponseWriter, r *http.Request) {
	run, err := s.resolveRunForFollowups(r)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	if err := s.syncRunFollowups(r.Context(), run); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	items, err := s.store.ListApprovalGates(r.Context(), run.ProjectID, run.ID, parseLimit(r.URL.Query().Get("limit"), 100))
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleUpdateResidualItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	var req updateResidualItemRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "open", "resolved", "waived":
	default:
		respondError(w, http.StatusBadRequest, errors.New("status must be one of: open, resolved, waived"))
		return
	}
	item, err := s.store.UpdateResidualItemStatus(r.Context(), itemID, status, req.ResolutionNote)
	if err != nil {
		handleAgentResolveError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, item)
}

func (s *Server) resolveRunForFollowups(r *http.Request) (store.PipelineRun, error) {
	runID := chi.URLParam(r, "runID")
	return s.store.GetPipelineRun(r.Context(), runID)
}

func (s *Server) syncRunFollowups(ctx context.Context, run store.PipelineRun) error {
	var agentRun *store.AgentRun
	var evaluations []store.AgentEvaluation
	var toolCalls []store.AgentToolCall
	if loaded, err := s.store.GetAgentRunByPipelineRun(ctx, run.ID); err == nil {
		agentRun = &loaded
		evaluations, _ = s.store.ListAgentEvaluations(ctx, loaded.ID)
		toolCalls, _ = s.store.ListAgentToolCalls(ctx, loaded.ID)
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	events, err := s.store.ListRunEvents(ctx, run.ID)
	if err != nil {
		return err
	}
	completion := buildPipelineCompletionResponse(run)
	residuals := deriveResidualItems(run, agentRun, evaluations, events, completion)
	approvalGates := deriveApprovalGates(run, toolCalls)

	activeResiduals := make(map[string]struct{}, len(residuals))
	for _, item := range residuals {
		activeResiduals[item.SourceKey] = struct{}{}
		if _, err := s.store.UpsertResidualItem(ctx, item); err != nil {
			return err
		}
	}
	existingResiduals, err := s.store.ListResidualItems(ctx, run.ProjectID, run.ID, 200)
	if err != nil {
		return err
	}
	for _, item := range existingResiduals {
		if !strings.HasPrefix(item.SourceKey, syncResidualPrefix) {
			continue
		}
		if _, ok := activeResiduals[item.SourceKey]; ok {
			continue
		}
		if item.Status == "open" {
			if _, err := s.store.UpdateResidualItemStatus(ctx, item.ID, "resolved", "Auto-resolved after run state changed"); err != nil {
				return err
			}
		}
	}

	activeGates := make(map[string]struct{}, len(approvalGates))
	for _, gate := range approvalGates {
		activeGates[gate.SourceKey] = struct{}{}
		if _, err := s.store.UpsertApprovalGate(ctx, gate); err != nil {
			return err
		}
	}
	existingGates, err := s.store.ListApprovalGates(ctx, run.ProjectID, run.ID, 200)
	if err != nil {
		return err
	}
	for _, gate := range existingGates {
		if !strings.HasPrefix(gate.SourceKey, syncApprovalPrefix) {
			continue
		}
		if _, ok := activeGates[gate.SourceKey]; ok {
			continue
		}
		if _, err := s.store.UpsertApprovalGate(ctx, store.ApprovalGate{
			ID:            gate.ID,
			ProjectID:     gate.ProjectID,
			PipelineRunID: gate.PipelineRunID,
			ChangeBatchID: gate.ChangeBatchID,
			GateType:      gate.GateType,
			Title:         gate.Title,
			Detail:        gate.Detail,
			ToolName:      gate.ToolName,
			RiskLevel:     gate.RiskLevel,
			SourceKey:     gate.SourceKey,
			Status:        "resolved",
		}); err != nil {
			return err
		}
	}
	return nil
}

func deriveResidualItems(run store.PipelineRun, agentRun *store.AgentRun, evaluations []store.AgentEvaluation, events []store.RunEvent, completion pipelineCompletionResponse) []store.ResidualItem {
	items := make([]store.ResidualItem, 0, 4)
	agentRunID := ""
	if agentRun != nil {
		agentRunID = agentRun.ID
	}
	latestEvaluation := latestEvaluationWithVerdict(evaluations, "need_human", "need_context")
	if latestEvaluation != nil {
		verdict := strings.ToLower(strings.TrimSpace(latestEvaluation.Verdict))
		if verdict == "need_context" || verdict == "need_human" {
			items = append(items, store.ResidualItem{
				ProjectID:        run.ProjectID,
				PipelineRunID:    run.ID,
				AgentRunID:       agentRunID,
				Stage:            firstNonEmpty(run.Stage, agentRunStage(agentRun)),
				Category:         residualCategoryFromStage(run.Stage, verdict),
				Severity:         mapResidualSeverity(verdict, run.Stage),
				Summary:          mapResidualSummary(verdict),
				Evidence:         residualEvidence(latestEvaluation),
				SuggestedCommand: residualSuggestedCommand(run.ID, verdict, latestEvaluation),
				SourceKey:        syncResidualPrefix + run.ID + ":" + verdict,
				Status:           "open",
			})
		}
		for index, missingItem := range latestEvaluation.MissingItems {
			trimmed := strings.TrimSpace(missingItem)
			if trimmed == "" {
				continue
			}
			items = append(items, store.ResidualItem{
				ProjectID:        run.ProjectID,
				PipelineRunID:    run.ID,
				AgentRunID:       agentRunID,
				Stage:            firstNonEmpty(run.Stage, agentRunStage(agentRun)),
				Category:         residualCategoryFromStage(run.Stage, verdict),
				Severity:         mapResidualSeverity(verdict, run.Stage),
				Summary:          trimmed,
				Evidence:         residualEvidence(latestEvaluation),
				SuggestedCommand: residualSuggestedCommand(run.ID, verdict, latestEvaluation),
				SourceKey:        fmt.Sprintf("%s%s:missing:%d", syncResidualPrefix, run.ID, index),
				Status:           "open",
			})
		}
	}
	if strings.EqualFold(strings.TrimSpace(run.Status), "failed") {
		items = append(items, store.ResidualItem{
			ProjectID:        run.ProjectID,
			PipelineRunID:    run.ID,
			AgentRunID:       agentRunID,
			Stage:            run.Stage,
			Category:         residualCategoryFromStage(run.Stage, "failed"),
			Severity:         mapResidualSeverity("failed", run.Stage),
			Summary:          fmt.Sprintf("运行失败：%s", firstNonEmpty(run.Stage, "unknown-stage")),
			Evidence:         latestRunMessage(events, "Pipeline run failed without detailed event message."),
			SuggestedCommand: fmt.Sprintf("POST /api/pipeline/runs/%s/auto-advance", run.ID),
			SourceKey:        syncResidualPrefix + run.ID + ":failed",
			Status:           "open",
		})
	}
	if strings.EqualFold(strings.TrimSpace(run.Status), "completed") && strings.TrimSpace(completion.PreviewURL) == "" {
		items = append(items, store.ResidualItem{
			ProjectID:        run.ProjectID,
			PipelineRunID:    run.ID,
			AgentRunID:       agentRunID,
			Stage:            "preview",
			Category:         "preview",
			Severity:         "medium",
			Summary:          "交付已完成，但缺少可预览产物",
			Evidence:         "未检测到 HTML 预览页面或主预览 URL。",
			SuggestedCommand: fmt.Sprintf("GET /api/pipeline/runs/%s/completion", run.ID),
			SourceKey:        syncResidualPrefix + run.ID + ":preview-missing",
			Status:           "open",
		})
	}
	return items
}

func deriveApprovalGates(run store.PipelineRun, toolCalls []store.AgentToolCall) []store.ApprovalGate {
	items := make([]store.ApprovalGate, 0, 2)
	for _, call := range toolCalls {
		state, ok := parseApprovalSnapshot(call)
		if !ok {
			continue
		}
		status := "resolved"
		if strings.EqualFold(run.Status, "awaiting_human") && strings.EqualFold(state.Status, "awaiting_approval") && !state.Approved {
			status = "open"
		}
		items = append(items, store.ApprovalGate{
			ProjectID:     run.ProjectID,
			PipelineRunID: run.ID,
			ChangeBatchID: run.ChangeBatchID,
			GateType:      "tool_governance",
			Title:         "高风险动作待人工确认",
			Detail:        firstNonEmpty(state.Reason, "高风险工具调用需要人工确认后继续执行。"),
			ToolName:      call.ToolName,
			RiskLevel:     firstNonEmpty(state.RiskLevel, "high"),
			SourceKey:     syncApprovalPrefix + run.ID + ":" + call.ToolName,
			Status:        status,
		})
	}
	return items
}

func latestEvaluationWithVerdict(items []store.AgentEvaluation, verdicts ...string) *store.AgentEvaluation {
	if len(items) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, verdict := range verdicts {
		allowed[strings.ToLower(strings.TrimSpace(verdict))] = struct{}{}
	}
	for idx := len(items) - 1; idx >= 0; idx-- {
		if _, ok := allowed[strings.ToLower(strings.TrimSpace(items[idx].Verdict))]; ok {
			return &items[idx]
		}
	}
	return nil
}

func parseApprovalSnapshot(call store.AgentToolCall) (toolApprovalSnapshot, bool) {
	var state toolApprovalSnapshot
	trimmed := strings.TrimSpace(call.ResponseJSON)
	if trimmed == "" {
		return toolApprovalSnapshot{}, false
	}
	if err := json.Unmarshal([]byte(trimmed), &state); err != nil {
		return toolApprovalSnapshot{}, false
	}
	if !state.RequiresConfirmation && !strings.EqualFold(state.Status, "awaiting_approval") && strings.TrimSpace(state.RiskLevel) == "" {
		return toolApprovalSnapshot{}, false
	}
	return state, true
}

func residualCategoryFromStage(stage, verdict string) string {
	lower := strings.ToLower(strings.TrimSpace(stage))
	if verdict == "need_context" {
		return "requirement"
	}
	if strings.Contains(lower, "quality") || strings.Contains(lower, "test") {
		return "quality"
	}
	if strings.Contains(lower, "preview") {
		return "preview"
	}
	if strings.Contains(lower, "release") || strings.Contains(lower, "deploy") {
		return "release"
	}
	if strings.Contains(lower, "requirement") || strings.Contains(lower, "context") || strings.Contains(lower, "discovery") {
		return "requirement"
	}
	return "dev"
}

func mapResidualSeverity(kind, stage string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "need_human":
		return "high"
	case "failed":
		if residualCategoryFromStage(stage, kind) == "release" {
			return "critical"
		}
		return "high"
	default:
		return "medium"
	}
}

func residualEvidence(evaluation *store.AgentEvaluation) string {
	if evaluation == nil {
		return ""
	}
	parts := make([]string, 0, 2)
	if reason := strings.TrimSpace(evaluation.Reason); reason != "" {
		parts = append(parts, reason)
	}
	if delta := strings.TrimSpace(evaluation.AcceptanceDelta); delta != "" {
		parts = append(parts, "Acceptance delta: "+delta)
	}
	return strings.Join(parts, "\n")
}

func mapResidualSummary(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "need_human":
		return "需要人工确认后继续推进"
	case "need_context":
		return "需要补强上下文后继续推进"
	default:
		return "存在待处理残留项"
	}
}

func residualSuggestedCommand(runID, kind string, evaluation *store.AgentEvaluation) string {
	if evaluation != nil {
		if suggested := suggestedCommandForNextCommand(runID, evaluation.NextCommand); suggested != "" {
			return suggested
		}
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "need_human":
		return fmt.Sprintf("Human review required before POST /api/pipeline/runs/%s/auto-advance", runID)
	case "need_context":
		return fmt.Sprintf("Add more context, then POST /api/pipeline/runs/%s/auto-advance", runID)
	default:
		return "Continue the standard repair workflow"
	}
}

func suggestedCommandForNextCommand(runID, nextCommand string) string {
	switch strings.ToLower(strings.TrimSpace(nextCommand)) {
	case "rerun_delivery":
		return fmt.Sprintf("POST /api/pipeline/runs/%s/auto-advance", runID)
	case "await_human":
		return fmt.Sprintf("Human review required before POST /api/pipeline/runs/%s/auto-advance", runID)
	case "review_preview":
		return fmt.Sprintf("Review preview first, then POST /api/pipeline/runs/%s/auto-advance", runID)
	case "complete_delivery":
		return fmt.Sprintf("GET /api/pipeline/runs/%s/completion", runID)
	default:
		return ""
	}
}

func latestRunMessage(events []store.RunEvent, fallback string) string {
	for idx := len(events) - 1; idx >= 0; idx-- {
		message := strings.TrimSpace(events[idx].Message)
		if message != "" {
			return message
		}
	}
	return fallback
}

func agentRunStage(agentRun *store.AgentRun) string {
	if agentRun == nil {
		return ""
	}
	return strings.TrimSpace(agentRun.CurrentNode)
}
