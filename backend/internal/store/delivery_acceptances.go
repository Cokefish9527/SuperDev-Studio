package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type deliveryAcceptanceRowScanner interface {
	Scan(dest ...any) error
}

func normalizeDeliveryAcceptanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "revoked":
		return "revoked"
	default:
		return "accepted"
	}
}

func (s *Store) GetDeliveryAcceptanceByRun(ctx context.Context, pipelineRunID string) (DeliveryAcceptance, error) {
	var item DeliveryAcceptance
	if err := scanDeliveryAcceptance(
		s.db.QueryRowContext(
			ctx,
			`SELECT id, project_id, pipeline_run_id, change_batch_id, status, reviewer_note, reviewed_at, created_at, updated_at FROM delivery_acceptances WHERE pipeline_run_id=?`,
			pipelineRunID,
		),
		&item,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeliveryAcceptance{}, ErrNotFound
		}
		return DeliveryAcceptance{}, err
	}
	return item, nil
}

func (s *Store) UpsertDeliveryAcceptance(ctx context.Context, acceptance DeliveryAcceptance) (DeliveryAcceptance, error) {
	if strings.TrimSpace(acceptance.ProjectID) == "" {
		return DeliveryAcceptance{}, errors.New("project_id is required")
	}
	if strings.TrimSpace(acceptance.PipelineRunID) == "" {
		return DeliveryAcceptance{}, errors.New("pipeline_run_id is required")
	}
	now := nowUTC()
	acceptance.Status = normalizeDeliveryAcceptanceStatus(acceptance.Status)
	acceptance.ReviewerNote = strings.TrimSpace(acceptance.ReviewerNote)
	acceptance.UpdatedAt = now
	acceptance.ReviewedAt = &now

	existing, err := s.GetDeliveryAcceptanceByRun(ctx, acceptance.PipelineRunID)
	if err == nil {
		acceptance.ID = existing.ID
		acceptance.CreatedAt = existing.CreatedAt
		_, execErr := s.db.ExecContext(
			ctx,
			`UPDATE delivery_acceptances
			 SET project_id=?, change_batch_id=?, status=?, reviewer_note=?, reviewed_at=?, updated_at=?
			 WHERE id=?`,
			acceptance.ProjectID,
			acceptance.ChangeBatchID,
			acceptance.Status,
			acceptance.ReviewerNote,
			nullableTime(acceptance.ReviewedAt),
			formatTime(acceptance.UpdatedAt),
			acceptance.ID,
		)
		if execErr != nil {
			return DeliveryAcceptance{}, execErr
		}
		if touchErr := s.TouchPipelineRun(ctx, acceptance.PipelineRunID); touchErr != nil {
			return DeliveryAcceptance{}, touchErr
		}
		return acceptance, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return DeliveryAcceptance{}, err
	}
	if acceptance.ID == "" {
		acceptance.ID = uuid.NewString()
	}
	if acceptance.CreatedAt.IsZero() {
		acceptance.CreatedAt = now
	}
	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO delivery_acceptances(
			id, project_id, pipeline_run_id, change_batch_id, status, reviewer_note, reviewed_at, created_at, updated_at
		) VALUES(?,?,?,?,?,?,?,?,?)`,
		acceptance.ID,
		acceptance.ProjectID,
		acceptance.PipelineRunID,
		acceptance.ChangeBatchID,
		acceptance.Status,
		acceptance.ReviewerNote,
		nullableTime(acceptance.ReviewedAt),
		formatTime(acceptance.CreatedAt),
		formatTime(acceptance.UpdatedAt),
	)
	if err != nil {
		return DeliveryAcceptance{}, err
	}
	if touchErr := s.TouchPipelineRun(ctx, acceptance.PipelineRunID); touchErr != nil {
		return DeliveryAcceptance{}, touchErr
	}
	return acceptance, nil
}

func scanDeliveryAcceptance(row deliveryAcceptanceRowScanner, item *DeliveryAcceptance) error {
	var reviewedRaw sql.NullString
	var createdRaw, updatedRaw string
	if err := row.Scan(&item.ID, &item.ProjectID, &item.PipelineRunID, &item.ChangeBatchID, &item.Status, &item.ReviewerNote, &reviewedRaw, &createdRaw, &updatedRaw); err != nil {
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
