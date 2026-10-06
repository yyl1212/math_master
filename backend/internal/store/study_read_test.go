package store_test

import (
	"encoding/json"
	"fmt"

	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"strings"
	"testing"
)

func (f *studyFixture) publishInput(in publication.DraftInput, member taxonomy.AssignmentInput) {
	f.t.Helper()
	a := func(name string) publication.Access { return f.workflowFixture.Access(name, false) }
	d, e := f.repo.CreateDraft(f.ctx, a("author_a"), in)
	if e != nil {
		f.t.Fatal(e)
	}
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
}
func (f *studyFixture) addPoint(id string) study.KnowledgeRef {
	f.t.Helper()
	raw, _ := json.Marshal(f.input)
	var in publication.DraftInput
	if json.Unmarshal(raw, &in) != nil {
		f.t.Fatal("clone")
	}
	in.Package.ID = "package-" + id
	in.Package.Knowledge[0].ID = id
	in.Package.Knowledge[0].Title = "Percent % and underscore _"
	in.Package.Knowledge[0].TitleZh = "百分数_记号"
	in.Package.Units[0].ID = "unit-" + id
	in.Package.Units[0].Knowledge.ID = id
	assetID := "asset-" + id
	oldID := in.Package.Assets[0].ID
	in.Package.Assets[0].ID = assetID
	in.Package.Assets[0].Knowledge.ID = id
	in.AssetBytes[0].ID = assetID
	in.Package.Units[0].AssetIDs = []string{assetID}
	for i := range in.Package.Units[0].Angles {
		in.Package.Units[0].Angles[i].Body = strings.ReplaceAll(in.Package.Units[0].Angles[i].Body, "asset:"+oldID, "asset:"+assetID)
	}
	in.SourceMap[0].Knowledge.ID = id
	member := f.member
	member.Knowledge.ID = id
	f.publishInput(in, member)
	ref := study.KnowledgeRef{ID: id, Version: 1}
	if e := f.db.QueryRow("SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1", id).Scan(&ref.SHA256); e != nil {
		f.t.Fatal(e)
	}
	return ref
}
func (f *studyFixture) progress(topic string) study.TopicProgress {
	f.t.Helper()
	page, e := f.repo.ListStudyTopics(f.ctx, f.Access("author_a"), study.ListQuery{TopicID: topic})
	if e != nil || len(page.Items) != 1 {
		f.t.Fatal("progress missing", page, e)
	}
	return page.Items[0]
}
func TestStudyReviewPreservesProgress(t *testing.T) {
	f := newStudyFixture(t)
	f.addPoint("study-second")
	f.begin()
	completed := f.complete(1)
	before := f.progress("msc-00")
	if before.Total != 2 || before.Completed != 1 || before.CompletedRatio == nil || *before.CompletedRatio != 0.5 {
		t.Fatal(before)
	}
	if _, e := f.repo.StartStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(completed.Record.Sequence)); e != nil {
		t.Fatal(e)
	}
	after := f.progress("msc-00")
	if after.Completed != 1 || after.Total != 2 || after.Reviewing != 1 || *after.CompletedRatio != 0.5 {
		t.Fatal("review reduced completed facts", after)
	}
}
func TestStudyZeroDenominatorRatioNull(t *testing.T) {
	f := newStudyFixture(t)
	p := f.progress("msc-01")
	if p.Total != 0 || p.CompletedRatio != nil {
		t.Fatal(p)
	}
}
func TestStudyAncestorDedup(t *testing.T) {
	f := newStudyFixture(t)
	member := f.member
	member.TopicIDs = []string{"msc-00a00", "msc-00a01"}
	f.publishInput(f.input, member)
	f.begin()
	f.complete(1)
	for _, topic := range []string{"msc-00", "msc-00a", "msc-00a00", "msc-00a01"} {
		p := f.progress(topic)
		if p.Total != 1 || p.Completed != 1 {
			t.Fatal("ancestor duplicated membership", p)
		}
	}
}
func TestStudyTopicMoveKeepsRecord(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	first := f.complete(1)
	member := f.member
	member.TopicIDs = []string{"msc-13c60"}
	f.publishInput(f.input, member)
	got, e := f.repo.ReadStudyKnowledge(f.ctx, f.Access("author_a"), f.Ref().ID)
	if e != nil || got.Record.State != study.Completed || got.Record.Sequence != first.Record.Sequence || got.MaterialChanged {
		t.Fatal("classification move reset record", got, e)
	}
	old, newp := f.progress("msc-00"), f.progress("msc-13")
	if old.Total != 0 || old.Removed != 1 || newp.Total != 1 || newp.Completed != 1 || newp.Added != 1 {
		t.Fatal("denominator change invisible", old, newp)
	}
}
func TestStudyReviewSearchOwnerOnly(t *testing.T) {
	f := newStudyFixture(t)
	second := f.addPoint("study-second")
	f.begin()
	other := study.CommandInput{Knowledge: second, ExpectedKnowledgeHead: *f.Pair().KnowledgeHead, ExpectedSequence: 0}
	if _, e := f.repo.BeginStudy(f.ctx, f.Access("author_b"), second.ID, other); e != nil {
		t.Fatal(e)
	}
	page, e := f.repo.ListStudyKnowledge(f.ctx, f.Access("author_a"), study.ListQuery{ReviewOnly: true})
	if e != nil || page.Total != 1 || page.Items[0].Record.KnowledgeID != f.Ref().ID {
		t.Fatal("review search leaked unstudied/other actor", page, e)
	}
	for _, q := range []string{"百分", "%", "_"} {
		p, e := f.repo.ListStudyKnowledge(f.ctx, f.Access("author_a"), study.ListQuery{Q: q})
		if e != nil || p.Total != 1 || p.Items[0].Record.KnowledgeID != second.ID {
			t.Fatal("literal title search", q, p, e)
		}
	}
	if _, e = f.repo.ListStudyKnowledge(f.ctx, f.Access("author_a"), study.ListQuery{Q: strings.Repeat("中", 171)}); e == nil {
		t.Fatal("byte limit ignored")
	}
}
func TestStudyReminderNoFanout(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	f.complete(1)
	for i := 0; i < 20; i++ {
		f.register(fmt.Sprintf("study_extra_%02d", i))
	}
	before := f.count("SELECT count(*) FROM study_content_changes")
	f.upgrade()
	if f.count("SELECT count(*) FROM study_content_changes")-before != 1 {
		t.Fatal("change events fanned out")
	}
	events := f.count("SELECT count(*) FROM study_events")
	view, e := f.repo.ReadStudyOverview(f.ctx, f.Access("author_a"))
	if e != nil || len(view.Reminders) != 1 || view.Reminders[0].Kind != "updated" || view.Completed != 1 {
		t.Fatal(view, e)
	}
	if f.count("SELECT count(*) FROM study_events") != events {
		t.Fatal("GET appended events")
	}
	if _, e = f.repo.ReadStudyOverview(f.ctx, study.Access{}); e == nil {
		t.Fatal("anonymous private summary")
	}

}

func TestStudyReminderWithdrawnKeepsPrivateSummary(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	f.complete(1)
	a := f.workflowFixture.Access("admin_a", true)
	_, e := f.repo.WithdrawVersion(f.ctx, a, publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: f.Ref().ID, Version: 1}, ExpectedHead: f.Pair().KnowledgeHead, Reason: "Preserve private study facts while withdrawing original fixture content."})
	if e != nil {
		t.Fatal(e)
	}
	detail, e := f.repo.ReadStudyKnowledge(f.ctx, f.Access("author_a"), f.Ref().ID)
	if e != nil || detail.Available || detail.CurrentKnowledge != nil || detail.Record.State != study.Completed {
		t.Fatal("withdrawal destroyed private summary", detail, e)
	}
	overview, e := f.repo.ReadStudyOverview(f.ctx, f.Access("author_a"))
	if e != nil || overview.Unavailable != 1 || len(overview.Reminders) != 1 || overview.Reminders[0].Kind != "withdrawn" || overview.Reminders[0].CurrentRef != nil {
		t.Fatal(overview, e)
	}
}
func TestStudyHistoryCursorAndOwnershipBoundaries(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	for _, q := range []study.HistoryQuery{{Cursor: "bad"}, {Cursor: string([]byte{0xff})}, {Limit: 51}, {Kind: "started|completed"}, {KnowledgeID: "../private"}} {
		if _, e := f.repo.ListStudyHistory(f.ctx, f.Access("author_a"), q); e == nil {
			t.Fatal("invalid history accepted", q)
		}
	}
	page, e := f.repo.ListStudyKnowledge(f.ctx, f.Access("author_b"), study.ListQuery{ReviewOnly: true})
	if e != nil || page.ActorID != f.ids["author_b"] || page.Total != 0 {
		t.Fatal("owned search crossed actors", page, e)
	}
}
