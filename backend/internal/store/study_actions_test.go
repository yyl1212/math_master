package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"testing"
)

func (f *studyFixture) begin() study.StudyDetail {
	f.t.Helper()
	v, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0))
	if e != nil {
		f.t.Fatal(e)
	}
	return v
}
func (f *studyFixture) complete(seq int64) study.StudyDetail {
	f.t.Helper()
	v, e := f.repo.CompleteStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(seq))
	if e != nil {
		f.t.Fatal(e)
	}
	return v
}
func (f *studyFixture) upgrade() {
	f.t.Helper()
	raw, _ := json.Marshal(f.input)
	var in publication.DraftInput
	if json.Unmarshal(raw, &in) != nil {
		f.t.Fatal("fixture clone")
	}
	in = newMaterial(in)
	in.Package.Knowledge[0].Scope += " Precisely revised material."
	a := func(name string) publication.Access { return f.workflowFixture.Access(name, false) }
	d, e := f.repo.CreateDraft(f.ctx, a("author_a"), in)
	if e != nil {
		f.t.Fatal(e)
	}
	member := f.member
	member.Knowledge.Version = 2
	if _, e = f.repo.SaveDraftTopics(f.ctx, a("author_a"), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: d.Revision, TaxonomyVersionID: f.v.ID, Member: member}); e != nil {
		f.t.Fatal(e)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, a("author_a"), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		f.t.Fatal(e)
	}
	if _, e = f.repo.DecideReview(f.ctx, a("reviewer_a"), sub.ID, approvedReviewInput()); e != nil {
		f.t.Fatal(e)
	}
	p := f.preparePair(f.Pair(), sub.ID)
	f.activatePair(p)
	f.pair = taxonomy.PairRef{KnowledgeHead: p.KnowledgePublicationID, TaxonomyHead: &p.ID, TaxonomyVersionID: f.v.ID}
	if e = f.db.QueryRow("SELECT id,version,sha256 FROM knowledge_versions WHERE id=$1 AND version=2", f.ref.ID).Scan(&f.ref.ID, &f.ref.Version, &f.ref.SHA256); e != nil {
		f.t.Fatal(e)
	}
}
func TestStudyBeginReplay(t *testing.T) {
	f := newStudyFixture(t)
	a := f.Access("author_a")
	in := f.Command(0)
	first, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in)
	if e != nil {
		t.Fatal(e)
	}
	for _, access := range []study.Access{a, f.Access("author_a")} {
		again, e := f.repo.BeginStudy(f.ctx, access, f.Ref().ID, in)
		if e != nil || again.Record.Sequence != first.Record.Sequence || again.Record.State != study.Learning {
			t.Fatal("duplicate begin changed state", again, e)
		}
	}
	if f.CountEvents("author_a", "started") != 1 {
		t.Fatal("duplicate started")
	}
	completed := f.complete(1)
	again, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, in)
	if e != nil || again.Record.State != study.Completed || again.Record.Sequence != completed.Record.Sequence {
		t.Fatal("reread reset completion", again, e)
	}
}
func TestStudyCompletionKeepsFirstTime(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	first := f.complete(1)
	dup := f.complete(1)
	if dup.Record.Sequence != first.Record.Sequence || f.CountEvents("author_a", "completed") != 1 {
		t.Fatal("duplicate completion")
	}
	f.upgrade()
	second := f.complete(first.Record.Sequence)
	if second.Record.FirstCompletedAt == nil || !second.Record.FirstCompletedAt.Equal(*first.Record.FirstCompletedAt) || second.Record.LastCompletedAt == nil || second.Record.CompletedRef == nil || second.Record.CompletedRef.Version != 2 || f.CountEvents("author_a", "completed")+f.CountEvents("author_a", "started") != 3 {
		t.Fatal("completion history lost or duplicated", second)
	}
	if second.MaterialChanged {
		t.Fatal("freshly completed material remains stale")
	}
}
func TestStudyReviewSingleRound(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	f.complete(1)
	gate := make(chan struct{})
	type result struct {
		v study.StudyDetail
		e error
	}
	done := make(chan result, 2)
	for _, a := range []study.Access{f.Access("author_a"), f.Access("author_a")} {
		go func(a study.Access) {
			<-gate
			v, e := f.repo.StartStudyReview(f.ctx, a, f.Ref().ID, f.Command(2))
			done <- result{v, e}
		}(a)
	}
	close(gate)
	one, two := <-done, <-done
	if one.e != nil || two.e != nil || one.v.Record.ActiveReviewID == nil || two.v.Record.ActiveReviewID == nil || *one.v.Record.ActiveReviewID != *two.v.Record.ActiveReviewID || f.CountEvents("author_a", "review-started") != 1 {
		t.Fatal("competing reviews created rounds", one, two)
	}
	in := study.ReviewInput{CommandInput: f.Command(one.v.Record.Sequence), ReviewID: *one.v.Record.ActiveReviewID}
	finished, e := f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, in)
	if e != nil || finished.Record.State != study.Completed || finished.Record.ActiveReviewID != nil || finished.Record.LastReviewID == nil {
		t.Fatal(finished, e)
	}
	again, e := f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, in)
	if e != nil || again.Record.Sequence != finished.Record.Sequence || f.CountEvents("author_a", "review-finished") != 1 {
		t.Fatal("finish replay created an event", again, e)
	}
	in.ReviewID = f.ID()
	if _, e = f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, in); !errors.Is(e, study.ErrStateConflict) {
		t.Fatal("unrelated round finished", e)
	}
}
func TestStudyChangedHeadRejectsCompletion(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	old := f.Command(1)
	f.upgrade()
	if _, e := f.repo.CompleteStudy(f.ctx, f.Access("author_a"), f.Ref().ID, old); !errors.Is(e, study.ErrVersionStale) {
		t.Fatal("stale material completed", e)
	}
	if f.CountEvents("author_a", "completed") != 0 {
		t.Fatal("stale completion committed")
	}
}
func TestStudyReviewChangedMaterialCanBeReconfirmed(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	f.complete(1)
	round, e := f.repo.StartStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(2))
	if e != nil {
		t.Fatal(e)
	}
	old := study.ReviewInput{CommandInput: f.Command(round.Record.Sequence), ReviewID: *round.Record.ActiveReviewID}
	f.upgrade()
	if _, e = f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, old); !errors.Is(e, study.ErrVersionStale) {
		t.Fatal("changed material finished silently", e)
	}
	fresh := study.ReviewInput{CommandInput: f.Command(round.Record.Sequence), ReviewID: old.ReviewID}
	finished, e := f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, fresh)
	if e != nil || finished.Record.LastReviewRef == nil || *finished.Record.LastReviewRef != f.Ref() || finished.MaterialChanged {
		t.Fatal("review could not acknowledge current material", finished, e)
	}
	if f.CountEvents("author_a", "review-started") != 1 || f.CountEvents("author_a", "review-finished") != 1 {
		t.Fatal("reconfirmation created another round")
	}
}
func TestStudyCommitRevocationBlocksCompletion(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	a := f.Access("author_a")
	f.exec("UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1", a.TokenHash[:])
	if _, e := f.repo.CompleteStudy(f.ctx, a, f.Ref().ID, f.Command(1)); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("revoked session completed", e)
	}
	if f.CountEvents("author_a", "completed") != 0 {
		t.Fatal("revoked completion committed")
	}
}

func TestStudyBeginReplayReturnsCurrentFacts(t *testing.T) {
	f := newStudyFixture(t)
	a := f.Access("author_a")
	in := f.Command(0)
	if _, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in); e != nil {
		t.Fatal(e)
	}
	completed := f.complete(1)
	replayed, e := f.repo.BeginStudy(f.ctx, a, f.Ref().ID, in)
	if e != nil || replayed.Record.State != study.Completed || replayed.Record.Sequence != completed.Record.Sequence {
		t.Fatal("old receipt hid current completion", replayed, e)
	}
	if f.CountEvents("author_a", "started") != 1 || f.CountEvents("author_a", "completed") != 1 {
		t.Fatal("replay changed timeline")
	}
}
