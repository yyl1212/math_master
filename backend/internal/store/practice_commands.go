package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
	"time"
)

func learningSeed() ([32]byte, error) { var seed [32]byte; _, e := rand.Read(seed[:]); return seed, e }
func learningActiveAssessmentItems(ctx context.Context, tx *sql.Tx, actor string, now time.Time) ([]question.Identity, error) {
	out := []question.Identity{}
	rows, e := tx.QueryContext(ctx, `SELECT ai.instance_id,ai.instance_version,ai.instance_sha256 FROM assessment_attempts a JOIN assessment_items ai ON ai.attempt_id=a.id WHERE a.owner_user_id=$1 AND a.state='active' AND a.expires_at>$2 ORDER BY ai.position`, actor, now)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var id question.Identity
		if e = rows.Scan(&id.ID, &id.Version, &id.SHA256); e != nil {
			return out, e
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
func learningFreezeItems(ctx context.Context, tx *sql.Tx, pool assessment.SourcePool, ids []question.Identity) ([]assessment.ItemBinding, []question.Instance, error) {
	bindings := make([]assessment.ItemBinding, 0, len(ids))
	items := make([]question.Instance, 0, len(ids))
	if len(ids) != 1 && len(ids) != 5 {
		return bindings, items, auth.ErrInvalidInput
	}
	wanted := make([]map[string]any, 0, len(ids))
	positions := map[question.Identity]int{}
	cs := map[question.Identity]assessment.Candidate{}
	for _, c := range pool.Candidates {
		cs[c.Identity] = c
	}
	for j, id := range ids {
		if _, ok := cs[id]; !ok {
			return bindings, items, auth.ErrUnavailable
		}
		positions[id] = j
		wanted = append(wanted, map[string]any{"position": j + 1, "id": id.ID, "version": id.Version, "sha256": id.SHA256})
	}
	rows, e := tx.QueryContext(ctx, `SELECT i.id,i.version,i.sha256,i.body_bytes,m.evidence FROM jsonb_to_recordset($1::jsonb) r(position integer,id text,version integer,sha256 text) JOIN question_instances i ON i.id=r.id AND i.version=r.version AND i.sha256=r.sha256 AND i.sealed JOIN question_publication_members m ON m.publication_id=$2 AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 JOIN question_publications p ON p.id=m.publication_id AND p.sealed AND p.status='published' JOIN question_review_decisions rd ON rd.id=m.review_id AND rd.decision='approve' JOIN question_submissions s ON s.id=m.submission_id AND s.id=rd.submission_id AND s.status='approved' AND s.sealed AND s.frozen_digest=rd.frozen_digest JOIN question_submission_members sm ON sm.submission_id=s.id AND sm.kind='instance' AND sm.id=i.id AND sm.version=i.version AND sm.sha256=i.sha256 ORDER BY r.position`, body(wanted), pool.QuestionHead)
	if e != nil {
		return bindings, items, e
	}
	unitRefs := map[question.Ref]bool{}
	for rows.Next() {
		var id question.Identity
		var raw, proof []byte
		if e = rows.Scan(&id.ID, &id.Version, &id.SHA256, &raw, &proof); e != nil {
			rows.Close()
			return bindings, items, e
		}
		var env struct {
			Purpose string            `json:"purpose"`
			Body    question.Instance `json:"body"`
		}
		var approval question.MemberEvidence
		if json.Unmarshal(raw, &env) != nil || env.Purpose != "question-instance-body-v1" || json.Unmarshal(proof, &approval) != nil {
			rows.Close()
			return bindings, items, auth.ErrUnavailable
		}
		env.Body.Identity.SHA256 = id.SHA256
		_, sha, e := question.CanonicalInstance(env.Body)
		if e != nil || sha != id.SHA256 || env.Body.Identity != id {
			rows.Close()
			return bindings, items, auth.ErrUnavailable
		}
		c := cs[id]
		bindings = append(bindings, assessment.ItemBinding{Position: positions[id] + 1, Instance: id, Template: env.Body.Template, Coverage: append([]int{}, c.Coverage...), Units: []question.Identity{}, Assets: append([]question.AssetRef{}, env.Body.Body.Assets...), Approval: approval})
		items = append(items, env.Body)
		for _, u := range env.Body.Body.Units {
			unitRefs[u] = true
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return bindings, items, e
	}
	if len(items) != len(ids) {
		return bindings, items, auth.ErrNotFound
	}
	refs := []question.Ref{}
	for u := range unitRefs {
		refs = append(refs, u)
	}
	rows, e = tx.QueryContext(ctx, `SELECT u.id,u.version,u.sha256 FROM jsonb_to_recordset($1::jsonb) r(id text,version integer) JOIN unit_versions u ON u.id=r.id AND u.version=r.version JOIN publication_members m ON m.snapshot_id=$2 AND m.kind='unit' AND m.id=u.id AND m.version=u.version AND m.availability='active'`, body(refs), pool.KnowledgeHead)
	if e != nil {
		return bindings, items, e
	}
	units := map[question.Ref]question.Identity{}
	for rows.Next() {
		var u question.Identity
		if e = rows.Scan(&u.ID, &u.Version, &u.SHA256); e != nil {
			rows.Close()
			return bindings, items, e
		}
		units[question.Ref{ID: u.ID, Version: u.Version}] = u
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return bindings, items, e
	}
	for j, i := range items {
		for _, r := range i.Body.Units {
			u, ok := units[r]
			if !ok {
				return bindings, items, learning.ErrVersionStale
			}
			bindings[j].Units = append(bindings[j].Units, u)
		}
	}
	return bindings, items, nil
}
func learningPracticeSelection(cs []assessment.Candidate, blocked []question.Identity, seed [32]byte) (question.Identity, bool) {
	excluded := map[question.Identity]bool{}
	for _, id := range blocked {
		excluded[id] = true
	}
	eligible := []assessment.Candidate{}
	for _, c := range cs {
		if !excluded[c.Identity] {
			eligible = append(eligible, c)
		}
	}
	rank := func(c assessment.Candidate) [32]byte {
		return sha256.Sum256(append(append([]byte{}, seed[:]...), []byte(body(c.Identity))...))
	}
	sort.Slice(eligible, func(i, j int) bool {
		a, b := eligible[i], eligible[j]
		if a.Seen != b.Seen {
			return !a.Seen
		}
		if a.LastSeenAt == nil && b.LastSeenAt != nil {
			return true
		}
		if a.LastSeenAt != nil && b.LastSeenAt == nil {
			return false
		}
		if a.LastSeenAt != nil && !a.LastSeenAt.Equal(*b.LastSeenAt) {
			return a.LastSeenAt.Before(*b.LastSeenAt)
		}
		ar, br := rank(a), rank(b)
		return hex.EncodeToString(ar[:]) < hex.EncodeToString(br[:])
	})
	if len(eligible) == 0 {
		return question.Identity{}, false
	}
	return eligible[0].Identity, true
}
func (s *Store) CreatePractice(ctx context.Context, a question.Access, in assessment.PracticeCreateInput) (assessment.PracticeView, error) {
	var out assessment.PracticeView
	if !question.ValidMathID(in.Knowledge.ID) || in.Knowledge.Version < 1 || !question.ValidSHA(in.Knowledge.SHA256) || in.ExpectedKnowledgeHead == "" || !question.ValidID(in.ExpectedQuestionHead) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.CreatePracticeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		digest := workflowRequestSHA(in.Knowledge.ID, in)
		receipt, found, e := learningReplay(ctx, tx, u.ID, string(learning.CreatePracticeAction), in.Knowledge.ID, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			p, e := learningReadPractice(ctx, tx, u.ID, receipt.ResourceID, true)
			if e != nil {
				return e
			}
			out, e = learningPracticeView(ctx, tx, u.ID, p, now, nil)
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE practice_attempts SET state='expired',terminal_at=$2 WHERE owner_user_id=$1 AND state='active' AND expires_at<=$2`, u.ID, now); e != nil {
			return e
		}
		var activeID string
		e = tx.QueryRowContext(ctx, `SELECT id::text FROM practice_attempts WHERE owner_user_id=$1 AND state='active' FOR UPDATE`, u.ID).Scan(&activeID)
		if e == nil {
			p, e := learningReadPractice(ctx, tx, u.ID, activeID, false)
			if e != nil {
				return e
			}
			return &learning.ActiveAttemptError{Summary: p.Summary}
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		pool, e := learningSourcePool(ctx, tx, u.ID, in.Knowledge, nil, now)
		if e != nil {
			return e
		}
		if pool.KnowledgeHead != in.ExpectedKnowledgeHead || pool.QuestionHead != in.ExpectedQuestionHead {
			return learning.ErrVersionStale
		}
		blocked, e := learningActiveAssessmentItems(ctx, tx, u.ID, now)
		if e != nil {
			return e
		}
		seed, e := learningSeed()
		if e != nil {
			return e
		}
		id, ok := learningPracticeSelection(pool.Candidates, blocked, seed)
		if !ok {
			return &learning.NotReadyError{}
		}
		bindings, items, e := learningFreezeItems(ctx, tx, pool, []question.Identity{id})
		if e != nil {
			return e
		}
		seal := assessment.Seal{Kind: "practice", Knowledge: pool.Knowledge, KnowledgePublicationID: pool.KnowledgeHead, QuestionPublicationID: pool.QuestionHead, RuleVersion: 1, Core: []int{}, Seed: hex.EncodeToString(seed[:]), Items: bindings}
		raw, sha, e := assessment.CanonicalSeal(seal)
		if e != nil {
			return e
		}
		attempt, e := workflowID()
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, attempt, u.ID, pool.Knowledge.ID, pool.Knowledge.Version, pool.Knowledge.SHA256, pool.KnowledgeHead, pool.QuestionHead, string(raw), raw, sha, now, now.Add(24*time.Hour)); e != nil {
			return e
		}
		if e = learningSaveDependencies(ctx, tx, u.ID, "practice", attempt, raw, false); e != nil {
			return e
		}
		if e = learningRecordViews(ctx, tx, u.ID, bindings, now); e != nil {
			return e
		}
		if e = learningRemember(ctx, tx, u.ID, string(learning.CreatePracticeAction), in.Knowledge.ID, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "practice", ResourceID: attempt, Status: 201}); e != nil {
			return e
		}
		p := practiceRecord{Summary: assessment.AttemptSummary{ID: attempt, Kind: "practice", Knowledge: pool.Knowledge, State: "active", CreatedAt: now.UTC(), ExpiresAt: now.Add(24 * time.Hour).UTC()}, Seal: seal}
		out, e = learningPracticeView(ctx, tx, u.ID, p, now, items)
		return e
	})
	if e != nil {
		return assessment.PracticeView{}, e
	}
	return out, nil
}
func (s *Store) AnswerPractice(ctx context.Context, a question.Access, id string, in assessment.Answer) (assessment.PracticeView, error) {
	return s.practiceCommand(ctx, a, id, in, learning.AnswerPracticeAction)
}
func (s *Store) RevealPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	return s.practiceCommand(ctx, a, id, nil, learning.RevealPracticeAction)
}
func (s *Store) AbandonPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	return s.practiceCommand(ctx, a, id, nil, learning.AbandonPracticeAction)
}
func (s *Store) practiceCommand(ctx context.Context, a question.Access, id string, input any, action learning.Action) (assessment.PracticeView, error) {
	var out assessment.PracticeView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	if input == nil {
		input = struct{}{}
	}
	e := s.learningTx(ctx, a, action, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadPractice(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		digest := workflowRequestSHA(id, input)
		_, found, e := learningReplay(ctx, tx, u.ID, string(action), id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out, e = learningPracticeView(ctx, tx, u.ID, p, now, nil)
			return e
		}
		if p.Summary.State != "active" {
			return learning.ErrStateConflict
		}
		if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return e
		}
		if !now.Before(p.Summary.ExpiresAt) {
			return learning.ErrAssessmentExpired
		}
		items, e := learningLoadItems(ctx, tx, p.Seal)
		if e != nil {
			return e
		}
		state := "abandoned"
		if action != learning.AbandonPracticeAction {
			blocked, e := learningActiveAssessmentItems(ctx, tx, u.ID, now)
			if e != nil {
				return e
			}
			for _, v := range blocked {
				if v == p.Seal.Items[0].Instance {
					return learning.ErrStateConflict
				}
			}
			current, e := learningIsCurrent(ctx, tx, p.Seal.Knowledge)
			if e != nil {
				return e
			}
			deps, e := learningEvidenceDependencies(ctx, tx, "practice", id)
			if e != nil {
				return e
			}
			rs, e := learningEvidenceRestrictions(ctx, tx, deps)
			if e != nil {
				return e
			}
			if !current || len(rs) > 0 {
				return learning.ErrVersionStale
			}
			state = "revealed"
			if action == learning.AnswerPracticeAction {
				answer := input.(assessment.Answer)
				if e = assessment.ValidateAnswer(items[0], answer, false); e != nil {
					return e
				}
				correct, e := assessment.GradeAnswer(items[0], answer)
				if e != nil {
					return e
				}
				p.Answer = &answer
				p.Correct = &correct
				state = "answered"
			}
		}
		var answer any
		if p.Answer != nil {
			answer = body(p.Answer)
		}
		if _, e = tx.ExecContext(ctx, `UPDATE practice_attempts SET state=$2,terminal_at=$3,answer=$4,correct=$5 WHERE id=$1`, id, state, now, answer, p.Correct); e != nil {
			return e
		}
		p.Summary.State = state
		p.TerminalAt = &now
		if state == "answered" {
			p.Summary.SubmittedAt = &now
		}
		out, e = learningPracticeView(ctx, tx, u.ID, p, now, items)
		if e != nil {
			return e
		}
		return learningRemember(ctx, tx, u.ID, string(action), id, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "practice", ResourceID: id, Status: 200})
	})
	if e != nil {
		return assessment.PracticeView{}, e
	}
	return out, nil
}
