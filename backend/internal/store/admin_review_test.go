package store_test

import (
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"testing"
)

func grantAdminReviewer(f *workflowFixture, name string) {
	f.t.Helper()
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'admin'),($1,'reviewer') ON CONFLICT DO NOTHING`, f.ids[name])
}

func TestAdminReviewKnowledgeCanPublishOwnSubmission(t *testing.T) {
	f := newWorkflowFixture(t)
	grantAdminReviewer(f, "author_a")
	sub := f.Submitted("author_a")
	in := approvedReviewInput()
	in.IndependenceNote = "Administrator self-review; I also authored this submitted content."
	approved, err := f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, in)
	if err != nil || approved.Status != "approved" || approved.Review.ReviewerID != sub.OwnerID {
		t.Fatal("administrator self-review rejected or attribution lost", err)
	}
	published := f.Activate(f.Prepare(approved, nil), nil)
	view, err := f.repo.GetPublishedKnowledge(f.ctx, "workflow-fractions")
	if err != nil || view.Knowledge.ID != "workflow-fractions" || published.Status != "published" {
		t.Fatal("approved self-review did not become readable", err)
	}
}

func TestAdminReviewQuestionsCanPublishOwnSubmission(t *testing.T) {
	f := newQuestionFixture(t)
	grantAdminReviewer(f.workflowFixture, "author_a")
	sub := f.QSubmitted("author_a")
	in := approvedQuestionInput()
	in.IndependenceNote = "Administrator self-review; I also authored these submitted questions."
	approved, err := f.repo.DecideQuestionReview(f.ctx, f.Access("author_a", false), sub.ID, in)
	if err != nil || approved.Status != "approved" || approved.Review.ReviewerID != sub.OwnerID {
		t.Fatal("administrator question self-review rejected or attribution lost", err)
	}
	published := f.QActivate(f.QPrepare(approved.ID))
	if f.QHead() == nil || *f.QHead() != published.ID {
		t.Fatal("self-reviewed questions did not activate")
	}
}

func TestAdminReviewKnowledgeRoleRevocationBlocksActivation(t *testing.T) {
	f := newWorkflowFixture(t)
	grantAdminReviewer(f, "author_a")
	approved := f.Approved("author_a", "author_a")
	p := f.Prepare(approved, nil)
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, f.ids["author_a"])
	_, err := f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), p.ID, publication.ActivateInput{ExpectedHead: nil, ExpectedManifestSHA: p.ManifestSHA, Reason: "Check revoked self-review administrator qualification."})
	if !errors.Is(err, publication.ErrReviewRequired) || f.count(`SELECT count(*) FROM publication_heads`) != 0 {
		t.Fatal("revoked administrator self-review activated", err)
	}
}

func TestAdminReviewQuestionRoleRevocationBlocksActivation(t *testing.T) {
	f := newQuestionFixture(t)
	grantAdminReviewer(f.workflowFixture, "author_a")
	approved := f.QApproved("author_a", "author_a")
	p := f.QPrepare(approved.ID)
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, f.ids["author_a"])
	_, err := f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), p.ID, question.ActivateInput{ExpectedKnowledgeHead: p.BaseKnowledgeHead, ExpectedQuestionHead: p.BaseQuestionHead, ExpectedManifestSHA: p.ManifestSHA, Reason: "Check revoked question self-review administrator qualification."})
	if !errors.Is(err, question.ErrReviewRequired) || f.QHead() != nil {
		t.Fatal("revoked administrator question self-review activated", err)
	}
}

func TestAdminReviewStillRequiresReviewerRoleAndChecks(t *testing.T) {
	f := newWorkflowFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'admin')`, f.ids["author_a"])
	sub := f.Submitted("author_a")
	if _, err := f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedReviewInput()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("admin without reviewer approved", err)
	}
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	in := approvedReviewInput()
	in.Checks.Mathematics = false
	if _, err := f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, in); !errors.Is(err, publication.ErrContentNotReady) {
		t.Fatal("administrator bypassed mathematical checks", err)
	}
}

func TestAdminReviewOrdinaryAuthorCannotReturnOwnSubmission(t *testing.T) {
	f := newWorkflowFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	sub := f.Submitted("author_a")
	_, err := f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, publication.ReviewInput{Decision: "return", Note: "An author cannot act as the independent reviewer."})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("ordinary author returned their own submission", err)
	}
}
func TestAdminReviewOrdinaryQuestionAuthorCannotReturnOwnSubmission(t *testing.T) {
	f := newQuestionFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	sub := f.QSubmitted("author_a")
	_, err := f.repo.DecideQuestionReview(f.ctx, f.Access("author_a", false), sub.ID, question.ReviewInput{Decision: "return", Note: "An author cannot act as the independent question reviewer."})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("ordinary question author returned own submission", err)
	}
}
func TestAdminReviewMigrationRetainsPendingVersion(t *testing.T) {
	f := newWorkflowFixture(t)
	grantAdminReviewer(f, "author_a")
	sub := f.Submitted("author_a")
	p, err := goose.NewProvider(goose.DialectPostgres, f.db, os.DirFS("../../../db/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.DownTo(f.ctx, 8); err != nil {
		t.Fatal("empty administrator-review rollback failed", err)
	}
	if _, err = p.Up(f.ctx); err != nil {
		t.Fatal("administrator-review re-upgrade failed", err)
	}
	stored, err := f.repo.ReadSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if err != nil || stored.Revision != sub.Revision || stored.Frozen.FrozenDigest != sub.Frozen.FrozenDigest {
		t.Fatal("migration changed fixed pending content", err)
	}
	if _, err = f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedReviewInput()); err != nil {
		t.Fatal("existing pending content cannot use new rule", err)
	}
}
func TestAdminReviewMigrationRefusesToDiscardApprovedHistory(t *testing.T) {
	f := newWorkflowFixture(t)
	grantAdminReviewer(f, "author_a")
	sub := f.Approved("author_a", "author_a")
	p, err := goose.NewProvider(goose.DialectPostgres, f.db, os.DirFS("../../../db/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.DownTo(f.ctx, 8); err == nil {
		t.Fatal("rollback accepted administrator self-review history")
	}
	stored, err := f.repo.ReadSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if err != nil || stored.Status != "approved" || stored.Frozen.FrozenDigest != sub.Frozen.FrozenDigest {
		t.Fatal("rejected rollback damaged history", err)
	}
}
func TestAdminReviewPublishedHistorySurvivesRoleRemovalAndInheritance(t *testing.T) {
	f := newWorkflowFixture(t)
	grantAdminReviewer(f, "author_a")
	first := f.Activate(f.Prepare(f.Approved("author_a", "author_a"), nil), nil)
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role IN('admin','reviewer')`, f.ids["author_a"])
	in := f.Input()
	in.Package.ID = "later-package"
	in.Package.Knowledge[0].ID = "later-knowledge"
	in.Package.Units[0].ID = "later-unit"
	in.Package.Units[0].Knowledge.ID = "later-knowledge"
	in.Package.Assets = []content.Asset{}
	in.AssetBytes = []publication.AssetInput{}
	in.Package.Units[0].AssetIDs = []string{}
	in.Package.Units[0].Angles[0].Body = "An original formal explanation for an isolated continuation fixture."
	in.Package.Units[0].Angles[1].Body = "An original intuitive explanation for an isolated continuation fixture."
	in.SourceMap[0].Knowledge.ID = "later-knowledge"
	d, err := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), in)
	if err != nil {
		t.Fatal(err)
	}
	s, err := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), s.ID, approvedReviewInput())
	if err != nil {
		t.Fatal(err)
	}
	next := f.Activate(f.Prepare(approved, &first.ID), &first.ID)
	if next.ID == first.ID {
		t.Fatal("new content did not advance head")
	}
	for _, id := range []string{"workflow-fractions", "later-knowledge"} {
		if _, err = f.repo.GetPublishedKnowledge(f.ctx, id); err != nil {
			t.Fatal("published administrator history lost", id, err)
		}
	}
}
