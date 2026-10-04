package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// This memo exists only inside one authenticated transaction. Shared content
// locks and the actor row lock keep these facts stable; no value survives commit.
// It contains no historical unlock flags, which may change during enrollment.
type learningProjectionKey struct{}
type learningIdentityFact struct {
	knowledge question.Identity
	head      string
}
type learningProjection struct {
	tx          *sql.Tx
	actor       string
	identities  map[string]learningIdentityFact
	completions map[question.Identity]*string
	evidence    map[question.Identity]learning.EvidenceView
	passes      map[question.Identity]bool
}

func learningWithProjection(ctx context.Context, tx *sql.Tx, actor string) context.Context {
	return context.WithValue(ctx, learningProjectionKey{}, &learningProjection{tx: tx, actor: actor, identities: map[string]learningIdentityFact{}, completions: map[question.Identity]*string{}, evidence: map[question.Identity]learning.EvidenceView{}, passes: map[question.Identity]bool{}})
}
func learningProjectionFor(ctx context.Context, tx *sql.Tx, actor string) *learningProjection {
	p, _ := ctx.Value(learningProjectionKey{}).(*learningProjection)
	if p == nil || p.tx != tx || (actor != "" && p.actor != actor) {
		return nil
	}
	return p
}

// Only internal table aliases and constants enter this expression.
func learningCleanEvidenceSQL(kind, id string) string {
	return fmt.Sprintf(`NOT EXISTS(SELECT 1 FROM learning_evidence_dependencies d WHERE d.evidence_kind='%s' AND d.evidence_id=%s AND
 (EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR (w.target_id=d.id AND w.target_version=d.version)))
 OR EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256)))`, kind, id)
}
func learningKnowledgeIdentity(ctx context.Context, tx *sql.Tx, id string) (question.Identity, string, error) {
	projection := learningProjectionFor(ctx, tx, "")
	if projection != nil {
		if cached, ok := projection.identities[id]; ok {
			return cached.knowledge, cached.head, nil
		}
	}
	var k question.Identity
	var head string
	e := tx.QueryRowContext(ctx, `SELECT k.id,k.version,k.sha256,h.snapshot_id FROM publication_heads h JOIN publication_snapshots p ON p.id=h.snapshot_id AND p.status='published' JOIN publication_members m ON m.snapshot_id=p.id AND m.kind='knowledge' AND m.availability='active' JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE h.singleton AND k.id=$1 AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version AND w.sha256=k.sha256)`, id).Scan(&k.ID, &k.Version, &k.SHA256, &head)
	if e == nil && projection != nil {
		projection.identities[id] = learningIdentityFact{k, head}
	}
	return k, head, workflowRowError(e)
}
func learningIsCurrent(ctx context.Context, tx *sql.Tx, k question.Identity) (bool, error) {
	current, _, e := learningKnowledgeIdentity(ctx, tx, k.ID)
	if errors.Is(e, auth.ErrNotFound) {
		return false, nil
	}
	return current == k, e
}
func learningLectureBasis(ctx context.Context, tx *sql.Tx, k question.Identity) (learning.EventSeal, error) {
	s := learning.EventSeal{Knowledge: k, Units: []question.Identity{}, Assets: []question.AssetRef{}}
	current, head, e := learningKnowledgeIdentity(ctx, tx, k.ID)
	if e != nil {
		return s, e
	}
	if current != k {
		return s, learning.ErrVersionStale
	}
	s.KnowledgePublicationID = head
	var approved bool
	if e = tx.QueryRowContext(ctx, `SELECT learning_content_approved($1,'knowledge',$2,$3,$4)`, head, k.ID, k.Version, k.SHA256).Scan(&approved); e != nil {
		return s, e
	}
	if !approved {
		return s, learning.ErrVersionStale
	}
	rows, e := tx.QueryContext(ctx, `SELECT u.id,u.version,u.sha256 FROM unit_versions u JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='unit' AND m.id=u.id AND m.version=u.version AND m.availability='active' WHERE u.knowledge_id=$2 AND u.knowledge_version=$3 AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='unit' AND w.target_id=u.id AND w.target_version=u.version AND w.sha256=u.sha256) ORDER BY u.id,u.version`, head, k.ID, k.Version)
	if e != nil {
		return s, e
	}
	for rows.Next() {
		var u question.Identity
		if e = rows.Scan(&u.ID, &u.Version, &u.SHA256); e != nil {
			rows.Close()
			return s, e
		}
		s.Units = append(s.Units, u)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return s, e
	}
	rows, e = tx.QueryContext(ctx, `SELECT DISTINCT b.asset_id,b.asset_sha256 FROM jsonb_to_recordset($2::jsonb) r(id text,version integer) JOIN unit_asset_bindings b ON b.unit_id=r.id AND b.unit_version=r.version JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='asset' AND m.id=b.asset_id AND m.availability='active' JOIN package_members pm ON pm.package_id=m.package_id AND pm.package_version=m.package_version AND pm.kind='asset' AND pm.id=m.id AND pm.asset_sha256=b.asset_sha256 WHERE NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='asset' AND w.sha256=b.asset_sha256) ORDER BY b.asset_id,b.asset_sha256`, head, body(s.Units))
	if e != nil {
		return s, e
	}
	defer rows.Close()
	for rows.Next() {
		var a question.AssetRef
		if e = rows.Scan(&a.ID, &a.SHA256); e != nil {
			return s, e
		}
		s.Assets = append(s.Assets, a)
	}
	return s, rows.Err()
}
func learningCurrentCompletion(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (*string, error) {
	projection := learningProjectionFor(ctx, tx, actor)
	if projection != nil {
		if cached, ok := projection.completions[k]; ok {
			return cached, nil
		}
	}
	out, e := learningCurrentCompletionUncached(ctx, tx, actor, k)
	if e == nil && projection != nil {
		projection.completions[k] = out
	}
	return out, e
}
func learningCurrentCompletionUncached(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (*string, error) {
	current, e := learningIsCurrent(ctx, tx, k)
	if e != nil || !current {
		return nil, e
	}
	// Most current nodes have never been completed by this actor. Do not build
	// and prove a complete lecture basis unless an exact completion exists.
	var hasCompletion bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_events WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND kind='completed')`, actor, k.ID, k.Version, k.SHA256).Scan(&hasCompletion); e != nil {
		return nil, e
	}
	if !hasCompletion {
		return nil, nil
	}
	basis, e := learningLectureBasis(ctx, tx, k)
	if e != nil {
		return nil, e
	}
	var id string
	e = tx.QueryRowContext(ctx, `SELECT e.id::text FROM learning_events e WHERE e.owner_user_id=$1 AND e.knowledge_id=$2 AND e.knowledge_version=$3 AND e.knowledge_sha256=$4 AND e.kind='completed' AND e.material_sha256=learning_material_hash(jsonb_build_object('body',$5::jsonb)) AND `+learningCleanEvidenceSQL("learning-event", "e.id")+` ORDER BY e.recorded_at DESC,e.id LIMIT 1`, actor, k.ID, k.Version, k.SHA256, body(basis)).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &id, nil
}
func learningCurrentEvidence(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (learning.EvidenceView, error) {
	projection := learningProjectionFor(ctx, tx, actor)
	if projection != nil {
		if cached, ok := projection.evidence[k]; ok {
			return cached, nil
		}
	}
	out, e := learningCurrentEvidenceUncached(ctx, tx, actor, k)
	if e == nil && projection != nil {
		projection.evidence[k] = out
	}
	return out, e
}
func learningCurrentEvidenceUncached(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (learning.EvidenceView, error) {
	out := learning.EvidenceView{}
	current, e := learningIsCurrent(ctx, tx, k)
	if e != nil || !current {
		return out, e
	}
	qual, e := correctionPassingEvidence(ctx, tx, actor, k)
	if e != nil {
		return out, e
	}
	if qual == nil {
		return out, nil
	}
	return learning.EvidenceView{Qualified: true, Qualification: qual}, nil
}
func learningEffectivePass(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (bool, error) {
	projection := learningProjectionFor(ctx, tx, actor)
	if projection != nil {
		if cached, ok := projection.passes[k]; ok {
			return cached, nil
		}
	}
	out, e := learningEffectivePassUncached(ctx, tx, actor, k)
	if e == nil && projection != nil {
		projection.passes[k] = out
	}
	return out, e
}
func learningEffectivePassUncached(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (bool, error) {
	current, e := learningIsCurrent(ctx, tx, k)
	if e != nil || !current {
		return false, e
	}
	var passed bool
	e = tx.QueryRowContext(ctx, `WITH candidates AS (`+correctionAssessmentProjectionSQL(ctx)+`) SELECT EXISTS(SELECT 1 FROM candidates WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND outcome='passed')`, actor, k.ID, k.Version, k.SHA256).Scan(&passed)
	return passed, e
}
func learningGrantEvidence(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, now time.Time) (bool, error) {
	v, e := learningCurrentEvidence(ctx, tx, actor, k)
	if e != nil || !v.Qualified {
		return false, e
	}
	if v.Qualification.CorrectionID != nil {
		return true, correctionGrantEvidence(ctx, tx, actor, k, *v.Qualification, now)
	}
	id, e := workflowID()
	if e != nil {
		return false, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO learning_qualification_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,attempt_id,completed_event_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`, id, actor, k.ID, k.Version, k.SHA256, v.Qualification.Kind, v.Qualification.EvidenceAttemptID, v.Qualification.CompletedEventID, now)
	return true, e
}
func learningPrerequisites(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) ([]learning.PrerequisiteState, bool, error) {
	out := []learning.PrerequisiteState{}
	rows, e := tx.QueryContext(ctx, `SELECT k.id,k.version,k.sha256 FROM knowledge_relations r JOIN knowledge_versions k ON k.id=r.target_id AND k.version=r.target_version WHERE r.source_id=$1 AND r.source_version=$2 AND r.kind='prerequisite' ORDER BY k.id,k.version`, k.ID, k.Version)
	if e != nil {
		return out, false, e
	}
	for rows.Next() {
		var p learning.PrerequisiteState
		p.Reasons = []assessment.RestrictionReason{}
		if e = rows.Scan(&p.Knowledge.ID, &p.Knowledge.Version, &p.Knowledge.SHA256); e != nil {
			rows.Close()
			return out, false, e
		}
		out = append(out, p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, false, e
	}
	all := true
	for j := range out {
		v, e := learningCurrentEvidence(ctx, tx, actor, out[j].Knowledge)
		if e != nil {
			return out, false, e
		}
		out[j].Qualified = v.Qualified
		if !v.Qualified {
			all = false
			current, e := learningIsCurrent(ctx, tx, out[j].Knowledge)
			if e != nil {
				return out, false, e
			}
			if !current {
				out[j].Reasons = append(out[j].Reasons, assessment.KnowledgeUpdated)
			}
		}
	}
	return out, all, nil
}
func learningKnowledgeState(ctx context.Context, tx *sql.Tx, actor string, k question.Identity) (learning.KnowledgeState, error) {
	out := learning.KnowledgeState{Knowledge: k, Prerequisites: []learning.PrerequisiteState{}}
	if e := tx.QueryRowContext(ctx, `SELECT body->>'title',body->>'titleZh' FROM knowledge_versions WHERE id=$1 AND version=$2 AND sha256=$3`, k.ID, k.Version, k.SHA256).Scan(&out.Title, &out.TitleZh); e != nil {
		return out, workflowRowError(e)
	}
	var started, completed sql.NullTime
	e := tx.QueryRowContext(ctx, `SELECT started_at,completed_at FROM learning_records WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4`, actor, k.ID, k.Version, k.SHA256).Scan(&started, &completed)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	if started.Valid {
		t := started.Time.UTC()
		out.StartedAt = &t
	}
	if completed.Valid {
		t := completed.Time.UTC()
		out.CompletedAt = &t
	}
	current, e := learningIsCurrent(ctx, tx, k)
	if e != nil {
		return out, e
	}
	v, e := learningCurrentEvidence(ctx, tx, actor, k)
	if e != nil {
		return out, e
	}
	out.Qualification = v.Qualification
	completion, e := learningCurrentCompletion(ctx, tx, actor, k)
	if e != nil {
		return out, e
	}
	out.CompletionValid = completion != nil
	out.Prerequisites, out.CanEnter, e = learningPrerequisites(ctx, tx, actor, k)
	if e != nil {
		return out, e
	}
	out.CanEnter = current && (out.CanEnter || v.Qualified && v.Qualification.Kind == "diagnostic")
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_unlocks WHERE owner_user_id=$1 AND knowledge_id=$2)`, actor, k.ID).Scan(&out.EverUnlocked); e != nil {
		return out, e
	}
	var had, pass, failed bool
	e = tx.QueryRowContext(ctx, `WITH candidates AS (`+correctionAssessmentProjectionSQL(ctx)+`) SELECT EXISTS(SELECT 1 FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.owner_user_id=$1 AND a.knowledge_id=$2 AND a.knowledge_version=$3 AND a.knowledge_sha256=$4 AND r.outcome='passed'), EXISTS(SELECT 1 FROM candidates WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND outcome='passed'),coalesce((SELECT outcome='failed' FROM candidates WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND mode='review' ORDER BY terminal_at DESC,attempt_id LIMIT 1),false)`, actor, k.ID, k.Version, k.SHA256).Scan(&had, &pass, &failed)
	if e != nil {
		return out, e
	}
	out.State = learning.ResolveState(learning.StateFacts{Started: started.Valid, Completed: out.CompletionValid, HadInvalidatedPass: had && !pass, EffectivePass: current && pass, LatestReviewFailed: current && failed})
	return out, nil
}
func learningApplyAssessment(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time, fact assessment.AttemptFact) (assessment.ProgressUpdate, error) {
	out := assessment.ProgressUpdate{Knowledge: fact.Knowledge, NewlyUnlocked: []question.Identity{}}
	var stored assessment.AttemptFact
	var state string
	e := tx.QueryRowContext(ctx, `SELECT a.id::text,a.knowledge_id,a.knowledge_version,a.knowledge_sha256,a.mode,a.state,a.terminal_at,r.outcome,r.score,r.passed FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.id=$1 AND a.owner_user_id=$2`, fact.ID, u.ID).Scan(&stored.ID, &stored.Knowledge.ID, &stored.Knowledge.Version, &stored.Knowledge.SHA256, &stored.Mode, &state, &stored.SubmittedAt, &stored.Outcome, &stored.Score, &stored.Passed)
	if e != nil {
		return out, workflowRowError(e)
	}
	if state != "submitted" || stored.Knowledge != fact.Knowledge || stored.Mode != fact.Mode || stored.Outcome != fact.Outcome || body(stored.Score) != body(fact.Score) || body(stored.Passed) != body(fact.Passed) || !stored.SubmittedAt.Equal(fact.SubmittedAt) {
		return out, learning.ErrStateConflict
	}
	if stored.Outcome != assessment.Passed {
		return out, nil
	}
	rs, e := correctionEvidenceGuard(ctx, tx, u.ID, correction.EvidenceRef{Kind: correction.EvidenceKind("assessment"), ID: stored.ID})
	if e != nil {
		return out, e
	}
	current, e := learningIsCurrent(ctx, tx, fact.Knowledge)
	if e != nil {
		return out, e
	}
	if !current || len(rs) > 0 || fact.Validity != assessment.Effective {
		return out, learning.ErrVersionStale
	}
	out.QualificationGranted, e = learningGrantEvidence(ctx, tx, u.ID, fact.Knowledge, now)
	if e != nil {
		return out, e
	}
	if out.QualificationGranted {
		if stored.Mode == assessment.ModeDiagnostic {
			added, e := learningUnlock(ctx, tx, u.ID, fact.Knowledge, "assessment", stored.ID, now)
			if e != nil {
				return out, e
			}
			if added {
				out.NewlyUnlocked = append(out.NewlyUnlocked, fact.Knowledge)
			}
		}
		next, e := learningUnlockSuccessors(ctx, tx, u.ID, fact.Knowledge, "assessment", stored.ID, now)
		if e != nil {
			return out, e
		}
		out.NewlyUnlocked = append(out.NewlyUnlocked, next...)
	}
	return out, nil
}

func learningSaveDependencies(ctx context.Context, tx *sql.Tx, actor, kind, id string, seal []byte, isEvent bool) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO learning_evidence_dependencies(evidence_kind,evidence_id,owner_user_id,kind,id,version,sha256) SELECT $1,$2,$3,kind,id,version,sha256 FROM learning_seal_dependencies($4::jsonb,$5) ON CONFLICT DO NOTHING`, kind, id, actor, string(seal), isEvent)
	return e
}
func learningDecodeSeal(raw []byte) (assessment.Seal, error) {
	var v struct {
		Body assessment.Seal `json:"body"`
	}
	e := json.Unmarshal(raw, &v)
	return v.Body, e
}
