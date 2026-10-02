package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

// Test-only bridges keep historical and offerability helpers out of the P4a HTTP/Repository contract.
func (s *Store) TestQuestionOfferable(ctx context.Context, a question.Access, target question.WithdrawalTarget) (bool, error) {
	var out bool
	err := s.questionReadTx(ctx, a, question.ReadCoverageAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var e error
		out, e = questionOfferable(ctx, tx, target)
		return e
	})
	return out, err
}
func (s *Store) TestQuestionHistoricalFacts(ctx context.Context, a question.Access, target question.WithdrawalTarget, publicationID *string) (question.HistoricalFacts, error) {
	var out question.HistoricalFacts
	err := s.questionReadTx(ctx, a, question.ReadCoverageAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var e error
		out, e = questionHistoricalFacts(ctx, tx, target, publicationID)
		return e
	})
	return out, err
}
