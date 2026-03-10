package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type previewRowScanner interface {
	Scan(dest ...any) error
}

func normalizePreviewSessionStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "accepted":
		return "accepted"
	case "rejected":
		return "rejected"
	default:
		return "generated"
	}
}

func (s *Store) UpsertPreviewSession(ctx context.Context, session PreviewSession) (PreviewSession, error) {
	if strings.TrimSpace(session.SourceKey) == "" {
		return PreviewSession{}, errors.New("source_key is required")
	}
	now := nowUTC()
	status := normalizePreviewSessionStatus(session.Status)
	existing, err := s.GetPreviewSessionBySourceKey(ctx, session.SourceKey)
	if err == nil {
		session.ID = existing.ID
		session.CreatedAt = existing.CreatedAt
		if existing.Status == "accepted" && status == "generated" {
			status = "accepted"
		}
		session.Status = status
		session.UpdatedAt = now
		if session.Status == "accepted" || session.Status == "rejected" {
			if existing.ReviewedAt != nil {
				session.ReviewedAt = existing.ReviewedAt
			} else {
				session.ReviewedAt = &now
			}
		} else {
			session.ReviewedAt = nil
		}
		_, execErr := s.db.ExecContext(
			ctx,
			`UPDATE preview_sessions
			 SET project_id=?, pipeline_run_id=?, change_batch_id=?, preview_url=?, preview_type=?, title=?, status=?, reviewer_note=?, reviewed_at=?, updated_at=?
			 WHERE id=?`,
			session.ProjectID,
			session.PipelineRunID,
			session.ChangeBatchID,
			session.PreviewURL,
			session.PreviewType,
			session.Title,
			session.Status,
			session.ReviewerNote,
			nullableTime(session.ReviewedAt),
			formatTime(session.UpdatedAt),
			session.ID,
		)
		if execErr != nil {
			return PreviewSession{}, execErr
		}
		return session, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return PreviewSession{}, err
	}
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	session.Status = status
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	if session.Status == "accepted" || session.Status == "rejected" {
		if session.ReviewedAt == nil {
			session.ReviewedAt = &now
		}
	} else {
		session.ReviewedAt = nil
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO preview_sessions(
			id, project_id, pipeline_run_id, change_batch_id, preview_url, preview_type, title,
			source_key, status, reviewer_note, reviewed_at, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		session.ID,
		session.ProjectID,
		session.PipelineRunID,
		session.ChangeBatchID,
		session.PreviewURL,
		session.PreviewType,
		session.Title,
		session.SourceKey,
		session.Status,
		session.ReviewerNote,
		nullableTime(session.ReviewedAt),
		formatTime(session.CreatedAt),
		formatTime(session.UpdatedAt),
	)
	if err != nil {
		return PreviewSession{}, err
	}
	return session, nil
}

func (s *Store) GetPreviewSessionBySourceKey(ctx context.Context, sourceKey string) (PreviewSession, error) {
	var item PreviewSession
	if err := scanPreviewSession(s.db.QueryRowContext(ctx, `SELECT id, project_id, pipeline_run_id, change_batch_id, preview_url, preview_type, title, source_key, status, reviewer_note, reviewed_at, created_at, updated_at FROM preview_sessions WHERE source_key=?`, sourceKey), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreviewSession{}, ErrNotFound
		}
		return PreviewSession{}, err
	}
	return item, nil
}

func (s *Store) GetPreviewSession(ctx context.Context, id string) (PreviewSession, error) {
	var item PreviewSession
	if err := scanPreviewSession(s.db.QueryRowContext(ctx, `SELECT id, project_id, pipeline_run_id, change_batch_id, preview_url, preview_type, title, source_key, status, reviewer_note, reviewed_at, created_at, updated_at FROM preview_sessions WHERE id=?`, id), &item); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PreviewSession{}, ErrNotFound
		}
		return PreviewSession{}, err
	}
	return item, nil
}

func (s *Store) ListPreviewSessions(ctx context.Context, projectID, pipelineRunID string, limit int) ([]PreviewSession, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `SELECT id, project_id, pipeline_run_id, change_batch_id, preview_url, preview_type, title, source_key, status, reviewer_note, reviewed_at, created_at, updated_at FROM preview_sessions WHERE project_id=?`
	args := []any{projectID}
	if strings.TrimSpace(pipelineRunID) != "" {
		query += ` AND pipeline_run_id=?`
		args = append(args, pipelineRunID)
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]PreviewSession, 0, 8)
	for rows.Next() {
		var item PreviewSession
		if err := scanPreviewSession(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpdatePreviewSessionStatus(ctx context.Context, id, status, reviewerNote string) (PreviewSession, error) {
	item, err := s.GetPreviewSession(ctx, id)
	if err != nil {
		return PreviewSession{}, err
	}
	item.Status = normalizePreviewSessionStatus(status)
	item.ReviewerNote = strings.TrimSpace(reviewerNote)
	now := nowUTC()
	item.UpdatedAt = now
	if item.Status == "accepted" || item.Status == "rejected" {
		item.ReviewedAt = &now
	} else {
		item.ReviewedAt = nil
	}
	_, err = s.db.ExecContext(
		ctx,
		`UPDATE preview_sessions SET status=?, reviewer_note=?, reviewed_at=?, updated_at=? WHERE id=?`,
		item.Status,
		item.ReviewerNote,
		nullableTime(item.ReviewedAt),
		formatTime(item.UpdatedAt),
		item.ID,
	)
	if err != nil {
		return PreviewSession{}, err
	}
	if touchErr := s.TouchPipelineRun(ctx, item.PipelineRunID); touchErr != nil {
		return PreviewSession{}, touchErr
	}
	return item, nil
}

func scanPreviewSession(row previewRowScanner, item *PreviewSession) error {
	var reviewedRaw sql.NullString
	var createdRaw, updatedRaw string
	if err := row.Scan(&item.ID, &item.ProjectID, &item.PipelineRunID, &item.ChangeBatchID, &item.PreviewURL, &item.PreviewType, &item.Title, &item.SourceKey, &item.Status, &item.ReviewerNote, &reviewedRaw, &createdRaw, &updatedRaw); err != nil {
		return err
	}
	var err error
	item.CreatedAt, err = parseTime(createdRaw)
	if err != nil {
		return err
	}
	item.UpdatedAt, err = parseTime(updatedRaw)
	if err != nil {
		return err
	}
	if reviewedRaw.Valid {
		reviewedAt, parseErr := parseTime(reviewedRaw.String)
		if parseErr == nil {
			item.ReviewedAt = &reviewedAt
		}
	}
	return nil
}
