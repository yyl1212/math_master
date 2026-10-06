package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func (f *studyFixture) note(rev int64, body string) study.NoteInput {
	return study.NoteInput{ExpectedRevision: rev, Knowledge: f.Ref(), Body: body}
}
func TestStudyNoteOwnerOnly(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	if _, e := f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(0, "My private derivation.")); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"author_b", "admin_a", "reviewer_a"} {
		if _, e := f.repo.ReadStudyNote(f.ctx, f.Access(name), f.Ref().ID); !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("other role read private note", name, e)
		}
	}
	v, e := f.repo.ReadStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID)
	if e != nil || v.Body != "My private derivation." || v.ActorID != f.ids["author_a"] {
		t.Fatal("owned note missing", e)
	}
}
func TestStudyNoteConflictingTabs(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	gate := make(chan struct{})
	done := make(chan error, 2)
	for _, a := range []study.Access{f.Access("author_a"), f.Access("author_a")} {
		go func(a study.Access) {
			<-gate
			_, e := f.repo.SaveStudyNote(f.ctx, a, f.Ref().ID, f.note(0, "A safe personal note."))
			done <- e
		}(a)
	}
	close(gate)
	success, conflict := 0, 0
	for range 2 {
		e := <-done
		if e == nil {
			success++
		} else if errors.Is(e, study.ErrNoteConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 || f.CountEvents("author_a", "note-saved") != 1 {
		t.Fatal("concurrent notes overwritten", success, conflict)
	}
}
func TestStudyNoteDeleteNoRecoverableBody(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	a := f.Access("author_a")
	in := f.note(0, "A private unique note that is deleted.")
	saved, e := f.repo.SaveStudyNote(f.ctx, a, f.Ref().ID, in)
	if e != nil {
		t.Fatal(e)
	}
	deleted, e := f.repo.DeleteStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, study.NoteDeleteInput{ExpectedRevision: saved.Revision})
	if e != nil || !deleted.Deleted || deleted.Revision != 2 {
		t.Fatal(deleted, e)
	}
	var body string
	if e = f.db.QueryRow("SELECT body FROM study_notes WHERE owner_user_id=$1 AND knowledge_id=$2", f.ids["author_a"], f.Ref().ID).Scan(&body); e != nil || body != "" {
		t.Fatal("deleted text retained", e)
	}
	if f.count("SELECT count(*) FROM study_idempotency WHERE receipt::text LIKE '%private unique note%'") != 0 || f.count("SELECT count(*) FROM study_events WHERE to_jsonb(study_events)::text LIKE '%private unique note%'") != 0 {
		t.Fatal("note copied into metadata")
	}
	if _, e = f.repo.SaveStudyNote(f.ctx, a, f.Ref().ID, in); e != nil {
		t.Fatal("same-key receipt could not be confirmed", e)
	}
	v, e := f.repo.ReadStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID)
	if e != nil || v.Body != "" || v.Revision != 2 {
		t.Fatal("old receipt recovered text", e)
	}
	if _, e = f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(0, "Stale input.")); !errors.Is(e, study.ErrNoteConflict) {
		t.Fatal("ABA overwrite accepted", e)
	}
	if _, e = f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(2, "Fresh personal note.")); e != nil {
		t.Fatal(e)
	}
}
func TestStudyNoteAfterWithdrawalAndChangedIdentity(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	if _, e := f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(0, "Private old material note.")); e != nil {
		t.Fatal(e)
	}
	_, e := f.repo.WithdrawVersion(f.ctx, f.workflowFixture.Access("admin_a", true), publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: f.Ref().ID, Version: 1}, ExpectedHead: f.Pair().KnowledgeHead, Reason: "Withdraw public fixture while preserving the owner's private notes."})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(1, "Private note after withdrawal.")); e != nil {
		t.Fatal("withdrawn note cannot be edited", e)
	}
	view, e := f.repo.ReadStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID)
	if e != nil || view.Body != "Private note after withdrawal." {
		t.Fatal(e)
	}
	f.exec("UPDATE auth_users SET must_change_password=true WHERE id=$1", f.ids["author_a"])
	if _, e = f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(2, "Must not commit.")); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("forced password allowed note write", e)
	}
}
func TestStudyNoteStorageBoundary(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	if _, e := f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(0, strings.Repeat("😀", 16000))); e != nil {
		t.Fatal("maximum note refused", e)
	}
	if _, e := f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, f.note(1, strings.Repeat("😀", 16001))); !errors.Is(e, study.ErrInvalid) {
		t.Fatal("oversized note saved", e)
	}
}

func TestStudyNoteWrongBindingAndRevokedReplay(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	in := f.note(0, "Private owned input.")
	wrong := in
	wrong.Knowledge.SHA256 = strings.Repeat("0", 64)
	if _, e := f.repo.SaveStudyNote(f.ctx, f.Access("author_a"), f.Ref().ID, wrong); !errors.Is(e, study.ErrVersionStale) {
		t.Fatal("unrelated knowledge bound to note", e)
	}
	a := f.Access("author_a")
	if _, e := f.repo.SaveStudyNote(f.ctx, a, f.Ref().ID, in); e != nil {
		t.Fatal(e)
	}
	f.exec("UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1", a.TokenHash[:])
	if _, e := f.repo.SaveStudyNote(f.ctx, a, f.Ref().ID, in); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("revoked proof replayed a note receipt", e)
	}
	if f.CountEvents("author_a", "note-saved") != 1 {
		t.Fatal("invalid note write appended metadata")
	}
}
