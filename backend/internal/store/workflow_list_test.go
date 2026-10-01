package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"testing"
)

func TestContentListScopesAndIndependentQueue(t *testing.T) {
	f := newWorkflowFixture(t)
	own := f.Submitted("author_a")
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	page, err := f.repo.ListSubmissions(f.ctx, f.Access("author_a", false), publication.ListQuery{Scope: "review"})
	if err != nil || len(page.Items) != 0 {
		t.Fatal("review queue includes its frozen author", err, len(page.Items))
	}
	pending, err := f.repo.ListSubmissions(f.ctx, f.Access("reviewer_a", false), publication.ListQuery{Scope: "review"})
	if err != nil || len(pending.Items) != 1 {
		t.Fatal("explicit independent review scope unavailable", err)
	}
	all, err := f.repo.ListSubmissions(f.ctx, f.Access("admin_a", false), publication.ListQuery{})
	if err != nil || len(all.Items) != 1 {
		t.Fatal("admin default is not all", err)
	}
	drafts, err := f.repo.ListDrafts(f.ctx, f.Access("admin_a", false), publication.ListQuery{})
	if err != nil || len(drafts.Items) != 1 {
		t.Fatal("admin draft default is not all", err)
	}
	_, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), own.ID, approvedReviewInput())
	if err != nil {
		t.Fatal(err)
	}
	reviewed, err := f.repo.ListSubmissions(f.ctx, f.Access("reviewer_a", false), publication.ListQuery{Scope: "review", Status: "approved"})
	if err != nil || len(reviewed.Items) != 1 {
		t.Fatal("reviewer terminal history unavailable", err)
	}
	other, err := f.repo.ListSubmissions(f.ctx, f.Access("reviewer_b", false), publication.ListQuery{Scope: "review", Status: "approved"})
	if err != nil || len(other.Items) != 0 {
		t.Fatal("reviewer terminal scope exposed another decision", err)
	}
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["author_a"])
	if _, err = f.repo.ListSubmissions(f.ctx, f.Access("author_a", false), publication.ListQuery{Scope: "review"}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("editor-only account can choose review scope", err)
	}
}
