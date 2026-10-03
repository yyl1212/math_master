package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func learningPageQuery(q learning.ListQuery) (learning.ListQuery, error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 100000 {
		return q, auth.ErrInvalidInput
	}
	return q, nil
}

const learningCurrentKnowledgeSQL = `SELECT k.id,k.version,k.sha256 FROM publication_heads h JOIN publication_snapshots p ON p.id=h.snapshot_id AND p.status='published' JOIN publication_members m ON m.snapshot_id=p.id AND m.kind='knowledge' AND m.availability='active' JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE h.singleton AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version AND w.sha256=k.sha256)`

func learningCurrentKnowledge(ctx context.Context, tx *sql.Tx, limit, offset int) ([]question.Identity, error) {
	out := []question.Identity{}
	rows, e := tx.QueryContext(ctx, learningCurrentKnowledgeSQL+` ORDER BY k.id,k.version LIMIT $1 OFFSET $2`, limit, offset)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var k question.Identity
		if e = rows.Scan(&k.ID, &k.Version, &k.SHA256); e != nil {
			return out, e
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
func learningActiveSummary(ctx context.Context, tx *sql.Tx, actor, kind string, now time.Time) (*assessment.AttemptSummary, error) {
	var v assessment.AttemptSummary
	var mode *assessment.Mode
	query := `SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,NULL::text,state,created_at,expires_at FROM practice_attempts WHERE owner_user_id=$1 AND state='active' AND expires_at>$2`
	if kind == "assessment" {
		query = `SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,mode,state,created_at,expires_at FROM assessment_attempts WHERE owner_user_id=$1 AND sealed AND state='active' AND expires_at>$2`
	}
	e := tx.QueryRowContext(ctx, query, actor, now).Scan(&v.ID, &v.Knowledge.ID, &v.Knowledge.Version, &v.Knowledge.SHA256, &mode, &v.State, &v.CreatedAt, &v.ExpiresAt)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	v.Kind = kind
	v.Mode = mode
	v.CreatedAt = v.CreatedAt.UTC()
	v.ExpiresAt = v.ExpiresAt.UTC()
	return &v, nil
}
func learningBlueprintOptions(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, now time.Time) ([]learning.BlueprintOption, *string, error) {
	out := []learning.BlueprintOption{}
	pool, e := learningSourcePool(ctx, tx, actor, k, nil, now)
	if e != nil {
		return out, nil, e
	}
	if pool.QuestionHead == "" {
		return out, nil, nil
	}
	head := pool.QuestionHead
	rows, e := tx.QueryContext(ctx, `SELECT b.id,b.version,b.sha256,b.body->'body' FROM question_blueprints b JOIN question_publication_members m ON m.publication_id=$1 AND m.kind='blueprint' AND m.id=b.id AND m.version=b.version AND m.sha256=b.sha256 JOIN question_review_decisions r ON r.id=m.review_id AND r.decision='approve' JOIN question_submissions s ON s.id=m.submission_id AND s.id=r.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=r.frozen_digest JOIN question_submission_members sm ON sm.submission_id=s.id AND sm.kind='blueprint' AND sm.id=b.id AND sm.version=b.version AND sm.sha256=b.sha256 WHERE b.sealed AND b.knowledge_id=$2 AND b.knowledge_version=$3 AND NOT EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind='blueprint' AND w.target_id=b.id AND w.target_version=b.version AND w.sha256=b.sha256) ORDER BY b.id,b.version LIMIT 1001`, head, k.ID, k.Version)
	if e != nil {
		return out, nil, e
	}
	type choice struct {
		id question.Identity
		bp question.Blueprint
	}
	choices := []choice{}
	for rows.Next() {
		var c choice
		var raw []byte
		if e = rows.Scan(&c.id.ID, &c.id.Version, &c.id.SHA256, &raw); e != nil {
			rows.Close()
			return out, nil, e
		}
		if json.Unmarshal(raw, &c.bp) != nil {
			rows.Close()
			return out, nil, auth.ErrUnavailable
		}
		choices = append(choices, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, nil, e
	}
	if len(choices) > 1000 {
		return out, nil, question.ErrLimitExceeded
	}
	// One metadata pool per knowledge; equivalent source/core sets share the DP.
	type readiness struct {
		ready   bool
		reasons []learning.ReadinessReason
		retry   *time.Time
	}
	cache := map[string]readiness{}
	var seed [32]byte
	for _, c := range choices {
		// The pool and its personal facts are fixed for this authenticated,
		// content-locked transaction. Cache before expanding equivalent sources.
		key := body(struct {
			Core    []int
			Sources []question.BlueprintSource
		}{c.bp.CoreObjectiveIndices, c.bp.Sources})
		r, ok := cache[key]
		if !ok {
			instances, templates := map[question.Ref]bool{}, map[question.Ref]bool{}
			for _, ref := range c.bp.Sources {
				if ref.Kind == "instance" {
					instances[ref.Ref] = true
				} else {
					templates[ref.Ref] = true
				}
			}
			candidates := []assessment.Candidate{}
			for _, v := range pool.Candidates {
				yes := instances[question.Ref{ID: v.Identity.ID, Version: v.Identity.Version}]
				if v.Template != nil {
					yes = yes || templates[question.Ref{ID: v.Template.ID, Version: v.Template.Version}]
				}
				if yes {
					candidates = append(candidates, v)
				}
			}
			_, r.ready, e = assessment.SelectFive(ctx, c.bp.CoreObjectiveIndices, assessment.EligibleCandidates(candidates, now), seed)
			if e != nil {
				return out, nil, e
			}
			r.reasons = []learning.ReadinessReason{}
			if !r.ready {
				r.retry, e = assessment.EarliestReady(ctx, c.bp.CoreObjectiveIndices, candidates, now, seed)
				if e != nil {
					return out, nil, e
				}
				_, eventually, e := assessment.SelectFive(ctx, c.bp.CoreObjectiveIndices, assessment.EligibleCandidates(candidates, now.Add(31*time.Minute)), seed)
				if e != nil {
					return out, nil, e
				}
				switch {
				case len(candidates) == 0:
					r.reasons = append(r.reasons, learning.NoInstances)
				case r.retry != nil:
					r.reasons = append(r.reasons, learning.ExposureCooldown)
				default:
					unblocked := append([]assessment.Candidate{}, candidates...)
					recent := false
					for j := range unblocked {
						recent = recent || unblocked[j].RecentSubmitted
						unblocked[j].RecentSubmitted = false
						unblocked[j].ExposedAt = nil
					}
					_, all, e := assessment.SelectFive(ctx, c.bp.CoreObjectiveIndices, unblocked, seed)
					if e != nil {
						return out, nil, e
					}
					if !eventually && recent && all {
						r.reasons = append(r.reasons, learning.RecentAssessmentExclusion)
					} else {
						r.reasons = append(r.reasons, learning.InsufficientCoverage)
					}
				}
			}
			cache[key] = r
		}
		if e = correctionNewAttemptGuard(ctx, tx, actor, k, c.bp.RuleVersion, now); e != nil {
			if !errors.Is(e, learning.ErrAssessmentNotReady) {
				return out, nil, e
			}
			r.ready = false
			r.retry = nil
			r.reasons = append(r.reasons, learning.GradingIssue)
		}
		out = append(out, learning.BlueprintOption{Blueprint: c.id, CoreObjectiveIndices: append([]int{}, c.bp.CoreObjectiveIndices...), Ready: r.ready, Reasons: append([]learning.ReadinessReason{}, r.reasons...), RetryAt: r.retry})
	}
	return out, &head, nil
}
func (s *Store) ListLearningKnowledge(ctx context.Context, a question.Access, q learning.ListQuery) (question.Page[learning.KnowledgeState], error) {
	var out question.Page[learning.KnowledgeState]
	q, e := learningPageQuery(q)
	if e != nil {
		return out, e
	}
	out = question.Page[learning.KnowledgeState]{Items: []learning.KnowledgeState{}, Limit: q.Limit, Offset: q.Offset}
	e = s.learningTx(ctx, a, learning.ListKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM (`+learningCurrentKnowledgeSQL+`) current`).Scan(&out.Total); e != nil {
			return e
		}
		ks, e := learningCurrentKnowledge(ctx, tx, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		for _, k := range ks {
			state, e := learningKnowledgeState(ctx, tx, u.ID, k)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, state)
		}
		return nil
	})
	if e != nil {
		return question.Page[learning.KnowledgeState]{}, e
	}
	return out, nil
}
func (s *Store) ReadLearningKnowledge(ctx context.Context, a question.Access, id string, version int) (learning.KnowledgeDetail, error) {
	var out learning.KnowledgeDetail
	if !question.ValidMathID(id) || version < 1 || version > 2147483647 {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		k := question.Identity{ID: id, Version: version}
		if e := tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, id, version).Scan(&k.SHA256); e != nil {
			return workflowRowError(e)
		}
		current, e := learningIsCurrent(ctx, tx, k)
		if e != nil {
			return e
		}
		out.Objectives = []string{}
		out.Blueprints = []learning.BlueprintOption{}
		if !current {
			// A former version is a private summary only, backed by this owner's facts.
			e = tx.QueryRowContext(ctx, `SELECT publication FROM (SELECT knowledge_publication_id publication,recorded_at occurred FROM learning_events WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 UNION ALL SELECT knowledge_publication_id,created_at FROM assessment_attempts WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 UNION ALL SELECT knowledge_publication_id,created_at FROM practice_attempts WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 UNION ALL SELECT e.knowledge_publication_id,e.created_at FROM learning_path_nodes n JOIN learning_path_enrollments e ON e.id=n.enrollment_id WHERE e.owner_user_id=$1 AND n.knowledge_id=$2 AND n.knowledge_version=$3 AND n.knowledge_sha256=$4) own ORDER BY occurred DESC LIMIT 1`, u.ID, id, version, k.SHA256).Scan(&out.KnowledgeHead)
			if e != nil {
				return workflowRowError(e)
			}
		} else {
			_, out.KnowledgeHead, e = learningKnowledgeIdentity(ctx, tx, id)
			if e != nil {
				return e
			}
			var raw []byte
			if e = tx.QueryRowContext(ctx, `SELECT body->'objectives' FROM knowledge_versions WHERE id=$1 AND version=$2 AND sha256=$3`, id, version, k.SHA256).Scan(&raw); e != nil {
				return e
			}
			if json.Unmarshal(raw, &out.Objectives) != nil {
				return auth.ErrUnavailable
			}
			out.Blueprints, out.QuestionHead, e = learningBlueprintOptions(ctx, tx, u.ID, k, now)
			if e != nil {
				return e
			}
			out.ActiveAssessment, e = learningActiveSummary(ctx, tx, u.ID, "assessment", now)
			if e != nil {
				return e
			}
		}
		out.State, e = learningKnowledgeState(ctx, tx, u.ID, k)
		return e
	})
	if e != nil {
		return learning.KnowledgeDetail{}, e
	}
	return out, nil
}
func learningAvailablePaths(ctx context.Context, tx *sql.Tx) ([]learning.PublishedPathSummary, error) {
	out := []learning.PublishedPathSummary{}
	rows, e := tx.QueryContext(ctx, `SELECT p.id,p.version,p.sha256,p.body->>'title',p.body->>'titleZh',(SELECT count(*) FROM path_nodes WHERE path_id=p.id AND path_version=p.version) FROM publication_heads h JOIN publication_snapshots pub ON pub.id=h.snapshot_id AND pub.status='published' JOIN publication_members m ON m.snapshot_id=pub.id AND m.kind='path' AND m.availability='active' JOIN path_versions p ON p.id=m.id AND p.version=m.version WHERE h.singleton AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='path' AND w.target_id=p.id AND w.target_version=p.version) AND NOT EXISTS(SELECT 1 FROM path_nodes pn LEFT JOIN publication_members km ON km.snapshot_id=h.snapshot_id AND km.kind='knowledge' AND km.id=pn.knowledge_id AND km.version=pn.knowledge_version AND km.availability='active' WHERE pn.path_id=p.id AND pn.path_version=p.version AND km.id IS NULL) ORDER BY p.id,p.version LIMIT 201`)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var p learning.PublishedPathSummary
		if e = rows.Scan(&p.Path.ID, &p.Path.Version, &p.Path.SHA256, &p.Title, &p.TitleZh, &p.TotalNodes); e != nil {
			return out, e
		}
		out = append(out, p)
	}
	if len(out) > 200 {
		return out, question.ErrLimitExceeded
	}
	return out, rows.Err()
}
func (s *Store) ReadLearningOverview(ctx context.Context, a question.Access) (learning.Overview, error) {
	out := learning.Overview{AvailablePaths: []learning.PublishedPathSummary{}, Recent: []learning.HistoryEntry{}}
	e := s.learningTx(ctx, a, learning.ReadOverviewAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		var khead, qhead sql.NullString
		if e := tx.QueryRowContext(ctx, `SELECT (SELECT h.snapshot_id FROM publication_heads h JOIN publication_snapshots p ON p.id=h.snapshot_id AND p.status='published' WHERE h.singleton),(SELECT h.publication_id::text FROM question_heads h JOIN question_publications p ON p.id=h.publication_id AND p.sealed AND p.status='published' WHERE h.singleton)`).Scan(&khead, &qhead); e != nil {
			return e
		}
		if khead.Valid {
			out.KnowledgeHead = &khead.String
		}
		if qhead.Valid {
			out.QuestionHead = &qhead.String
		}
		var e error
		out.AvailablePaths, e = learningAvailablePaths(ctx, tx)
		if e != nil {
			return e
		}
		ks, e := learningCurrentKnowledge(ctx, tx, 1001, 0)
		if e != nil {
			return e
		}
		if len(ks) > 1000 {
			return question.ErrLimitExceeded
		}
		// Count exact current personal facts without each node's title and
		// prerequisite projection. Material hashes still validate completion.
		if e = tx.QueryRowContext(ctx, `WITH current AS MATERIALIZED (`+learningCurrentKnowledgeSQL+`)
 SELECT (SELECT count(*) FROM learning_records r JOIN current k ON k.id=r.knowledge_id AND k.version=r.knowledge_version AND k.sha256=r.knowledge_sha256 WHERE r.owner_user_id=$1),
 (SELECT count(*) FROM current k WHERE EXISTS(SELECT 1 FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.owner_user_id=$1 AND a.knowledge_id=k.id AND a.knowledge_version=k.version AND a.knowledge_sha256=k.sha256 AND a.state='submitted' AND r.outcome='passed' AND r.passed AND r.score BETWEEN 4 AND 5 AND `+(learningCleanEvidenceSQL("assessment", "a.id")+` AND `+correctionOriginalEvidenceSQL(ctx, "assessment", "a.id"))+`))`, u.ID).Scan(&out.StartedCount, &out.EffectivePassedCount); e != nil {
			return e
		}
		for _, k := range ks {
			completed, e := learningCurrentCompletion(ctx, tx, u.ID, k)
			if e != nil {
				return e
			}
			if completed != nil {
				out.CompletedCount++
			}
		}

		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM learning_unlocks WHERE owner_user_id=$1`, u.ID).Scan(&out.HistoricalUnlockedCount); e != nil {
			return e
		}
		out.ActivePractice, e = learningActiveSummary(ctx, tx, u.ID, "practice", now)
		if e != nil {
			return e
		}
		out.ActiveAssessment, e = learningActiveSummary(ctx, tx, u.ID, "assessment", now)
		if e != nil {
			return e
		}
		history, e := learningHistoryPage(ctx, tx, u.ID, learning.ListQuery{Limit: 20}, now)
		if e != nil {
			return e
		}
		out.Recent = history.Items
		return nil
	})
	if e != nil {
		return learning.Overview{}, e
	}
	return out, nil
}
func (s *Store) ListLearningPaths(ctx context.Context, a question.Access, q learning.ListQuery) (question.Page[learning.PathSummary], error) {
	var out question.Page[learning.PathSummary]
	q, e := learningPageQuery(q)
	if e != nil {
		return out, e
	}
	out = question.Page[learning.PathSummary]{Items: []learning.PathSummary{}, Limit: q.Limit, Offset: q.Offset}
	e = s.learningTx(ctx, a, learning.ListPathsAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM learning_path_enrollments WHERE owner_user_id=$1`, u.ID).Scan(&out.Total); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT id::text FROM learning_path_enrollments WHERE owner_user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, u.ID, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			p, e := learningPathSummary(ctx, tx, u.ID, id)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, p)
		}
		return nil
	})
	if e != nil {
		return question.Page[learning.PathSummary]{}, e
	}
	return out, nil
}
func (s *Store) ReadLearningPath(ctx context.Context, a question.Access, id string) (learning.PathView, error) {
	var out learning.PathView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.ReadPathAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		var e error
		out.Summary, e = learningPathSummary(ctx, tx, u.ID, id)
		return e
	})
	if e != nil {
		return learning.PathView{}, e
	}
	return out, nil
}
func (s *Store) ListLearningPathNodes(ctx context.Context, a question.Access, id string, q learning.ListQuery) (question.Page[learning.PathNode], error) {
	var out question.Page[learning.PathNode]
	q, e := learningPageQuery(q)
	if e != nil {
		return out, e
	}
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	out = question.Page[learning.PathNode]{Items: []learning.PathNode{}, Limit: q.Limit, Offset: q.Offset}
	e = s.learningTx(ctx, a, learning.ListPathNodesAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		if e := tx.QueryRowContext(ctx, `SELECT total_nodes FROM learning_path_enrollments WHERE id=$1 AND owner_user_id=$2`, id, u.ID).Scan(&out.Total); e != nil {
			return workflowRowError(e)
		}
		rows, e := tx.QueryContext(ctx, `SELECT position,knowledge_id,knowledge_version,knowledge_sha256 FROM learning_path_nodes WHERE enrollment_id=$1 ORDER BY position LIMIT $2 OFFSET $3`, id, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		type node struct {
			pos int
			k   question.Identity
		}
		ns := []node{}
		for rows.Next() {
			var n node
			if e = rows.Scan(&n.pos, &n.k.ID, &n.k.Version, &n.k.SHA256); e != nil {
				rows.Close()
				return e
			}
			ns = append(ns, n)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, n := range ns {
			state, e := learningKnowledgeState(ctx, tx, u.ID, n.k)
			if e != nil {
				return e
			}
			available, e := learningIsCurrent(ctx, tx, n.k)
			if e != nil {
				return e
			}
			reasons, e := learningEvidenceRestrictions(ctx, tx, []learning.EvidenceDependency{{Kind: "knowledge", ID: n.k.ID, Version: &n.k.Version, SHA256: n.k.SHA256}})
			if e != nil {
				return e
			}
			if !available && len(reasons) == 0 {
				reasons = append(reasons, assessment.KnowledgeUpdated)
			}
			out.Items = append(out.Items, learning.PathNode{Position: n.pos, Title: state.Title, TitleZh: state.TitleZh, State: state, Available: available, Reasons: reasons})
		}
		return nil
	})
	if e != nil {
		return question.Page[learning.PathNode]{}, e
	}
	return out, nil
}
