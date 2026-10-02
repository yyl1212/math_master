package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
	"time"
)

type assessmentRecord struct {
	Summary          assessment.AttemptSummary
	Seal             assessment.Seal
	CreationSequence int64
}

func learningReadAssessment(ctx context.Context, tx *sql.Tx, actor, id string, lock bool) (assessmentRecord, error) {
	var p assessmentRecord
	var raw []byte
	var terminal sql.NullTime
	var mode assessment.Mode
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	e := tx.QueryRowContext(ctx, `SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,mode,state,created_at,expires_at,terminal_at,seal_bytes,creation_exposure_sequence FROM assessment_attempts WHERE id=$1 AND owner_user_id=$2 AND sealed`+suffix, id, actor).Scan(&p.Summary.ID, &p.Summary.Knowledge.ID, &p.Summary.Knowledge.Version, &p.Summary.Knowledge.SHA256, &mode, &p.Summary.State, &p.Summary.CreatedAt, &p.Summary.ExpiresAt, &terminal, &raw, &p.CreationSequence)
	if e != nil {
		return p, workflowRowError(e)
	}
	p.Summary.Kind = "assessment"
	p.Summary.Mode = &mode
	p.Summary.CreatedAt = p.Summary.CreatedAt.UTC()
	p.Summary.ExpiresAt = p.Summary.ExpiresAt.UTC()
	if p.Summary.State == "submitted" && terminal.Valid {
		t := terminal.Time.UTC()
		p.Summary.SubmittedAt = &t
	}
	p.Seal, e = learningDecodeSeal(raw)
	if e != nil {
		return p, auth.ErrUnavailable
	}
	return p, nil
}
func learningAssessmentView(ctx context.Context, tx *sql.Tx, p assessmentRecord, now time.Time, cached []question.Instance) (assessment.AttemptView, error) {
	out := assessment.AttemptView{Summary: p.Summary, Questions: []assessment.SafeQuestion{}}
	if out.Summary.State == "active" && !now.Before(out.Summary.ExpiresAt) {
		out.Summary.State = "expired"
	}
	items := cached
	var e error
	if items == nil {
		items, e = learningLoadItems(ctx, tx, p.Seal)
		if e != nil {
			return out, e
		}
	}
	out.Questions, e = learningSafeQuestions(p.Seal, items)
	return out, e
}
func learningHasReason(rs []assessment.RestrictionReason, r assessment.RestrictionReason) bool {
	for _, x := range rs {
		if x == r {
			return true
		}
	}
	return false
}
func learningUniqueReasons(rs []assessment.RestrictionReason) []assessment.RestrictionReason {
	out := []assessment.RestrictionReason{}
	for _, r := range rs {
		if !learningHasReason(out, r) {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func learningItemDependencies(seal assessment.Seal, b assessment.ItemBinding) []learning.EvidenceDependency {
	v := func(n int) *int { return &n }
	out := []learning.EvidenceDependency{{Kind: "knowledge", ID: seal.Knowledge.ID, Version: v(seal.Knowledge.Version), SHA256: seal.Knowledge.SHA256}, {Kind: "instance", ID: b.Instance.ID, Version: v(b.Instance.Version), SHA256: b.Instance.SHA256}}
	if seal.Blueprint != nil {
		out = append(out, learning.EvidenceDependency{Kind: "blueprint", ID: seal.Blueprint.ID, Version: v(seal.Blueprint.Version), SHA256: seal.Blueprint.SHA256})
	}
	if b.Template != nil {
		out = append(out, learning.EvidenceDependency{Kind: "template", ID: b.Template.ID, Version: v(b.Template.Version), SHA256: b.Template.SHA256})
	}
	for _, u := range b.Units {
		out = append(out, learning.EvidenceDependency{Kind: "unit", ID: u.ID, Version: v(u.Version), SHA256: u.SHA256})
	}
	for _, a := range b.Assets {
		out = append(out, learning.EvidenceDependency{Kind: "asset", ID: a.ID, SHA256: a.SHA256})
	}
	return out
}
func learningAssessmentResult(ctx context.Context, tx *sql.Tx, actor string, p assessmentRecord, now time.Time, cached []question.Instance) (assessment.ResultView, error) {
	out := assessment.ResultView{Summary: p.Summary, RuleVersion: 1, Validity: assessment.Effective, Reasons: []assessment.RestrictionReason{}, Items: []assessment.ResultItem{}}
	if p.Summary.State != "submitted" {
		return out, learning.ErrStateConflict
	}
	var reasons, progress []byte
	e := tx.QueryRowContext(ctx, `SELECT outcome,score,passed,original_reasons,progress FROM assessment_results WHERE attempt_id=$1 AND owner_user_id=$2 AND sealed`, p.Summary.ID, actor).Scan(&out.Outcome, &out.Score, &out.Passed, &reasons, &progress)
	if e != nil {
		return out, workflowRowError(e)
	}
	if json.Unmarshal(reasons, &out.Reasons) != nil || json.Unmarshal(progress, &out.Progress) != nil {
		return out, auth.ErrUnavailable
	}
	deps, e := learningEvidenceDependencies(ctx, tx, "assessment", p.Summary.ID)
	if e != nil {
		return out, e
	}
	rs, e := learningEvidenceRestrictions(ctx, tx, deps)
	if e != nil {
		return out, e
	}
	out.Reasons = learningUniqueReasons(append(out.Reasons, rs...))
	current, e := learningIsCurrent(ctx, tx, p.Seal.Knowledge)
	if e != nil {
		return out, e
	}
	if !current && !learningHasReason(out.Reasons, assessment.KnowledgeWithdrawn) {
		out.Reasons = learningUniqueReasons(append(out.Reasons, assessment.KnowledgeUpdated))
	}
	if len(rs) > 0 || out.Outcome == assessment.Affected {
		out.Validity = assessment.Restricted
	} else if !current {
		out.Validity = assessment.Stale
	}
	items := cached
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
	rows, e := tx.QueryContext(ctx, `SELECT position,answer FROM assessment_answers WHERE attempt_id=$1 AND owner_user_id=$2 ORDER BY position`, p.Summary.ID, actor)
	if e != nil {
		return out, e
	}
	answers := map[int]assessment.Answer{}
	for rows.Next() {
		var n int
		var raw []byte
		var a assessment.Answer
		if e = rows.Scan(&n, &raw); e != nil {
			rows.Close()
			return out, e
		}
		if json.Unmarshal(raw, &a) != nil {
			rows.Close()
			return out, auth.ErrUnavailable
		}
		answers[n] = a
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if len(answers) != 5 {
		return out, auth.ErrUnavailable
	}
	refs := []learning.ExposureRef{}
	for j, i := range items {
		a := answers[j+1]
		item := assessment.ResultItem{Question: qs[j], Answer: &a, CorrectChoiceID: i.Body.CorrectChoiceID, CorrectNumeric: i.Body.CorrectNumeric, Explanation: &i.Body.Explanation, Validity: assessment.Effective, Reasons: []assessment.RestrictionReason{}}
		if out.Outcome != assessment.Affected {
			correct, e := assessment.GradeAnswer(i, a)
			if e != nil {
				return out, e
			}
			item.Correct = &correct
		}
		item.Reasons, e = learningEvidenceRestrictions(ctx, tx, learningItemDependencies(p.Seal, p.Seal.Items[j]))
		if e != nil {
			return out, e
		}
		if len(item.Reasons) > 0 {
			item.Validity = assessment.Restricted
			item.Correct = nil
			item.CorrectChoiceID = nil
			item.CorrectNumeric = nil
			item.Explanation = nil
		} else if !current {
			item.Validity = assessment.Stale
			item.Reasons = []assessment.RestrictionReason{assessment.KnowledgeUpdated}
		}
		out.Items = append(out.Items, item)
		refs = append(refs, learning.ExposureRef{Kind: "instance", Identity: qs[j].Instance})
	}
	// Its own terminal answer delivery is recorded after the eligibility decision.
	if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
		return out, e
	}
	_, e = learningRecordExposure(ctx, tx, actor, refs, now)
	return out, e
}
func (s *Store) ReadAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error) {
	var out assessment.AttemptView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.ReadAssessmentAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadAssessment(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		out, e = learningAssessmentView(ctx, tx, p, now, nil)
		return e
	})
	if e != nil {
		return assessment.AttemptView{}, e
	}
	return out, nil
}
