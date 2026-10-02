package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type practiceRecord struct {
	Summary    assessment.AttemptSummary
	Seal       assessment.Seal
	Answer     *assessment.Answer
	Correct    *bool
	TerminalAt *time.Time
}

func learningReadPractice(ctx context.Context, tx *sql.Tx, actor, id string, lock bool) (practiceRecord, error) {
	var p practiceRecord
	var raw, answer []byte
	var terminal sql.NullTime
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	e := tx.QueryRowContext(ctx, `SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,state,created_at,expires_at,terminal_at,seal_bytes,answer,correct FROM practice_attempts WHERE id=$1 AND owner_user_id=$2`+suffix, id, actor).Scan(&p.Summary.ID, &p.Summary.Knowledge.ID, &p.Summary.Knowledge.Version, &p.Summary.Knowledge.SHA256, &p.Summary.State, &p.Summary.CreatedAt, &p.Summary.ExpiresAt, &terminal, &raw, &answer, &p.Correct)
	if e != nil {
		return p, workflowRowError(e)
	}
	p.Summary.Kind = "practice"
	p.Summary.CreatedAt = p.Summary.CreatedAt.UTC()
	p.Summary.ExpiresAt = p.Summary.ExpiresAt.UTC()
	if terminal.Valid {
		t := terminal.Time.UTC()
		p.TerminalAt = &t
		if p.Summary.State == "answered" {
			p.Summary.SubmittedAt = &t
		}
	}
	p.Seal, e = learningDecodeSeal(raw)
	if e != nil {
		return p, auth.ErrUnavailable
	}
	if len(answer) > 0 {
		if json.Unmarshal(answer, &p.Answer) != nil {
			return p, auth.ErrUnavailable
		}
	}
	return p, nil
}
func learningSafeQuestions(seal assessment.Seal, items []question.Instance) ([]assessment.SafeQuestion, error) {
	out := make([]assessment.SafeQuestion, 0, len(items))
	if len(items) != len(seal.Items) {
		return out, auth.ErrUnavailable
	}
	for j, i := range items {
		q, e := assessment.ProjectQuestion(j+1, i)
		if e != nil {
			return out, e
		}
		if q.Knowledge.ID != seal.Knowledge.ID || q.Knowledge.Version != seal.Knowledge.Version || q.Instance != seal.Items[j].Instance {
			return out, auth.ErrUnavailable
		}
		q.Knowledge = seal.Knowledge
		out = append(out, q)
	}
	return out, nil
}
func learningPracticeView(ctx context.Context, tx *sql.Tx, actor string, p practiceRecord, now time.Time, cached []question.Instance) (assessment.PracticeView, error) {
	out := assessment.PracticeView{Summary: p.Summary}
	if out.Summary.State == "active" && !now.Before(out.Summary.ExpiresAt) {
		out.Summary.State = "expired"
	}
	if p.Summary.State == "answered" || p.Summary.State == "revealed" {
		blocked, e := learningActiveAssessmentItems(ctx, tx, actor, now)
		if e != nil {
			return out, e
		}
		for _, id := range blocked {
			if id == p.Seal.Items[0].Instance {
				return out, learning.ErrStateConflict
			}
		}
	}
	items := cached
	var e error
	if items == nil {
		items, e = learningLoadItems(ctx, tx, p.Seal)
		if e != nil {
			return out, e
		}
	}
	qs, e := learningSafeQuestions(p.Seal, items)
	if e != nil {
		return out, e
	}
	out.Question = qs[0]
	if p.Summary.State == "answered" || p.Summary.State == "revealed" {
		item := assessment.ResultItem{Question: out.Question, Answer: p.Answer, Correct: p.Correct, CorrectChoiceID: items[0].Body.CorrectChoiceID, CorrectNumeric: items[0].Body.CorrectNumeric, Explanation: &items[0].Body.Explanation, Validity: assessment.Effective, Reasons: []assessment.RestrictionReason{}}
		deps, e := learningEvidenceDependencies(ctx, tx, "practice", p.Summary.ID)
		if e != nil {
			return out, e
		}
		item.Reasons, e = learningEvidenceRestrictions(ctx, tx, deps)
		if e != nil {
			return out, e
		}
		if len(item.Reasons) > 0 {
			item.Validity = assessment.Restricted
			item.Correct = nil
			item.CorrectChoiceID = nil
			item.CorrectNumeric = nil
			item.Explanation = nil
		}
		out.Result = &assessment.PracticeResult{Outcome: assessment.PracticeState(p.Summary.State), Item: item}
		refs := []learning.ExposureRef{{Kind: "instance", Identity: out.Question.Instance}}
		if _, e = learningRecordExposure(ctx, tx, actor, refs, now); e != nil {
			return out, e
		}
	}
	return out, nil
}
func (s *Store) ReadPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	var out assessment.PracticeView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.ReadPracticeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadPractice(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		out, e = learningPracticeView(ctx, tx, u.ID, p, now, nil)
		return e
	})
	if e != nil {
		return assessment.PracticeView{}, e
	}
	return out, nil
}
