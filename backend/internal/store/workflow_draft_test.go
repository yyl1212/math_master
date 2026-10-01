package store_test

import (
	"encoding/base64"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"testing"
)

func TestWorkflowDraftOwnership(t *testing.T) {
	f := newWorkflowFixture(t)
	a := f.Access("author_a", false)
	d, err := f.repo.CreateDraft(f.ctx, a, f.Input())
	if err != nil || d.Revision != 1 || d.Status != "editing" {
		t.Fatal(d, err)
	}
	if _, err = f.repo.ReadDraft(f.ctx, f.Access("author_b", false), d.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other editor read draft", err)
	}
	input := publication.SaveDraftInput{DraftInput: f.Input(), ExpectedRevision: 1}
	if _, err = f.repo.SaveDraft(f.ctx, f.Access("author_b", false), d.ID, input); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other editor saved draft", err)
	}
	if _, err = f.repo.ReadDraft(f.ctx, f.Access("admin_a", false), d.ID); err != nil {
		t.Fatal("admin cannot read", err)
	}
	if _, err = f.repo.CreateDraft(f.ctx, f.Access("admin_a", false), f.Input()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("admin-only authored content")
	}
	input.Package.Knowledge[0].Scope = "Exact rational arithmetic with a nonzero denominator."
	d2, err := f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, input)
	if err != nil || d2.Revision != 2 {
		t.Fatal(d2, err)
	}
	if _, err = f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, input); !errors.Is(err, publication.ErrDraftConflict) {
		t.Fatal("stale save accepted", err)
	}
	d3, err := f.repo.ReadDraft(f.ctx, a, d.ID)
	if err != nil || d3.Revision != 2 || d3.Package.Knowledge[0].Scope != d2.Package.Knowledge[0].Scope {
		t.Fatal("stale save changed content")
	}
	if _, err = f.repo.ReadDraftAsset(f.ctx, f.Access("author_b", false), d.ID, d.Assets[0].SHA256); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("private SVG leaked")
	}
	if _, err = f.repo.ReadDraftAsset(f.ctx, a, d.ID, strings.Repeat("f", 64)); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("unbound private SVG available")
	}
	page, err := f.repo.ListDrafts(f.ctx, a, publication.ListQuery{Scope: "mine"})
	if err != nil || len(page.Items) != 1 || page.Total != 1 || page.Limit != 20 {
		t.Fatal(page, err)
	}
	if _, err = f.repo.ListDrafts(f.ctx, a, publication.ListQuery{Scope: "all"}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("editor used all scope")
	}
}
func TestWorkflowAdoptAndRevision(t *testing.T) {
	f := newWorkflowFixture(t)
	d, err := f.repo.AdoptDraft(f.ctx, f.Access("author_a", false), publication.AdoptInput{PackageID: "elementary-fractions", PackageVersion: 1, Reason: "Take responsibility for this legacy editorial draft."})
	if err != nil || !d.LegacyUnattributed || d.Status != "editing" || d.Gate.ReadyToSubmit {
		t.Fatal(d, err)
	}
	sub := f.Submitted("author_a")
	f.DecideFixture(sub, "reviewer_a", "approve")
	revision, err := f.repo.ReviseSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if err != nil || revision.ID == sub.WorkspaceID || revision.Revision != 1 || revision.Status != "editing" || len(revision.AuthorIDs) != 1 {
		t.Fatal(revision, err)
	}
	if _, err = f.repo.ReviseSubmission(f.ctx, f.Access("author_b", false), sub.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other owner revised submission")
	}
	large := input(t, func(p *content.Package) {
		p.ID = "oversized-legacy"
		k := p.Knowledge[0]
		k.ID = "large-legacy-number"
		k.Statement = strings.Repeat("x", 3<<20)
		k.Relations = []content.Relation{}
		p.Knowledge = []content.Knowledge{k}
		p.Units = []content.Unit{}
		p.Paths = []content.Path{}
		p.Assets = []content.Asset{}
	})
	if _, err = f.repo.ImportDraft(f.ctx, large); err != nil {
		t.Fatal(err)
	}
	before := f.count(`SELECT count(*) FROM content_workspaces`)
	if _, err = f.repo.AdoptDraft(f.ctx, f.Access("author_b", false), publication.AdoptInput{PackageID: "oversized-legacy", PackageVersion: 1, Reason: "Technical large legacy package adoption test."}); !errors.Is(err, publication.ErrContentLimitExceeded) {
		t.Fatal("oversized legacy accepted", err)
	}
	if f.count(`SELECT count(*) FROM content_workspaces`) != before {
		t.Fatal("oversized adoption created workspace")
	}
}
func TestWorkflowDraftInputBoundaries(t *testing.T) {
	f := newWorkflowFixture(t)
	for _, kind := range []string{"base64", "asset mismatch", "source path", "source digest", "nul", "catalogue"} {
		v := f.Input()
		switch kind {
		case "base64":
			v.AssetBytes[0].Base64 += "\n"
		case "asset mismatch":
			v.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString([]byte("wrong bytes"))
		case "source path":
			v.SourceMap[0].RelativePath = "../source.json"
		case "source digest":
			v.SourceMap[0].SHA256 = "untrusted"
		case "nul":
			v.SourceMap[0].Note = "bad\x00note"
		case "catalogue":
			v.CatalogueVersion = 0
		}
		before := f.count(`SELECT count(*) FROM content_workspaces`)
		if _, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), v); err == nil {
			t.Fatalf("%s accepted", kind)
		}
		if f.count(`SELECT count(*) FROM content_workspaces`) != before {
			t.Fatal("invalid draft committed")
		}
	}
}

func TestWorkflowAdoptNoteCountsOriginalCodepoints(t *testing.T) {
	f := newWorkflowFixture(t)
	if _, err := f.repo.AdoptDraft(f.ctx, f.Access("author_a", false), publication.AdoptInput{PackageID: "elementary-fractions", PackageVersion: 1, Reason: "a         "}); err != nil {
		t.Fatalf("ten codepoints with nonblank text rejected: %v", err)
	}
}

func TestWorkflowDraftValidationUsesCurrentRevision(t *testing.T) {
	f := newWorkflowFixture(t)
	a := f.Access("author_a", false)
	d, err := f.repo.CreateDraft(f.ctx, a, f.Input())
	if err != nil {
		t.Fatal(err)
	}
	report, err := f.repo.ValidateDraft(f.ctx, f.Access("author_a", false), d.ID, publication.ValidateInput{ExpectedRevision: 1})
	if err != nil || !report.ReadyToSubmit || report.Digest != d.Gate.Digest || report.HumanReviewTotal == 0 {
		t.Fatal("validation lost digest or pending human review", err)
	}
	if _, err = f.repo.ValidateDraft(f.ctx, f.Access("author_a", false), d.ID, publication.ValidateInput{ExpectedRevision: 2}); !errors.Is(err, publication.ErrDraftConflict) {
		t.Fatal("stale revision validated", err)
	}
	if f.count(`SELECT count(*) FROM content_idempotency WHERE route='validateDraft'`) != 0 {
		t.Fatal("validation wrote successful idempotency")
	}
	before := f.count(`SELECT count(*) FROM content_idempotency`)
	other := f.Access("author_a", false)
	other.CSRF[0]++
	if _, err = f.repo.ValidateDraft(f.ctx, other, d.ID, publication.ValidateInput{ExpectedRevision: 1}); !errors.Is(err, auth.ErrCSRF) {
		t.Fatal("validation ignored CSRF", err)
	}
	if f.count(`SELECT count(*) FROM content_idempotency`) != before {
		t.Fatal("failed validation recorded success")
	}
}
