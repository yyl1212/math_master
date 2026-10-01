package store_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"reflect"
	"testing"
)

func TestWorkflowSubmitAtomic(t *testing.T) {
	f := newWorkflowFixture(t)
	for _, kind := range []string{"proof", "body"} {
		v := f.Input()
		if kind == "proof" {
			v.Package.Knowledge[0].Type = "theorem"
			v.Package.Knowledge[0].Proof = ""
		} else {
			v.Package.Knowledge[0].Statement = ""
		}
		d, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest}); !errors.Is(err, publication.ErrContentNotReady) {
			t.Fatal("incomplete submitted", err)
		}
	}
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), f.Input())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: "bad"}); !errors.Is(err, publication.ErrDraftConflict) {
		t.Fatal("stale digest accepted", err)
	}
	a := f.Access("author_a", false)
	command := publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest}
	sub, err := f.repo.SubmitDraft(f.ctx, a, d.ID, command)
	if err != nil || sub.Status != "pending" || sub.Frozen.FrozenDigest == "" {
		t.Fatal(sub, err)
	}
	again, err := f.repo.SubmitDraft(f.ctx, a, d.ID, command)
	if err != nil || again.ID != sub.ID {
		t.Fatal("submission duplicated", err)
	}
	saved, err := f.repo.ReadDraft(f.ctx, f.Access("author_a", false), d.ID)
	if err != nil || saved.Status != "submitted" {
		t.Fatal("workspace not frozen")
	}
	if f.count(`SELECT count(*) FROM content_submissions`) != 1 || f.count(`SELECT count(*) FROM content_submission_members WHERE submission_id=$1`, sub.ID) != 3 {
		t.Fatal("wrong frozen member set")
	}
	if f.count(`SELECT count(*) FROM publication_heads`) != 0 {
		t.Fatal("submission auto-published")
	}
	f2 := newWorkflowFixture(t)
	d, err = f2.repo.CreateDraft(f2.ctx, f2.Access("author_a", false), f2.Input())
	if err != nil {
		t.Fatal(err)
	}
	events := f2.count(`SELECT count(*) FROM content_workflow_events`)
	packages := f2.count(`SELECT count(*) FROM imported_packages`)
	f2.exec(`ALTER TABLE content_submissions ADD CONSTRAINT isolated_late_failure CHECK(false)`)
	if _, err = f2.repo.SubmitDraft(f2.ctx, f2.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest}); err == nil {
		t.Fatal("late failure committed")
	}
	if f2.count(`SELECT count(*) FROM content_submissions`) != 0 || f2.count(`SELECT count(*) FROM imported_packages`) != packages || f2.count(`SELECT count(*) FROM content_workflow_events`) != events {
		t.Fatal("late failure left partial data")
	}
}
func TestFrozenSubmissionDoesNotReadMutableWorkspace(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	a := f.Access("author_a", false)
	old, err := f.repo.ReadSubmissionAsset(f.ctx, a, sub.ID, sub.Frozen.Assets[0].SHA256)
	if err != nil {
		t.Fatal(err)
	}
	f.DecideFixture(sub, "reviewer_a", "return")
	d, err := f.repo.ReadDraft(f.ctx, a, sub.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	v := newMaterial(f.Input())
	v.Package.Knowledge[0].Statement = "A changed, still safe, rational number statement."
	v.SourceMap[0].Note = "A later source mapping must not alter the first submission."
	changed := append(append([]byte{}, old...), []byte("\n")...)
	v.Package.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(changed))
	v.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString(changed)
	if _, err = f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SaveDraftInput{DraftInput: v, ExpectedRevision: d.Revision}); err != nil {
		t.Fatal(err)
	}
	frozen, err := f.repo.ReadSubmission(f.ctx, a, sub.ID)
	if err != nil || !reflect.DeepEqual(frozen.Frozen, sub.Frozen) {
		t.Fatal("frozen body/source/authors changed", err)
	}
	current, err := f.repo.ReadSubmissionAsset(f.ctx, a, sub.ID, sub.Frozen.Assets[0].SHA256)
	if err != nil || !bytes.Equal(current, old) {
		t.Fatal("frozen SVG changed")
	}
	if _, err = f.db.ExecContext(f.ctx, `INSERT INTO content_submission_authors VALUES($1,$2)`, sub.ID, f.ids["author_b"]); err == nil {
		t.Fatal("frozen authors extended")
	}
	if _, err = f.repo.ReadSubmission(f.ctx, f.Access("author_b", false), sub.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other editor read private submission")
	}
	if _, err = f.repo.ReadSubmission(f.ctx, f.Access("reviewer_a", false), sub.ID); err != nil {
		t.Fatal("reviewer cannot read frozen content")
	}
}
func TestWorkflowAuthorInheritance(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	adopted, err := f.repo.AdoptDraft(f.ctx, f.Access("author_b", false), publication.AdoptInput{PackageID: sub.Frozen.Package.ID, PackageVersion: sub.Frozen.Package.Version, Reason: "Take responsibility while keeping prior known authors."})
	if err != nil {
		t.Fatal(err)
	}
	contains := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	if !contains(adopted.AuthorIDs, f.ids["author_a"]) || !contains(adopted.AuthorIDs, f.ids["author_b"]) {
		t.Fatal("known authors lost on adoption")
	}
	v := newMaterial(f.Input())
	v.SourceMap[0].Note = "Client text claiming author " + f.ids["admin_a"] + " is data only."
	revised, err := f.repo.SaveDraft(f.ctx, f.Access("author_b", false), adopted.ID, publication.SaveDraftInput{DraftInput: v, ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(revised.AuthorIDs, f.ids["author_a"]) || contains(revised.AuthorIDs, f.ids["admin_a"]) {
		t.Fatal("base responsibility lost or text became authority")
	}
	page, err := f.repo.ListSubmissions(f.ctx, f.Access("reviewer_a", false), publication.ListQuery{Status: "pending"})
	if err != nil || len(page.Items) != 1 {
		t.Fatal("reviewer queue missing", err)
	}
}

func TestWorkflowNoteDatabaseBoundary(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(f.ctx, `INSERT INTO content_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,note) VALUES($1,$2,$3,$4,'approve','{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}','Independent technical fixture accounts.','a         ')`, f.ID(), sub.ID, f.ids["reviewer_a"], sub.Frozen.FrozenDigest); err != nil {
		t.Fatalf("database rejected a valid original-codepoint note: %v", err)
	}
	if _, err = tx.ExecContext(f.ctx, `UPDATE content_submissions SET status='approved' WHERE id=$1`, sub.ID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
