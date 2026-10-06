package store_test

import (
	"context"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
)

func TestStudyWithoutQuestionBank(t *testing.T) {
	f := newStudyFixture(t)
	f.ResetQuestionTablesForIsolation()
	got, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0))
	if e != nil || got.Record.State != study.Learning {
		t.Fatal("new study depends on question bank", got, e)
	}
	if f.CountEvents("author_a", "started") != 1 {
		t.Fatal("missing explicit start")
	}
}
func TestStudyMissingSchemaFailClosed(t *testing.T) {
	for _, damage := range []string{"table", "guard", "marker"} {
		t.Run(damage, func(t *testing.T) {
			f := newStudyFixture(t)
			switch damage {
			case "table":
				f.exec("ALTER TABLE study_events RENAME TO hidden_study_events")
			case "guard":
				f.exec("ALTER TABLE study_events DISABLE TRIGGER study_event_immutable")
			case "marker":
				f.exec("ALTER TABLE goose_db_version RENAME COLUMN topic_study_enabled TO hidden_topic_study_enabled")
			}
			_, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0))
			if !errors.Is(e, study.ErrNotConfigured) {
				t.Fatal("damaged study writes allowed", e)
			}
			if f.count("SELECT count(*) FROM study_records") != 0 {
				t.Fatal("damaged study committed")
			}
		})
	}
}
func TestStudyCrossActorKeyIsolation(t *testing.T) {
	f := newStudyFixture(t)
	a, b := f.Access("author_a"), f.Access("author_b")
	b.IdempotencyKey = a.IdempotencyKey
	ga, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, f.Command(0))
	if e != nil {
		t.Fatal(e)
	}
	gb, e := f.repo.BeginStudy(f.ctx, b, f.Ref().ID, f.Command(0))
	if e != nil {
		t.Fatal(e)
	}
	if ga.ActorID != f.ids["author_a"] || gb.ActorID != f.ids["author_b"] || ga.ActorID == gb.ActorID {
		t.Fatal("actor receipts mixed")
	}
	if f.CountEvents("author_a", "started") != 1 || f.CountEvents("author_b", "started") != 1 {
		t.Fatal("actor event isolation")
	}
	changed := f.Command(1)
	if _, e = f.repo.BeginStudy(f.ctx, a, f.Ref().ID, changed); !errors.Is(e, study.ErrIdempotencyConflict) {
		t.Fatal("same key changed input accepted", e)
	}
}
func TestStudyReplayCommitIdentity(t *testing.T) {
	f := newStudyFixture(t)
	a := f.Access("author_a")
	in := f.Command(0)
	first, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in)
	if e != nil {
		t.Fatal(e)
	}
	second, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in)
	if e != nil || first.Record.Sequence != second.Record.Sequence || !first.Record.FirstStartedAt.Equal(*second.Record.FirstStartedAt) {
		t.Fatal("replay changed facts", e)
	}
	f.exec("UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1", a.TokenHash[:])
	if _, e = f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("revoked proof replayed", e)
	}
	if f.CountEvents("author_a", "started") != 1 {
		t.Fatal("replay appended")
	}
}
func TestStudySchemaEmptyRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	var n int
	if e := db.QueryRow("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('study_records','study_events','study_notes','study_idempotency','study_content_changes')").Scan(&n); e != nil || n != 5 {
		t.Fatal("study tables missing", n, e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(ctx, 10); e != nil {
		t.Fatal(e)
	}
	var enabled bool
	if e = db.QueryRow("SELECT topic_study_enabled FROM goose_db_version WHERE version_id=0").Scan(&enabled); e != nil || !enabled {
		t.Fatal("permanent study marker lost", e)
	}
	if _, e = db.Exec("UPDATE goose_db_version SET topic_study_enabled=false WHERE version_id=0"); e == nil {
		t.Fatal("study marker cleared")
	}
	if e = store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
}
func TestStudyNonemptyDownDenied(t *testing.T) {
	f := newStudyFixture(t)
	if _, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0)); e != nil {
		t.Fatal(e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, f.db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(f.ctx, 10); e == nil {
		t.Fatal("nonempty study removed")
	}
	if f.CountEvents("author_a", "started") != 1 {
		t.Fatal("failed down altered event")
	}
}

func TestStudyReplayCommitIdentityRechecksAtCommit(t *testing.T) {
	for _, tt := range []struct {
		name, sql string
		want      error
	}{
		{"session", "UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE user_id=NEW.owner_user_id", auth.ErrAuthenticationRequired},
		{"credential", "UPDATE auth_users SET credential_version=credential_version+1 WHERE id=NEW.owner_user_id", auth.ErrAuthenticationRequired},
		{"password", "UPDATE auth_users SET must_change_password=true WHERE id=NEW.owner_user_id", auth.ErrPasswordChangeRequired},
		{"role", "DELETE FROM auth_user_roles WHERE user_id=NEW.owner_user_id AND role='learner'", auth.ErrForbidden},
		{"expires", "UPDATE auth_sessions SET absolute_expires_at=clock_timestamp() WHERE user_id=NEW.owner_user_id", auth.ErrAuthenticationRequired},
		{"csrf", "UPDATE auth_sessions SET csrf=decode(repeat('03',32),'hex') WHERE user_id=NEW.owner_user_id", auth.ErrCSRF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newStudyFixture(t)
			f.exec("CREATE FUNCTION change_study_proof() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN " + tt.sql + "; RETURN NEW; END $$")
			f.exec("CREATE TRIGGER change_study_proof AFTER INSERT ON study_records FOR EACH ROW EXECUTE FUNCTION change_study_proof()")
			_, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0))
			if !errors.Is(e, tt.want) {
				t.Fatal("commit accepted changed proof", e)
			}
			if f.count("SELECT count(*) FROM study_records") != 0 || f.count("SELECT count(*) FROM study_events") != 0 || f.count("SELECT count(*) FROM study_idempotency") != 0 {
				t.Fatal("changed proof committed private facts")
			}
		})
	}
}
