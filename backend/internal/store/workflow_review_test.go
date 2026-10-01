package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"testing"
)

func approvedReviewInput() publication.ReviewInput {
	return publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Independent technical test accounts verified for this fixture.", Note: "This is a technical approval in an isolated database."}
}
func TestCopiedAuthorsCannotApprove(t *testing.T) {
	f := newWorkflowFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	sub := f.Submitted("author_a")
	if _, err := f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedReviewInput()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("author approved own content", err)
	}
	adopted, err := f.repo.AdoptDraft(f.ctx, f.Access("author_b", false), publication.AdoptInput{PackageID: sub.Frozen.Package.ID, PackageVersion: sub.Frozen.Package.Version, Reason: "Copy while retaining the original author's responsibility."})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), adopted.ID, publication.SubmitInput{ExpectedRevision: adopted.Revision, ExpectedDigest: adopted.Gate.Digest})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.DecideReview(f.ctx, f.Access("author_a", false), copied.ID, approvedReviewInput()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("copy erased author independence", err)
	}
	approved, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), copied.ID, approvedReviewInput())
	if err != nil || approved.Status != "approved" || approved.Review == nil || approved.Review.FrozenDigest != copied.Frozen.FrozenDigest {
		t.Fatal(approved, err)
	}
}
func TestReviewChecksAndReturn(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	bad := approvedReviewInput()
	bad.Checks.Mathematics = false
	if _, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, bad); !errors.Is(err, publication.ErrContentNotReady) {
		t.Fatal("unchecked content approved", err)
	}
	bad = approvedReviewInput()
	bad.IndependenceNote = ""
	if _, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, bad); !errors.Is(err, publication.ErrContentNotReady) {
		t.Fatal("independence omitted", err)
	}
	command := publication.ReviewInput{Decision: "return", Note: "Clarify the mathematical assumptions in the examples."}
	a := f.Access("reviewer_a", false)
	returned, err := f.repo.DecideReview(f.ctx, a, sub.ID, command)
	if err != nil || returned.Status != "returned" || returned.Review == nil {
		t.Fatal(returned, err)
	}
	replay, err := f.repo.DecideReview(f.ctx, a, sub.ID, command)
	if err != nil || replay.Review.ID != returned.Review.ID {
		t.Fatal("return replay changed decision", err)
	}
	draft, err := f.repo.ReadDraft(f.ctx, f.Access("author_a", false), sub.WorkspaceID)
	if err != nil || draft.Status != "editing" || draft.Revision != 2 {
		t.Fatal("return did not restore editable revision", err)
	}
	if _, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), sub.ID, approvedReviewInput()); !errors.Is(err, publication.ErrReviewConflict) {
		t.Fatal("terminal decision changed", err)
	}
	command.Note = "A different command cannot reuse the earlier operation key."
	if _, err = f.repo.DecideReview(f.ctx, a, sub.ID, command); !errors.Is(err, publication.ErrIdempotencyConflict) {
		t.Fatal("changed replay accepted", err)
	}
	if f.count(`SELECT count(*) FROM content_review_decisions`) != 1 || f.count(`SELECT count(*) FROM content_workflow_events WHERE action='decideReview'`) != 1 {
		t.Fatal("return/replay duplicated audit")
	}
}
func TestReviewChecksAuditFailureRollsBack(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	f.exec(`CREATE FUNCTION isolated_fail_review_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='decideReview' THEN RAISE EXCEPTION 'isolated forced audit failure'; END IF; RETURN NEW; END $$`)
	f.exec(`CREATE TRIGGER isolated_review_audit BEFORE INSERT ON content_workflow_events FOR EACH ROW EXECUTE FUNCTION isolated_fail_review_audit()`)
	if _, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, publication.ReviewInput{Decision: "return", Note: "Return for a technical rollback verification."}); err == nil {
		t.Fatal("failed audit committed review")
	}
	saved, err := f.repo.ReadSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if err != nil || saved.Status != "pending" || saved.Review != nil {
		t.Fatal("failed review partially committed")
	}
	draft, err := f.repo.ReadDraft(f.ctx, f.Access("author_a", false), sub.WorkspaceID)
	if err != nil || draft.Status != "submitted" || draft.Revision != 1 {
		t.Fatal("audit failure reopened workspace")
	}
	if f.count(`SELECT count(*) FROM content_review_decisions`) != 0 {
		t.Fatal("audit failure kept decision")
	}
}

func TestReviewChecksCanonicalPayloadBoundary(t *testing.T) {
	f := newWorkflowFixture(t)
	v := f.Input()
	link := v.SourceMap[0]
	for len(v.SourceMap) < 100 {
		v.SourceMap = append(v.SourceMap, link)
	}
	views := []content.AssetView{{ID: v.Package.Assets[0].ID, SHA256: v.Package.Assets[0].SHA256, Author: v.Package.Assets[0].Author, License: v.Package.Assets[0].License, Attribution: v.Package.Assets[0].Attribution, Knowledge: v.Package.Assets[0].Knowledge}}
	frozen := publication.FrozenBody{CatalogueVersion: 1, Package: v.Package, SourceMap: v.SourceMap, AuthorIDs: []string{f.ids["author_a"]}, Assets: views}
	if err := f.db.QueryRowContext(f.ctx, `SELECT sha256 FROM catalogue_versions WHERE version=1`).Scan(&frozen.CatalogueSHA256); err != nil {
		t.Fatal(err)
	}
	raw, err := publication.FrozenBytes(frozen)
	if err != nil {
		t.Fatal(err)
	}
	n := ((4 << 20) - 1600 - len(raw)) / 2
	v.Package.Assets[0].Attribution += strings.Repeat("x", n)
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), v)
	if err != nil {
		t.Fatal("bounded fixture create failed", err)
	}
	sub, err := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if err != nil {
		t.Fatal("bounded fixture submit failed", err)
	}
	var storedSize int
	if err = f.db.QueryRowContext(f.ctx, `SELECT octet_length(frozen_body::text) FROM content_submissions WHERE id=$1`, sub.ID).Scan(&storedSize); err != nil || storedSize <= 4<<20 {
		t.Fatal("fixture did not cross JSONB whitespace boundary", storedSize, err)
	}
	if _, err = f.repo.ReadSubmission(f.ctx, f.Access("reviewer_a", false), sub.ID); err != nil {
		t.Fatalf("canonical payload below 4 MiB rejected because JSONB adds whitespace: %v", err)
	}
}
