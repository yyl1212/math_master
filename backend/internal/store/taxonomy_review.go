package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

func (s *Store) ReadSubmissionTopics(ctx context.Context, a publication.Access, id string) (taxonomy.DraftTopicView, error) {
	var out taxonomy.DraftTopicView
	e := s.workflowReadTx(ctx, a, publication.ReadSubmissionAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if _, e := s.readWorkflowSubmission(ctx, tx, u, id, false); e != nil {
			return e
		}
		var raw []byte
		if e := tx.QueryRowContext(ctx, "SELECT body FROM taxonomy_submission_assignments WHERE submission_id=$1", id).Scan(&raw); e != nil {
			return workflowRowError(e)
		}
		if json.Unmarshal(raw, &out) != nil {
			return taxonomy.ErrInvalid
		}
		return nil
	})
	return out, e
}
func bindTopicReviewTx(ctx context.Context, tx *sql.Tx, submissionID, reviewID string) error {
	if e := taxonomyConfigured(ctx, tx); errors.Is(e, taxonomy.ErrNotConfigured) {
		return nil
	} else if e != nil {
		return e
	}
	var digest string
	e := tx.QueryRowContext(ctx, "SELECT digest FROM taxonomy_submission_assignments WHERE submission_id=$1", submissionID).Scan(&digest)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_review_bindings(decision_id,submission_id,digest) VALUES($1,$2,$3)", reviewID, submissionID, digest)
	return e
}
