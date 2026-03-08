package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Store) CreateAgentRun(ctx context.Context, run AgentRun) (AgentRun, error) {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	if strings.TrimSpace(run.AgentName) == "" {
		run.AgentName = "delivery-agent"
	}
	if strings.TrimSpace(run.ModeName) == "" {
		run.ModeName = "step_by_step"
	}
	if strings.TrimSpace(run.Status) == "" {
		run.Status = "running"
	}
	now := nowUTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = run.CreatedAt
	}
	if run.StartedAt == nil {
		run.StartedAt = &run.CreatedAt
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agent_runs(id, pipeline_run_id, project_id, change_batch_id, agent_name, mode_name, status, current_node, summary, started_at, finished_at, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		run.ID,
		run.PipelineRunID,
		run.ProjectID,
		run.ChangeBatchID,
		run.AgentName,
		run.ModeName,
		run.Status,
		run.CurrentNode,
		run.Summary,
		nullableTime(run.StartedAt),
		nullableTime(run.FinishedAt),
		formatTime(run.CreatedAt),
		formatTime(run.UpdatedAt),
	)
	if err != nil {
		return AgentRun{}, err
	}
	return run, nil
}

func (s *Store) UpdateAgentRun(ctx context.Context, runID, status, currentNode, summary string, finishedAt *time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE agent_runs SET status=?, current_node=?, summary=COALESCE(NULLIF(?, ''), summary), finished_at=COALESCE(?, finished_at), updated_at=? WHERE id=?`,
		strings.TrimSpace(status),
		strings.TrimSpace(currentNode),
		strings.TrimSpace(summary),
		nullableTime(finishedAt),
		formatTime(nowUTC()),
		runID,
	)
	return err
}

func (s *Store) GetAgentRun(ctx context.Context, runID string) (AgentRun, error) {
	var run AgentRun
	if err := scanAgentRun(s.db.QueryRowContext(ctx, `SELECT id, pipeline_run_id, project_id, change_batch_id, agent_name, mode_name, status, current_node, summary, started_at, finished_at, created_at, updated_at FROM agent_runs WHERE id=?`, runID), &run); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentRun{}, ErrNotFound
		}
		return AgentRun{}, err
	}
	return run, nil
}

func (s *Store) GetAgentRunByPipelineRun(ctx context.Context, pipelineRunID string) (AgentRun, error) {
	var run AgentRun
	if err := scanAgentRun(s.db.QueryRowContext(ctx, `SELECT id, pipeline_run_id, project_id, change_batch_id, agent_name, mode_name, status, current_node, summary, started_at, finished_at, created_at, updated_at FROM agent_runs WHERE pipeline_run_id=? ORDER BY created_at DESC LIMIT 1`, pipelineRunID), &run); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AgentRun{}, ErrNotFound
		}
		return AgentRun{}, err
	}
	return run, nil
}

func (s *Store) CreateAgentStep(ctx context.Context, step AgentStep) (AgentStep, error) {
	if step.ID == "" {
		step.ID = uuid.NewString()
	}
	if strings.TrimSpace(step.Status) == "" {
		step.Status = "running"
	}
	if strings.TrimSpace(step.InputJSON) == "" {
		step.InputJSON = "{}"
	}
	if strings.TrimSpace(step.OutputJSON) == "" {
		step.OutputJSON = "{}"
	}
	now := nowUTC()
	if step.CreatedAt.IsZero() {
		step.CreatedAt = now
	}
	if step.UpdatedAt.IsZero() {
		step.UpdatedAt = step.CreatedAt
	}
	if step.StartedAt == nil {
		step.StartedAt = &step.CreatedAt
	}
	if step.StepIndex <= 0 {
		step.StepIndex = s.nextAgentStepIndex(ctx, step.AgentRunID)
	}
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agent_steps(id, agent_run_id, step_index, node_name, title, input_json, output_json, decision_summary, status, started_at, finished_at, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		step.ID,
		step.AgentRunID,
		step.StepIndex,
		step.NodeName,
		step.Title,
		step.InputJSON,
		step.OutputJSON,
		step.DecisionSummary,
		step.Status,
		nullableTime(step.StartedAt),
		nullableTime(step.FinishedAt),
		formatTime(step.CreatedAt),
		formatTime(step.UpdatedAt),
	)
	if err != nil {
		return AgentStep{}, err
	}
	return step, nil
}

func (s *Store) UpdateAgentStep(ctx context.Context, stepID, status, outputJSON, decisionSummary string, finishedAt *time.Time) error {
	_, err := s.db.ExecContext(
		ctx,
		`UPDATE agent_steps SET status=?, output_json=COALESCE(NULLIF(?, ''), output_json), decision_summary=COALESCE(NULLIF(?, ''), decision_summary), finished_at=COALESCE(?, finished_at), updated_at=? WHERE id=?`,
		strings.TrimSpace(status),
		strings.TrimSpace(outputJSON),
		strings.TrimSpace(decisionSummary),
		nullableTime(finishedAt),
		formatTime(nowUTC()),
		stepID,
	)
	return err
}

func (s *Store) ListAgentSteps(ctx context.Context, agentRunID string) ([]AgentStep, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, agent_run_id, step_index, node_name, title, input_json, output_json, decision_summary, status, started_at, finished_at, created_at, updated_at FROM agent_steps WHERE agent_run_id=? ORDER BY step_index ASC, created_at ASC`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AgentStep, 0, 8)
	for rows.Next() {
		var step AgentStep
		if err := scanAgentStep(rows, &step); err != nil {
			return nil, err
		}
		items = append(items, step)
	}
	return items, rows.Err()
}

func (s *Store) CreateAgentToolCall(ctx context.Context, call AgentToolCall) (AgentToolCall, error) {
	if call.ID == "" {
		call.ID = uuid.NewString()
	}
	now := nowUTC()
	if call.CreatedAt.IsZero() {
		call.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_tool_calls(id, agent_step_id, tool_name, request_json, response_json, success, latency_ms, created_at) VALUES(?,?,?,?,?,?,?,?)`, call.ID, call.AgentStepID, call.ToolName, defaultJSONObject(call.RequestJSON), defaultJSONObject(call.ResponseJSON), boolToInt(call.Success), call.LatencyMS, formatTime(call.CreatedAt))
	if err != nil {
		return AgentToolCall{}, err
	}
	return call, nil
}

func (s *Store) ListAgentToolCalls(ctx context.Context, agentRunID string) ([]AgentToolCall, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tc.id, tc.agent_step_id, tc.tool_name, tc.request_json, tc.response_json, tc.success, tc.latency_ms, tc.created_at FROM agent_tool_calls tc JOIN agent_steps st ON st.id = tc.agent_step_id WHERE st.agent_run_id=? ORDER BY tc.created_at ASC`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AgentToolCall, 0, 8)
	for rows.Next() {
		var call AgentToolCall
		var successRaw int
		var createdRaw string
		if err := rows.Scan(&call.ID, &call.AgentStepID, &call.ToolName, &call.RequestJSON, &call.ResponseJSON, &successRaw, &call.LatencyMS, &createdRaw); err != nil {
			return nil, err
		}
		call.Success = intToBool(successRaw)
		call.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, call)
	}
	return items, rows.Err()
}

func (s *Store) CreateAgentEvidence(ctx context.Context, evidence AgentEvidence) (AgentEvidence, error) {
	if evidence.ID == "" {
		evidence.ID = uuid.NewString()
	}
	if strings.TrimSpace(evidence.MetadataJSON) == "" {
		evidence.MetadataJSON = "{}"
	}
	now := nowUTC()
	if evidence.CreatedAt.IsZero() {
		evidence.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_evidence(id, agent_step_id, source_type, source_id, title, snippet, score, metadata_json, created_at) VALUES(?,?,?,?,?,?,?,?,?)`, evidence.ID, evidence.AgentStepID, evidence.SourceType, evidence.SourceID, evidence.Title, evidence.Snippet, evidence.Score, evidence.MetadataJSON, formatTime(evidence.CreatedAt))
	if err != nil {
		return AgentEvidence{}, err
	}
	return evidence, nil
}

func (s *Store) ListAgentEvidence(ctx context.Context, agentRunID string) ([]AgentEvidence, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT ev.id, ev.agent_step_id, ev.source_type, ev.source_id, ev.title, ev.snippet, ev.score, ev.metadata_json, ev.created_at FROM agent_evidence ev JOIN agent_steps st ON st.id = ev.agent_step_id WHERE st.agent_run_id=? ORDER BY ev.created_at ASC`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AgentEvidence, 0, 8)
	for rows.Next() {
		var item AgentEvidence
		var createdRaw string
		if err := rows.Scan(&item.ID, &item.AgentStepID, &item.SourceType, &item.SourceID, &item.Title, &item.Snippet, &item.Score, &item.MetadataJSON, &createdRaw); err != nil {
			return nil, err
		}
		item.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateAgentEvaluation(ctx context.Context, evaluation AgentEvaluation) (AgentEvaluation, error) {
	if evaluation.ID == "" {
		evaluation.ID = uuid.NewString()
	}
	now := nowUTC()
	if evaluation.CreatedAt.IsZero() {
		evaluation.CreatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_evaluations(id, agent_step_id, evaluation_type, verdict, reason, next_action, created_at) VALUES(?,?,?,?,?,?,?)`, evaluation.ID, evaluation.AgentStepID, evaluation.EvaluationType, evaluation.Verdict, evaluation.Reason, evaluation.NextAction, formatTime(evaluation.CreatedAt))
	if err != nil {
		return AgentEvaluation{}, err
	}
	return evaluation, nil
}

func (s *Store) ListAgentEvaluations(ctx context.Context, agentRunID string) ([]AgentEvaluation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.id, e.agent_step_id, e.evaluation_type, e.verdict, e.reason, e.next_action, e.created_at FROM agent_evaluations e JOIN agent_steps st ON st.id = e.agent_step_id WHERE st.agent_run_id=? ORDER BY e.created_at ASC`, agentRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]AgentEvaluation, 0, 8)
	for rows.Next() {
		var item AgentEvaluation
		var createdRaw string
		if err := rows.Scan(&item.ID, &item.AgentStepID, &item.EvaluationType, &item.Verdict, &item.Reason, &item.NextAction, &createdRaw); err != nil {
			return nil, err
		}
		item.CreatedAt, err = parseTime(createdRaw)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) nextAgentStepIndex(ctx context.Context, agentRunID string) int {
	var currentMax int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(step_index), 0) FROM agent_steps WHERE agent_run_id=?`, agentRunID).Scan(&currentMax); err != nil || currentMax < 0 {
		return 1
	}
	return currentMax + 1
}

func scanAgentRun(row rowScanner, run *AgentRun) error {
	var createdRaw, updatedRaw string
	var startedRaw, finishedRaw sql.NullString
	if err := row.Scan(&run.ID, &run.PipelineRunID, &run.ProjectID, &run.ChangeBatchID, &run.AgentName, &run.ModeName, &run.Status, &run.CurrentNode, &run.Summary, &startedRaw, &finishedRaw, &createdRaw, &updatedRaw); err != nil {
		return err
	}
	var err error
	run.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return err
	}
	run.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return err
	}
	if startedRaw.Valid {
		startedAt, parseErr := parseTime(startedRaw.String)
		if parseErr == nil {
			run.StartedAt = &startedAt
		}
	}
	if finishedRaw.Valid {
		finishedAt, parseErr := parseTime(finishedRaw.String)
		if parseErr == nil {
			run.FinishedAt = &finishedAt
		}
	}
	return nil
}

func scanAgentStep(row rowScanner, step *AgentStep) error {
	var createdRaw, updatedRaw string
	var startedRaw, finishedRaw sql.NullString
	if err := row.Scan(&step.ID, &step.AgentRunID, &step.StepIndex, &step.NodeName, &step.Title, &step.InputJSON, &step.OutputJSON, &step.DecisionSummary, &step.Status, &startedRaw, &finishedRaw, &createdRaw, &updatedRaw); err != nil {
		return err
	}
	var err error
	step.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return err
	}
	step.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return err
	}
	if startedRaw.Valid {
		startedAt, parseErr := parseTime(startedRaw.String)
		if parseErr == nil {
			step.StartedAt = &startedAt
		}
	}
	if finishedRaw.Valid {
		finishedAt, parseErr := parseTime(finishedRaw.String)
		if parseErr == nil {
			step.FinishedAt = &finishedAt
		}
	}
	return nil
}

func defaultJSONObject(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
}
