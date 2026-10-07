package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/learning"
)

func feedbackDiscussionExposure(ctx context.Context, tx *sql.Tx, reader string, b feedback.Binding) error {
	if e := feedbackSourceConfigured(ctx, tx, b); e != nil {
		return e
	}
	if b.Instance == nil {
		return nil
	}
	enabled, e := learningConfigured(ctx, tx)
	if e != nil || !enabled {
		return feedback.ErrNotConfigured
	}
	var sequence int64
	e = tx.QueryRowContext(ctx, `SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$1 FOR UPDATE`, reader).Scan(&sequence)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return e
	}
	now, e := dbClock(ctx, tx)
	if e != nil {
		return e
	}
	var template any
	if b.Template != nil {
		template = body(b.Template)
	}
	var overlap bool
	e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assessment_attempts a JOIN assessment_items i ON i.attempt_id=a.id WHERE a.owner_user_id=$1 AND a.state='active' AND a.sealed AND a.expires_at>$2 AND (i.binding->'instance'=$3::jsonb OR $4::jsonb IS NOT NULL AND i.binding->'template'=$4::jsonb))`, reader, now, body(b.Instance), template).Scan(&overlap)
	if e != nil {
		return e
	}
	if overlap {
		return feedback.ErrAnswerOverlap
	}
	refs := []learning.ExposureRef{{Kind: "instance", Identity: *b.Instance}}
	if b.Template != nil {
		refs = append(refs, learning.ExposureRef{Kind: "template", Identity: *b.Template})
	}
	_, e = learningRecordExposure(ctx, tx, reader, refs, now)
	return e
}
