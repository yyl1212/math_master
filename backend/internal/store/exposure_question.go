package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// Canonicalize exactly as the persisted P4a package, without rerunning generators.
func questionExposureRefs(input question.DraftInput, instances []question.Instance) ([]learning.ExposureRef, error) {
	out := []learning.ExposureRef{}
	raw, _, e := question.CanonicalPackage(input.QuestionPackage)
	if e != nil {
		return out, e
	}
	var env struct {
		Body question.QuestionPackage `json:"body"`
	}
	if json.Unmarshal(raw, &env) != nil {
		return out, auth.ErrUnavailable
	}
	for _, t := range env.Body.Templates {
		_, sha, e := questionCanonical("question-template-v1", t)
		if e != nil {
			return out, e
		}
		out = append(out, learning.ExposureRef{Kind: "template", Identity: question.Identity{ID: t.ID, Version: t.Version, SHA256: sha}})
	}
	for _, f := range env.Body.FixedQuestions {
		instances = append(instances, question.Instance{Identity: question.Identity{ID: f.ID, Version: f.Version}, Origin: "fixed", Parameters: []question.ParameterValue{}, Body: f.Body})
	}
	for _, i := range instances {
		_, sha, e := question.CanonicalInstance(i)
		if e != nil {
			return out, e
		}
		if i.Identity.SHA256 != "" && sha != i.Identity.SHA256 {
			return out, auth.ErrUnavailable
		}
		i.Identity.SHA256 = sha
		out = append(out, learning.ExposureRef{Kind: "instance", Identity: i.Identity})
	}
	return out, nil
}
func questionExposeResponse(ctx context.Context, tx *sql.Tx, u auth.User, response any, now time.Time) error {
	enabled, e := learningConfigured(ctx, tx)
	if e != nil || !enabled {
		return e
	}
	var input question.DraftInput
	var instances []question.Instance
	switch r := response.(type) {
	case question.DraftView:
		input = questionDraftInputOf(r)
	case question.SubmissionView:
		input = question.DraftInput{QuestionPackage: r.Frozen.QuestionPackage}
	case question.Page[question.Instance]:
		instances = r.Items
	default:
		return nil
	}
	refs, e := questionExposureRefs(input, instances)
	if e != nil {
		return e
	}
	_, e = learningRecordExposure(ctx, tx, u.ID, refs, now)
	return e
}
func questionExposeNow(ctx context.Context, tx *sql.Tx, u auth.User, response any) error {
	var now time.Time
	if e := tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
		return e
	}
	return questionExposeResponse(ctx, tx, u, response, now)
}

func (s *Store) questionAnswerReadTx(ctx context.Context, a question.Access, action question.Action, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return questionError(e)
	}
	defer tx.Rollback()
	if e = questionConfigured(ctx, tx); e != nil {
		return questionError(e)
	}
	if _, e = learningConfigured(ctx, tx); e != nil {
		return questionError(e)
	}
	if e = learningLocks(ctx, tx); e != nil {
		return questionError(e)
	}
	u, now, e := questionIdentity(ctx, tx, a, action, true, nil)
	if e != nil {
		return questionError(e)
	}
	if e = fn(ctx, tx, u, now); e != nil {
		return questionError(e)
	}
	if _, _, e = questionIdentity(ctx, tx, a, action, false, nil); e != nil {
		return questionError(e)
	}
	return questionError(tx.Commit())
}
