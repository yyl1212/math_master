package store_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"reflect"
	"testing"
)

type topicWorkflowFixture struct {
	*workflowFixture
	v      taxonomy.Version
	input  publication.DraftInput
	member taxonomy.AssignmentInput
}

func newTopicWorkflowFixture(t *testing.T) *topicWorkflowFixture {
	f := newWorkflowFixture(t)
	b := taxonomyFixtureBatch()
	in := f.Input()
	source := in.SourceMap[0]
	ref := taxonomy.SourceRecordRef{SourceID: "fixture-source", WorkFamilyID: "fixture-work", RecordID: source.LegacyID, Path: source.RelativePath, SHA256: source.SHA256}
	b.Manifest.SourceFiles = append(b.Manifest.SourceFiles, taxonomy.SourceFile{Path: "Knowledge_JSON/" + ref.Path, SizeBytes: 2, SHA256: ref.SHA256})
	files, _ := json.Marshal(b.Manifest.SourceFiles)
	h := sha256.Sum256(files)
	b.Manifest.SnapshotID = hex.EncodeToString(h[:])
	b.SourceRecordIndex = []taxonomy.SourceRecordRef{ref}
	v, e := f.repo.InstallTaxonomyBatch(f.ctx, b)
	if e != nil {
		t.Fatal(e)
	}
	in.SourceMap[0].BatchSHA256 = v.SnapshotID
	member := taxonomy.AssignmentInput{Knowledge: source.Knowledge, TopicIDs: []string{"msc-00a00"}, SourceRefs: []taxonomy.SourceRecordRef{ref}, SourceBatchSHA: v.SnapshotID}
	return &topicWorkflowFixture{workflowFixture: f, v: v, input: in, member: member}
}
func (f *topicWorkflowFixture) draft() publication.DraftView {
	d, e := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), f.input)
	if e != nil {
		f.t.Fatal(e)
	}
	return d
}
func (f *topicWorkflowFixture) save(d publication.DraftView) taxonomy.DraftTopicView {
	got, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: d.Revision, ExpectedAssignmentRevision: 0, TaxonomyVersionID: f.v.ID, Member: f.member})
	if e != nil {
		f.t.Fatal(e)
	}
	return got
}
func TestTopicAssignmentOwnerOnly(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	in := taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, ExpectedAssignmentRevision: 0, TaxonomyVersionID: f.v.ID, Member: f.member}
	if _, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_b", false), d.ID, in); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("other owner writes", e)
	}
	got := f.save(d)
	if !got.ReadyToSubmit || len(got.Members) != 1 {
		t.Fatal(got)
	}
}
func TestTopicAssignmentStaleRevision(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	in := publication.SaveDraftInput{DraftInput: f.input, ExpectedRevision: 1}
	in.Package.Knowledge[0].Scope += " More precise scope."
	d2, e := f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d2.Revision, ExpectedDigest: d2.Gate.Digest}); !errors.Is(e, publication.ErrContentNotReady) {
		t.Fatal("stale classification submitted", e)
	}
}
func TestTopicAssignmentDisabledGuardFailsClosed(t *testing.T) {
	for _, action := range []string{"save", "submit", "review"} {
		t.Run(action, func(t *testing.T) {
			f := newTopicWorkflowFixture(t)
			d := f.draft()
			f.save(d)
			var sub publication.SubmissionView
			if action == "review" {
				var e error
				sub, e = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
				if e != nil {
					t.Fatal(e)
				}
			}
			if action == "review" {
				f.exec("ALTER TABLE taxonomy_review_bindings DISABLE TRIGGER taxonomy_review_immutable")
			} else {
				f.exec("ALTER TABLE taxonomy_submission_assignments DISABLE TRIGGER taxonomy_submission_immutable")
			}
			var e error
			switch action {
			case "save":
				in := publication.SaveDraftInput{DraftInput: f.input, ExpectedRevision: d.Revision}
				in.Package.Knowledge[0].Scope += " Changed scope."
				_, e = f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, in)
			case "submit":
				_, e = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
			case "review":
				_, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedReviewInput())
			}
			if !errors.Is(e, taxonomy.ErrNotConfigured) {
				t.Fatal("disabled guard silently bypassed taxonomy", e)
			}
			var count int
			if action == "review" {
				if e = f.db.QueryRow("SELECT count(*) FROM content_review_decisions WHERE submission_id=$1", sub.ID).Scan(&count); e != nil {
					t.Fatal(e)
				}
			} else {
				if e = f.db.QueryRow("SELECT count(*) FROM content_submissions WHERE workspace_id=$1", d.ID).Scan(&count); e != nil {
					t.Fatal(e)
				}
			}
			if count != 0 {
				t.Fatal("failure committed workflow evidence", count)
			}
			if action == "save" {
				var revision int64
				if e = f.db.QueryRow("SELECT revision FROM content_workspaces WHERE id=$1", d.ID).Scan(&revision); e != nil {
					t.Fatal(e)
				}
				if revision != d.Revision {
					t.Fatal("failed save committed", revision)
				}
			}
		})
	}
}
func TestTopicFrozenAssignmentIgnoresWorkspace(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	frozen, e := f.repo.ReadSubmissionTopics(f.ctx, f.Access("reviewer_a", false), sub.ID)
	if e != nil {
		t.Fatal(e)
	}
	if frozen.Digest == "" || !frozen.ReadyToSubmit {
		t.Fatal(frozen)
	}
	if _, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedReviewInput()); e != nil {
		t.Fatal(e)
	}
	var bound string
	if e = f.db.QueryRow("SELECT digest FROM taxonomy_review_bindings WHERE submission_id=$1", sub.ID).Scan(&bound); e != nil || bound != frozen.Digest {
		t.Fatal(e)
	}
	newDraft, e := f.repo.ReviseSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if e != nil {
		t.Fatal(e)
	}
	original, _ := json.Marshal(f.input)
	var cloned publication.DraftInput
	if json.Unmarshal(original, &cloned) != nil {
		t.Fatal("fixture clone")
	}
	changed := publication.SaveDraftInput{DraftInput: cloned, ExpectedRevision: newDraft.Revision}
	changed.Package.Knowledge[0].Scope += " Updated revision only."
	if _, e = f.repo.SaveDraft(f.ctx, f.Access("author_a", false), newDraft.ID, changed); e != nil {
		t.Fatal(e)
	}
	after, e := f.repo.ReadSubmissionTopics(f.ctx, f.Access("reviewer_a", false), sub.ID)
	if e != nil || after.Digest != frozen.Digest || !reflect.DeepEqual(after.Members, frozen.Members) {
		t.Fatal("frozen topics changed", e)
	}
	if !reflect.DeepEqual(sub.Frozen.Package, f.input.Package) {
		t.Fatal("mathematical frozen body rewritten")
	}
}
func TestTopicAuthorReviewBoundary(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	f.exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')", f.ids["author_a"])
	if _, e = f.repo.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedReviewInput()); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("ordinary author self-reviewed", e)
	}
}
func TestTopicAssignmentSourceMustBeCaptured(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	v := f.member
	v.SourceRefs = append([]taxonomy.SourceRecordRef(nil), v.SourceRefs...)
	v.SourceRefs[0].RecordID = "invented"
	if _, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, TaxonomyVersionID: f.v.ID, Member: v}); !errors.Is(e, taxonomy.ErrInvalid) {
		t.Fatal("invented source typed error", e)
	}
}

func TestTopicAssignmentUnchangedKnowledgeRebinds(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	in := publication.SaveDraftInput{DraftInput: f.input, ExpectedRevision: 1}
	in.Package.Units[0].Examples = append(in.Package.Units[0].Examples, "Another original example.")
	d2, e := f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	v, e := f.repo.ReadDraftTopics(f.ctx, f.Access("author_a", false), d.ID)
	if e != nil || !v.ReadyToSubmit || v.DraftRevision != d2.Revision {
		t.Fatal(v, e)
	}
	_, e = f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, ExpectedAssignmentRevision: 1, TaxonomyVersionID: f.v.ID, Member: f.member})
	if !errors.Is(e, publication.ErrDraftConflict) {
		t.Fatal("stale write", e)
	}
}
func TestTopicAdministratorAuthorReviewBoundary(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	f.exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer'),($1,'admin')", f.ids["author_a"])
	_, e = f.repo.DecideReview(f.ctx, f.Access("author_a", true), sub.ID, approvedReviewInput())
	if e != nil {
		t.Fatal(e)
	}
	var n int
	if e = f.db.QueryRow("SELECT count(*) FROM taxonomy_review_bindings WHERE submission_id=$1", sub.ID).Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
}
func TestTopicAssignmentRemovedKnowledgeNotFrozen(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	f.save(d)
	changed := f.input
	changed.Package.Knowledge[0].ID = "numbers-replaced"
	changed.Package.Units[0].Knowledge.ID = "numbers-replaced"
	changed.Package.Assets[0].Knowledge.ID = "numbers-replaced"
	changed.SourceMap[0].Knowledge.ID = "numbers-replaced"
	d2, e := f.repo.SaveDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SaveDraftInput{DraftInput: changed, ExpectedRevision: 1})
	if e != nil {
		t.Fatal(e)
	}
	member := f.member
	member.Knowledge.ID = "numbers-replaced"
	v, e := f.repo.SaveDraftTopics(f.ctx, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: d2.Revision, ExpectedAssignmentRevision: 1, TaxonomyVersionID: f.v.ID, Member: member})
	if e != nil || len(v.Members) != 1 || v.Members[0].Knowledge.ID != "numbers-replaced" {
		t.Fatal("removed knowledge still frozen", v, e)
	}
}
func TestTopicAssignmentEmptyViewUsesInstalledTaxonomy(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	d := f.draft()
	v, e := f.repo.ReadDraftTopics(f.ctx, f.Access("author_a", false), d.ID)
	if e != nil || v.TaxonomyVersionID != f.v.ID || v.ReadyToSubmit || len(v.Members) != 0 {
		t.Fatal("empty assignment view violates the catalogue contract", v, e)
	}
}
