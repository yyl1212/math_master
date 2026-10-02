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
