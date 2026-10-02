package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestLearningIdempotencyBusinessReceiptAndConflict(t *testing.T) {
	s, _, a, id := workflowGuardFixture(t)
	ctx := context.Background()
	digest := strings.Repeat("a", 64)
	receipt := learning.Receipt{ResourceKind: "knowledge", ResourceID: "fractions", Status: 201}
	if e := s.learningTx(ctx, a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		old, ok, e := learningReplay(ctx, tx, id, "startKnowledge", "fractions", a.IdempotencyKey, digest)
		if e != nil || ok || old.ResourceID != "" {
			t.Fatal(old, ok, e)
		}
		return learningRemember(ctx, tx, id, "startKnowledge", "fractions", a.IdempotencyKey, digest, receipt)
	}); e != nil {
		t.Fatal(e)
	}
	if e := s.learningTx(ctx, a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		old, ok, e := learningReplay(ctx, tx, id, "startKnowledge", "fractions", a.IdempotencyKey, digest)
		if e != nil || !ok || old != receipt {
			t.Fatal(old, ok, e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	for _, tt := range []struct{ target, hash string }{{"other", digest}, {"fractions", strings.Repeat("b", 64)}} {
		e := s.learningTx(ctx, a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
			_, _, e := learningReplay(ctx, tx, id, "startKnowledge", tt.target, a.IdempotencyKey, tt.hash)
			return e
		})
		if !errors.Is(e, question.ErrIdempotencyConflict) {
			t.Fatal("key reused for other input", e)
		}
	}
}
func TestLearningIdempotencyLastWriteFailureRollsBack(t *testing.T) {
	s, db, a, id := workflowGuardFixture(t)
	e := s.learningTx(context.Background(), a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		if _, e := tx.ExecContext(ctx, `INSERT INTO learner_exposure_state(owner_user_id,sequence) VALUES($1,1)`, id); e != nil {
			return e
		}
		return learningRemember(ctx, tx, id, "startKnowledge", "fractions", a.IdempotencyKey, strings.Repeat("a", 64), learning.Receipt{ResourceKind: "knowledge", ResourceID: "fractions", Status: 503})
	})
	var n int
	db.QueryRow(`SELECT count(*) FROM learner_exposure_state`).Scan(&n)
	if e == nil || n != 0 {
		t.Fatal("failed final receipt retained changes", n, e)
	}
}
