package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLearningExposureRecordAccurateWatermarkAndRollback(t *testing.T) {
	s, db, a, owner := workflowGuardFixture(t)
	id := question.Identity{ID: "prepublication-template", Version: 1, SHA256: strings.Repeat("1", 64)}
	e := s.learningTx(context.Background(), a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		refs := []learning.ExposureRef{{Kind: "template", Identity: id}, {Kind: "template", Identity: id}}
		n, e := learningRecordExposure(ctx, tx, u.ID, refs, now)
		if e != nil {
			return e
		}
		if n != 1 {
			t.Fatal(n)
		}
		yes, e := learningExposureSince(ctx, tx, u.ID, 0, []assessment.ItemBinding{{Template: &id}})
		if e != nil || !yes {
			t.Fatal("accurate post-creation exposure missing", yes, e)
		}
		id.SHA256 = strings.Repeat("2", 64)
		yes, e = learningExposureSince(ctx, tx, u.ID, 0, []assessment.ItemBinding{{Template: &id}})
		if e != nil || yes {
			t.Fatal("different body matched exposure", yes, e)
		}
		id.SHA256 = strings.Repeat("1", 64)
		yes, e = learningExposureSince(ctx, tx, u.ID, 1, []assessment.ItemBinding{{Template: &id}})
		if e != nil || yes {
			t.Fatal("creation watermark ignored", yes, e)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1`, owner).Scan(&n); e != nil || n != 1 {
		t.Fatal("duplicate identity expanded", n, e)
	}
	e = s.learningTx(context.Background(), a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		_, e := learningRecordExposure(ctx, tx, u.ID, []learning.ExposureRef{{Kind: "template", Identity: id}}, now)
		if e != nil {
			return e
		}
		return auth.ErrForbidden
	})
	if !errors.Is(e, auth.ErrForbidden) {
		t.Fatal(e)
	}
	var seq int64
	if e = db.QueryRow(`SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$1`, owner).Scan(&seq); e != nil || seq != 1 {
		t.Fatal("failed response left partial exposure", seq, e)
	}
}
func TestLearningExposureLegacyQuestionDelivery(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.UpTo(ctx, 5); e != nil {
		t.Fatal(e)
	}
	owner := "11111111-1111-4111-8111-111111111111"
	var a question.Access
	a.TokenHash[0] = 1
	a.CSRF[0] = 2
	seed, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer seed.Rollback()
	if _, e = seed.Exec(`INSERT INTO auth_users(id,username,password_phc) VALUES($1,'legacy_editor','isolated-test-only')`, owner); e != nil {
		t.Fatal(e)
	}
	if _, e = seed.Exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'learner'),($1,'editor')`, owner); e != nil {
		t.Fatal(e)
	}
	if _, e = seed.Exec(`INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,absolute_expires_at) VALUES($1,$2,1,$3,clock_timestamp()+interval '1 hour')`, a.TokenHash[:], owner, a.CSRF[:]); e != nil {
		t.Fatal(e)
	}
	if e = seed.Commit(); e != nil {
		t.Fatal(e)
	}
	called := false
	e = New(db).questionAnswerReadTx(ctx, a, question.ReadDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		called = true
		return questionExposeResponse(ctx, tx, u, question.DraftView{QuestionPackage: question.QuestionPackage{Templates: []question.Template{{ID: "legacy-template", Version: 1}}}}, now)
	})
	if e != nil || !called {
		t.Fatal("never-enabled P4a answer delivery must remain available", called, e)
	}
}

func TestLearningExposureQuestionReadRechecksPermission(t *testing.T) {
	s, db, a, owner := workflowGuardFixture(t)
	e := s.questionAnswerReadTx(context.Background(), a, question.ReadDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		if e := questionExposeResponse(ctx, tx, u, question.DraftView{QuestionPackage: question.QuestionPackage{Templates: []question.Template{{ID: "permission-template", Version: 1}}}}, now); e != nil {
			return e
		}
		_, e := tx.ExecContext(ctx, `DELETE FROM auth_user_roles WHERE user_id=$1 AND role='editor'`, owner)
		return e
	})
	if !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("stale answer read proof committed", e)
	}
	var n int
	if e = db.QueryRow(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1`, owner).Scan(&n); e != nil || n != 0 {
		t.Fatal("failed read left exposure", n, e)
	}
}

func TestLearningExposureUsesDeliveryDatabaseTime(t *testing.T) {
	s, db, a, owner := workflowGuardFixture(t)
	var prepared time.Time
	e := s.learningTx(context.Background(), a, learning.ReadKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, entered time.Time) error {
		if e := tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&prepared); e != nil {
			return e
		}
		_, e := learningRecordExposure(ctx, tx, u.ID, []learning.ExposureRef{{Kind: "template", Identity: question.Identity{ID: "prepared-template", Version: 1, SHA256: strings.Repeat("5", 64)}}}, entered.Add(-time.Hour))
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	var recorded time.Time
	if e = db.QueryRow(`SELECT exposed_at FROM learner_answer_exposures WHERE owner_user_id=$1`, owner).Scan(&recorded); e != nil || recorded.Before(prepared) {
		t.Fatal("exposure used entry/supplied time rather than prepared delivery", recorded, prepared, e)
	}
}
