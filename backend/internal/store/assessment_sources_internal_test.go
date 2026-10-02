package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) LearningSourcePoolForTest(ctx context.Context, a question.Access, k question.Identity, bp *question.Identity) (assessment.SourcePool, error) {
	var out assessment.SourcePool
	err := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		var e error
		out, e = learningSourcePool(ctx, tx, u.ID, k, bp, now)
		return e
	})
	return out, err
}
func (s *Store) LearningLoadItemsForTest(ctx context.Context, a question.Access, seal assessment.Seal) ([]question.Instance, error) {
	var out []question.Instance
	err := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		var e error
		out, e = learningLoadItems(ctx, tx, seal)
		return e
	})
	return out, err
}
func (s *Store) LearningRestrictionsForTest(ctx context.Context, a question.Access, deps []learning.EvidenceDependency) ([]assessment.RestrictionReason, error) {
	var out []assessment.RestrictionReason
	err := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		var e error
		out, e = learningEvidenceRestrictions(ctx, tx, deps)
		return e
	})
	return out, err
}

// This probe runs the real bounded selection SQL and a five-id body join in the same authenticated read transaction.
func (s *Store) LearningCapacityPlansForTest(ctx context.Context, a question.Access, k question.Identity, bp *question.Identity, attemptID string) (map[string]string, error) {
	out := map[string]string{}
	e := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		pool, e := learningSourcePool(ctx, tx, u.ID, k, bp, now)
		if e != nil {
			return e
		}
		queries := []struct {
			name, sql string
			args      []any
		}{{"metadata-1000", "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) " + learningCandidateSQL, []any{u.ID, k.ID, k.Version, pool.KnowledgeHead, pool.QuestionHead, bp.ID, bp.Version}}}
		var sealRaw []byte
		if e = tx.QueryRowContext(ctx, `SELECT seal_bytes FROM assessment_attempts WHERE id=$1 AND owner_user_id=$2 AND sealed`, attemptID, u.ID).Scan(&sealRaw); e != nil {
			return e
		}
		seal, e := learningDecodeSeal(sealRaw)
		if e != nil {
			return e
		}
		queries = append(queries, struct {
			name, sql string
			args      []any
		}{"private-five-bodies", "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) " + learningItemsSQL, []any{body(seal.Items), seal.QuestionPublicationID}})

		for _, q := range queries {
			var raw []byte
			if e = tx.QueryRowContext(ctx, q.sql, q.args...).Scan(&raw); e != nil {
				return e
			}
			out[q.name] = string(raw)
		}
		return nil
	})
	return out, e
}
