package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type followupRowScanner interface {
	Scan(dest ...any) error
}

func normalizeResidualStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "resolved":
		return "resolved"
	case "waived":
		return "waived"
	default:
		return "open"
	}
}

func normalizeApprovalGateStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "resolved") {
		return "resolved"
	}
	return "open"
}

func (s *Store) UpsertResidualItem(ctx context.Context, item ResidualItem) (ResidualItem, error) {
	if strings.TrimSpace(item.SourceKey) == "" {
		return ResidualItem{}, errors.New("source_key is required")
	}
	now := nowUTC()
	status := normalizeResidualStatus(item.Status)
	existing, err := s.GetResidualItemBySourceKey(ctx, item.SourceKey)
	if err == nil {
		item.ID = existing.ID
		item.CreatedAt = existing.CreatedAt
		if existing.Status == "waived" && status == "open" {
			status = "waived"
		}
		item.Status = status
		item.UpdatedAt = now
		if item.Status == "resolved" || item.Status == "waived" {
			if existing.ResolvedAt != nil {
				item.ResolvedAt = existing.ResolvedAt
			} else {
				item.ResolvedAt = &now
			}
		} else {
			item.ResolvedAt = nil
		}
		_, execErr := s.db.ExecContext(
			ctx,
			`UPDATE residual_items
			 SET project_id=?, pipeline_run_id=?, agent_run_id=?, stage=?, category=?, severity=?,
			     summary=?, evidence=?, suggested_command=?, status=?, resolution_note=?, resolved_at=?, updated_at=?
			 WHERE id=?`,
			item.ProjectID,
			item.PipelineRunID,
			item.AgentRunID,
			item.Stage,
			item.Category,
			item.Severity,
			item.Summary,
			item.Evidence,
			item.SuggestedCommand,
			item.Status,
			item.ResolutionNote,
			nullableTime(item.ResolvedAt),
			formatTime(item.UpdatedAt),
			item.ID,
		)
		if execErr != nil {
			return ResidualItem{}, execErr
		}
		return item, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return ResidualItem{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.Status = status
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if item.Status == "resolved" || item.Status == "waived" {
		if item.ResolvedAt == nil {
			item.ResolvedAt = &now
		}
	} else {
		item.ResolvedAt = nil
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO residual_items(
			id, project_id, pipeline_run_id, agent_run_id, stage, category, severity,
			summary, evidence, suggested_command, source_key, status, resolution_note,
			resolved_at, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		item.ID,
		item.ProjectID,
		item.PipelineRunID,
		item.AgentRunID,
		item.Stage,
		item.Category,
		item.Severity,
		item.Summary,
		item.Evidence,
		item.SuggestedCommand,
		item.SourceKey,
		item.Status,
		item.ResolutionNote,
		nullableTime(item.ResolvedAt),
		formatTime(item.CreatedAt),
		formatTime(item.UpdatedAt),
	)
	if err != nil {
		return ResidualItem{}, err
	}
	if touchErr := s.TouchPipelineRun(ctx, item.PipelineRunID); touchErr != nil {
		return ResidualItem{}, touchErr
	}
	return item, nil
}

func (s *Store) GetResidualItem(ctx context.Context, id string) (ResidualItem, error) {
	var item ResidualItem
	if err := scanResidualItem(s.db.QueryRowContext(ctx, `SELECT id, project_id, pipeline_run_id, agent_run_id, stage, category, severity, summary, evidence, suggested_command, source_key, status, resolution_note, resolved_at, created_at, updated_at FROM residual_items WHERE id=?`, id), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResidualItem{}, ErrNotFound
		}
		return ResidualItem{}, err
	}
	return item, nil
}

func (s *Store) GetResidualItemBySourceKey(ctx context.Context, sourceKey string) (ResidualItem, error) {
	var item ResidualItem
	if err := scanResidualItem(s.db.QueryRowContext(ctx, `SELECT id, project_id, pipeline_run_id, agent_run_id, stage, category, severity, summary, evidence, suggested_command, source_key, status, resolution_note, resolved_at, created_at, updated_at FROM residual_items WHERE source_key=?`, sourceKey), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ResidualItem{}, ErrNotFound
		}
		return ResidualItem{}, err
	}
	return item, nil
}

func (s *Store) ListResidualItems(ctx context.Context, projectID, pipelineRunID string, limit int) ([]ResidualItem, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, project_id, pipeline_run_id, agent_run_id, stage, category, severity, summary, evidence, suggested_command, source_key, status, resolution_note, resolved_at, created_at, updated_at FROM residual_items WHERE project_id=?`
	args := []any{projectID}
	if strings.TrimSpace(pipelineRunID) != "" {
		query += ` AND pipeline_run_id=?`
		args = append(args, pipelineRunID)
	}
	query += ` ORDER BY CASE status WHEN 'open' THEN 0 WHEN 'waived' THEN 1 ELSE 2 END, updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ResidualItem, 0, 8)
	for rows.Next() {
		var item ResidualItem
		if err := scanResidualItem(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpdateResidualItemStatus(ctx context.Context, id, status, resolutionNote string) (ResidualItem, error) {
	item, err := s.GetResidualItem(ctx, id)
	if err != nil {
		return ResidualItem{}, err
	}
	item.Status = normalizeResidualStatus(status)
	item.ResolutionNote = strings.TrimSpace(resolutionNote)
	now := nowUTC()
	item.UpdatedAt = now
	if item.Status == "resolved" || item.Status == "waived" {
		item.ResolvedAt = &now
	} else {
		item.ResolvedAt = nil
	}
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE residual_items SET status=?, resolution_note=?, resolved_at=?, updated_at=? WHERE id=?`,
		item.Status,
		item.ResolutionNote,
		nullableTime(item.ResolvedAt),
		formatTime(item.UpdatedAt),
		item.ID,
	)
	if err != nil {
		return ResidualItem{}, err
	}
	return item, nil
}

func (s *Store) UpsertApprovalGate(ctx context.Context, gate ApprovalGate) (ApprovalGate, error) {
	if strings.TrimSpace(gate.SourceKey) == "" {
		return ApprovalGate{}, errors.New("source_key is required")
	}
	now := nowUTC()
	status := normalizeApprovalGateStatus(gate.Status)
	existing, err := s.GetApprovalGateBySourceKey(ctx, gate.SourceKey)
	if err == nil {
		gate.ID = existing.ID
		gate.CreatedAt = existing.CreatedAt
		gate.Status = status
		gate.UpdatedAt = now
		if gate.Status == "resolved" {
			if existing.ResolvedAt != nil {
				gate.ResolvedAt = existing.ResolvedAt
			} else {
				gate.ResolvedAt = &now
			}
		} else {
			gate.ResolvedAt = nil
		}
		_, execErr := s.db.ExecContext(
			ctx,
			`UPDATE approval_gates
			 SET project_id=?, pipeline_run_id=?, change_batch_id=?, gate_type=?, title=?, detail=?, tool_name=?, risk_level=?, status=?, resolved_at=?, updated_at=?
			 WHERE id=?`,
			gate.ProjectID,
			gate.PipelineRunID,
			gate.ChangeBatchID,
			gate.GateType,
			gate.Title,
			gate.Detail,
			gate.ToolName,
			gate.RiskLevel,
			gate.Status,
			nullableTime(gate.ResolvedAt),
			formatTime(gate.UpdatedAt),
			gate.ID,
		)
		if execErr != nil {
			return ApprovalGate{}, execErr
		}
		return gate, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return ApprovalGate{}, err
	}
	if gate.ID == "" {
		gate.ID = uuid.NewString()
	}
	gate.Status = status
	if gate.CreatedAt.IsZero() {
		gate.CreatedAt = now
	}
	gate.UpdatedAt = now
	if gate.Status == "resolved" {
		if gate.ResolvedAt == nil {
			gate.ResolvedAt = &now
		}
	} else {
		gate.ResolvedAt = nil
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO approval_gates(
			id, project_id, pipeline_run_id, change_batch_id, gate_type, title, detail,
			tool_name, risk_level, source_key, status, resolved_at, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		gate.ID,
		gate.ProjectID,
		gate.PipelineRunID,
		gate.ChangeBatchID,
		gate.GateType,
		gate.Title,
		gate.Detail,
		gate.ToolName,
		gate.RiskLevel,
		gate.SourceKey,
		gate.Status,
		nullableTime(gate.ResolvedAt),
		formatTime(gate.CreatedAt),
		formatTime(gate.UpdatedAt),
	)
	if err != nil {
		return ApprovalGate{}, err
	}
	return gate, nil
}

func (s *Store) GetApprovalGateBySourceKey(ctx context.Context, sourceKey string) (ApprovalGate, error) {
	var gate ApprovalGate
	if err := scanApprovalGate(s.db.QueryRowContext(ctx, `SELECT id, project_id, pipeline_run_id, change_batch_id, gate_type, title, detail, tool_name, risk_level, source_key, status, resolved_at, created_at, updated_at FROM approval_gates WHERE source_key=?`, sourceKey), &gate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ApprovalGate{}, ErrNotFound
		}
		return ApprovalGate{}, err
	}
	return gate, nil
}

func (s *Store) ListApprovalGates(ctx context.Context, projectID, pipelineRunID string, limit int) ([]ApprovalGate, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, project_id, pipeline_run_id, change_batch_id, gate_type, title, detail, tool_name, risk_level, source_key, status, resolved_at, created_at, updated_at FROM approval_gates WHERE project_id=?`
	args := []any{projectID}
	if strings.TrimSpace(pipelineRunID) != "" {
		query += ` AND pipeline_run_id=?`
		args = append(args, pipelineRunID)
	}
	query += ` ORDER BY CASE status WHEN 'open' THEN 0 ELSE 1 END, updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]ApprovalGate, 0, 8)
	for rows.Next() {
		var item ApprovalGate
		if err := scanApprovalGate(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanResidualItem(scanner followupRowScanner, item *ResidualItem) error {
	var resolvedRaw sql.NullString
	var createdRaw, updatedRaw string
	if err := scanner.Scan(
		&item.ID,
		&item.ProjectID,
		&item.PipelineRunID,
		&item.AgentRunID,
		&item.Stage,
		&item.Category,
		&item.Severity,
		&item.Summary,
		&item.Evidence,
		&item.SuggestedCommand,
		&item.SourceKey,
		&item.Status,
		&item.ResolutionNote,
		&resolvedRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return err
	}
	var parseErr error
	item.CreatedAt, parseErr = parseTime(createdRaw)
	if parseErr != nil {
		return fmt.Errorf("parse residual item created_at: %w", parseErr)
	}
	item.UpdatedAt, parseErr = parseTime(updatedRaw)
	if parseErr != nil {
		return fmt.Errorf("parse residual item updated_at: %w", parseErr)
	}
	if resolvedRaw.Valid && strings.TrimSpace(resolvedRaw.String) != "" {
		resolvedAt, err := parseTime(resolvedRaw.String)
		if err != nil {
			return fmt.Errorf("parse residual item resolved_at: %w", err)
		}
		item.ResolvedAt = &resolvedAt
	}
	return nil
}

func scanApprovalGate(scanner followupRowScanner, gate *ApprovalGate) error {
	var resolvedRaw sql.NullString
	var createdRaw, updatedRaw string
	if err := scanner.Scan(
		&gate.ID,
		&gate.ProjectID,
		&gate.PipelineRunID,
		&gate.ChangeBatchID,
		&gate.GateType,
		&gate.Title,
		&gate.Detail,
		&gate.ToolName,
		&gate.RiskLevel,
		&gate.SourceKey,
		&gate.Status,
		&resolvedRaw,
		&createdRaw,
		&updatedRaw,
	); err != nil {
		return err
	}
	var parseErr error
	gate.CreatedAt, parseErr = parseTime(createdRaw)
	if parseErr != nil {
		return fmt.Errorf("parse approval gate created_at: %w", parseErr)
	}
	gate.UpdatedAt, parseErr = parseTime(updatedRaw)
	if parseErr != nil {
		return fmt.Errorf("parse approval gate updated_at: %w", parseErr)
	}
	if resolvedRaw.Valid && strings.TrimSpace(resolvedRaw.String) != "" {
		resolvedAt, err := parseTime(resolvedRaw.String)
		if err != nil {
			return fmt.Errorf("parse approval gate resolved_at: %w", err)
		}
		gate.ResolvedAt = &resolvedAt
	}
	return nil
}
