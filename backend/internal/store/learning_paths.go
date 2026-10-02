package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func learningUnlock(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, kind, id string, now time.Time) (bool, error) {
	current, head, e := learningKnowledgeIdentity(ctx, tx, k.ID)
	if e != nil {
		return false, e
	}
	if current != k {
		return false, learning.ErrVersionStale
	}
	r, e := tx.ExecContext(ctx, `INSERT INTO learning_unlocks(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,source_kind,source_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(owner_user_id,knowledge_id) DO NOTHING`, actor, k.ID, k.Version, k.SHA256, head, kind, id, now)
	if e != nil {
		return false, e
	}
	n, e := r.RowsAffected()
	return n == 1, e
}
func learningUnlockSuccessors(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, kind, id string, now time.Time) ([]question.Identity, error) {
	ctx = learningWithProjection(ctx, tx, actor)
	out := []question.Identity{}
	rows, e := tx.QueryContext(ctx, `SELECT DISTINCT kv.id,kv.version,kv.sha256,h.snapshot_id FROM knowledge_relations r JOIN knowledge_versions kv ON kv.id=r.source_id AND kv.version=r.source_version JOIN publication_heads h ON h.singleton JOIN publication_members m ON m.snapshot_id=h.snapshot_id AND m.kind='knowledge' AND m.id=kv.id AND m.version=kv.version AND m.availability='active' WHERE r.kind='prerequisite' AND r.target_id=$1 AND r.target_version=$2 AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=kv.id AND w.target_version=kv.version) ORDER BY kv.id,kv.version LIMIT 1001`, k.ID, k.Version)
	if e != nil {
		return out, e
	}
	ks := []question.Identity{}
	var head string
	for rows.Next() {
		var v question.Identity
		if e = rows.Scan(&v.ID, &v.Version, &v.SHA256, &head); e != nil {
			rows.Close()
			return out, e
		}
		ks = append(ks, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	if len(ks) > 1000 {
		return out, question.ErrLimitExceeded
	}
	if len(ks) == 0 {
		return out, nil
	}
	prerequisites, e := learningSuccessorPrerequisites(ctx, tx, ks)
	if e != nil {
		return out, e
	}
	eligible := []question.Identity{}
	for _, v := range ks {
		all := true
		for _, prerequisite := range prerequisites[v] {
			evidence, e := learningCurrentEvidence(ctx, tx, actor, prerequisite)
			if e != nil {
				return out, e
			}
			if !evidence.Qualified {
				all = false
			}
		}
		if all {
			eligible = append(eligible, v)
		}
	}
	// The current identities and head came from one read under the shared content
	// lock. All original owner and approval constraints still run on every row.
	return learningUnlockMany(ctx, tx, actor, eligible, head, kind, id, now)
}

func learningSuccessorPrerequisites(ctx context.Context, tx *sql.Tx, ks []question.Identity) (map[question.Identity][]question.Identity, error) {
	out := map[question.Identity][]question.Identity{}
	rows, e := tx.QueryContext(ctx, `SELECT source.id,source.version,source.sha256,k.id,k.version,k.sha256
 FROM jsonb_to_recordset($1::jsonb) source(id text,version integer,sha256 text)
 JOIN knowledge_relations r ON r.source_id=source.id AND r.source_version=source.version AND r.kind='prerequisite'
 JOIN knowledge_versions k ON k.id=r.target_id AND k.version=r.target_version
 ORDER BY source.id,source.version,k.id,k.version`, body(ks))
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var source, target question.Identity
		if e = rows.Scan(&source.ID, &source.Version, &source.SHA256, &target.ID, &target.Version, &target.SHA256); e != nil {
			return out, e
		}
		out[source] = append(out[source], target)
	}
	return out, rows.Err()
}

func learningUnlockMany(ctx context.Context, tx *sql.Tx, actor string, ks []question.Identity, head, kind, id string, now time.Time) ([]question.Identity, error) {
	out := []question.Identity{}
	if len(ks) == 0 {
		return out, nil
	}
	rows, e := tx.QueryContext(ctx, `WITH inserted AS (
 INSERT INTO learning_unlocks(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,source_kind,source_id,created_at)
 SELECT $1,k.id,k.version,k.sha256,$3,$4,$5,$6 FROM jsonb_to_recordset($2::jsonb) k(id text,version integer,sha256 text)
 ON CONFLICT(owner_user_id,knowledge_id) DO NOTHING
 RETURNING knowledge_id,knowledge_version,knowledge_sha256
 ) SELECT knowledge_id,knowledge_version,knowledge_sha256 FROM inserted ORDER BY knowledge_id,knowledge_version`, actor, body(ks), head, kind, id, now)
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
func (s *Store) EnrollLearningPath(ctx context.Context, a question.Access, id string, in learning.EnrollInput) (learning.PathView, error) {
	var out learning.PathView
	if !question.ValidMathID(id) || in.Path.ID != id || in.Path.Version < 1 || !question.ValidSHA(in.Path.SHA256) || in.ExpectedKnowledgeHead == "" {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.EnrollPathAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		ctx = learningWithProjection(ctx, tx, u.ID)
		digest := workflowRequestSHA(id, in)
		receipt, found, e := learningReplay(ctx, tx, u.ID, string(learning.EnrollPathAction), id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Summary, e = learningPathSummary(ctx, tx, u.ID, receipt.ResourceID)
			return e
		}
		var head string
		var approved bool
		e = tx.QueryRowContext(ctx, `SELECT h.snapshot_id,learning_content_approved(h.snapshot_id,'path',p.id,p.version,p.sha256) FROM publication_heads h JOIN publication_members m ON m.snapshot_id=h.snapshot_id AND m.kind='path' AND m.availability='active' JOIN path_versions p ON p.id=m.id AND p.version=m.version WHERE h.singleton AND p.id=$1 AND p.version=$2 AND p.sha256=$3 AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='path' AND w.target_id=p.id AND w.target_version=p.version)`, id, in.Path.Version, in.Path.SHA256).Scan(&head, &approved)
		if e != nil || !approved || head != in.ExpectedKnowledgeHead {
			if e != nil && e != sql.ErrNoRows {
				return e
			}
			return learning.ErrVersionStale
		}
		var enrollment string
		e = tx.QueryRowContext(ctx, `SELECT id::text FROM learning_path_enrollments WHERE owner_user_id=$1 AND path_id=$2 AND path_version=$3 AND path_sha256=$4`, u.ID, id, in.Path.Version, in.Path.SHA256).Scan(&enrollment)
		if e != nil && e != sql.ErrNoRows {
			return e
		}
		if e == sql.ErrNoRows {
			rows, e := tx.QueryContext(ctx, `SELECT pn.position,k.id,k.version,k.sha256 FROM path_nodes pn JOIN knowledge_versions k ON k.id=pn.knowledge_id AND k.version=pn.knowledge_version JOIN publication_members m ON m.snapshot_id=$3 AND m.kind='knowledge' AND m.id=k.id AND m.version=k.version AND m.availability='active' WHERE pn.path_id=$1 AND pn.path_version=$2 AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version) ORDER BY pn.position LIMIT 1001`, id, in.Path.Version, head)
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
			if len(ns) == 0 || len(ns) > 1000 {
				return question.ErrLimitExceeded
			}
			enrollment, e = workflowID()
			if e != nil {
				return e
			}
			if _, e = tx.ExecContext(ctx, `INSERT INTO learning_path_enrollments(id,owner_user_id,path_id,path_version,path_sha256,knowledge_publication_id,total_nodes,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, enrollment, u.ID, id, in.Path.Version, in.Path.SHA256, head, len(ns), now); e != nil {
				return e
			}
			for _, n := range ns {
				if _, e = tx.ExecContext(ctx, `INSERT INTO learning_path_nodes(enrollment_id,position,knowledge_id,knowledge_version,knowledge_sha256) VALUES($1,$2,$3,$4,$5)`, enrollment, n.pos, n.k.ID, n.k.Version, n.k.SHA256); e != nil {
					return e
				}
				state, e := learningKnowledgeState(ctx, tx, u.ID, n.k)
				if e != nil {
					return e
				}
				if state.CanEnter {
					if _, e = learningUnlock(ctx, tx, u.ID, n.k, "enrollment", enrollment, now); e != nil {
						return e
					}
				}
			}
		}
		if e = learningRemember(ctx, tx, u.ID, string(learning.EnrollPathAction), id, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "enrollment", ResourceID: enrollment, Status: 201}); e != nil {
			return e
		}
		out.Summary, e = learningPathSummary(ctx, tx, u.ID, enrollment)
		return e
	})
	return out, e
}
func learningPathSummary(ctx context.Context, tx *sql.Tx, actor, id string) (learning.PathSummary, error) {
	var out learning.PathSummary
	e := tx.QueryRowContext(ctx, `SELECT e.id::text,e.path_id,e.path_version,e.path_sha256,p.body->>'title',p.body->>'titleZh',e.knowledge_publication_id,e.total_nodes,e.created_at,EXISTS(SELECT 1 FROM publication_heads h JOIN publication_members m ON m.snapshot_id=h.snapshot_id AND m.kind='path' AND m.availability='active' WHERE m.id=e.path_id AND m.version<>e.path_version) FROM learning_path_enrollments e JOIN path_versions p ON p.id=e.path_id AND p.version=e.path_version WHERE e.id=$1 AND e.owner_user_id=$2`, id, actor).Scan(&out.ID, &out.Path.ID, &out.Path.Version, &out.Path.SHA256, &out.Title, &out.TitleZh, &out.KnowledgePublicationID, &out.TotalNodes, &out.CreatedAt, &out.NewVersionAvailable)
	if e != nil {
		return out, workflowRowError(e)
	}
	out.CreatedAt = out.CreatedAt.UTC()
	rows, e := tx.QueryContext(ctx, `SELECT knowledge_id,knowledge_version,knowledge_sha256 FROM learning_path_nodes WHERE enrollment_id=$1 ORDER BY position`, id)
	if e != nil {
		return out, e
	}
	ks := []question.Identity{}
	for rows.Next() {
		var k question.Identity
		if e = rows.Scan(&k.ID, &k.Version, &k.SHA256); e != nil {
			rows.Close()
			return out, e
		}
		ks = append(ks, k)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	for _, k := range ks {
		completed, e := learningCurrentCompletion(ctx, tx, actor, k)
		if e != nil {
			return out, e
		}
		if completed != nil {
			out.CompletedNodes++
		}
		passed, e := learningEffectivePass(ctx, tx, actor, k)
		if e != nil {
			return out, e
		}
		if passed {
			out.PassedNodes++
		}
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM learning_path_nodes n JOIN learning_unlocks u ON u.owner_user_id=$2 AND u.knowledge_id=n.knowledge_id WHERE n.enrollment_id=$1`, id, actor).Scan(&out.UnlockedNodes); e != nil {
		return out, e
	}

	return out, nil
}
