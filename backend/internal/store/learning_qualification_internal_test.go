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

func (s *Store) LearningEvidenceForTest(ctx context.Context, a question.Access, k question.Identity) (learning.EvidenceView, error) {
	var out learning.EvidenceView
	e := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		var e error
		out, e = learningCurrentEvidence(ctx, tx, u.ID, k)
		return e
	})
	return out, e
}
func (s *Store) LearningApplyForTest(ctx context.Context, a question.Access, fact assessment.AttemptFact) (assessment.ProgressUpdate, error) {
	var out assessment.ProgressUpdate
	e := s.learningTx(ctx, a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		var e error
		out, e = learningApplyAssessment(ctx, tx, u, now, fact)
		return e
	})
	return out, e
}
