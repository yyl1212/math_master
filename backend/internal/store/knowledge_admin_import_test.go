package store_test

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"sync"
	"testing"
)

func cloneManagedSource(t *testing.T, d knowledgeadmin.SourceDocument) knowledgeadmin.SourceDocument {
	t.Helper()
	b, _ := json.Marshal(d)
	var c knowledgeadmin.SourceDocument
	if e := json.Unmarshal(b, &c); e != nil {
		t.Fatal(e)
	}
	return c
}
func (f *knowledgeAdminFixture) Import(name, key string, d knowledgeadmin.SourceDocument, publish bool) knowledgeadmin.Receipt {
	f.t.Helper()
	p, e := f.repo.PreviewManagedImport(f.ctx, f.Access(name, key+"-preview"), d, managedInputSHA(d))
	if e != nil {
		f.t.Fatal(e)
	}
	selected := []int{}
	for _, i := range p.Items {
		if i.Action == "create" || i.Action == "link" {
			selected = append(selected, i.Index)
		}
	}
	r, e := f.repo.ApplyManagedImport(f.ctx, f.Access(name, key+"-apply"), p.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: selected, Publish: publish, PreviewToken: p.PreviewToken})
	if e != nil {
		f.t.Fatal(e)
	}
	return r
}
func TestManagedReplayPreservesCorrection(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	r := f.Import("admin_a", "first", f.doc, true)
	if r.Counts.CreatedKnowledge != 2 {
		t.Fatal(r)
	}
	id := knowledgeadmin.KnowledgeID(f.doc.KnowledgePoints[0].ID)
	k, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), id)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}
	in.Point.Statement += " 管理员补充条件说明。"
	in.Point.LearningDifficulty.Level = 2
	in.Point.MSCCodes = []string{"97F50"}
	in.Point.ClassificationEvidence[0].MSCCode = "97F50"
	edited, e := f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "edit"), id, k.EditToken, in)
	if e != nil {
		t.Fatal(e)
	}
	r = f.Import("admin_a", "replay", f.doc, true)
	if r.Counts.CreatedKnowledge != 0 || r.Counts.SkippedItems != 2 {
		t.Fatal(r)
	}
	after, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-after"), id)
	if e != nil || after.Point.Statement != edited.Point.Statement || after.Point.LearningDifficulty.Level != 2 {
		t.Fatal("correction overwritten", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge_topics WHERE internal_id=$1 AND topic_key='97F40' AND active", id) != 0 {
		t.Fatal("removed topic revived")
	}
	next := cloneManagedSource(t, f.doc)
	next.KnowledgePoints = next.KnowledgePoints[:1]
	next.KnowledgePoints[0].MSCCodes = []string{"97F60"}
	next.KnowledgePoints[0].ClassificationEvidence[0].MSCCode = "97F60"
	next.KnowledgePoints[0].Relations = []knowledgeadmin.Relation{}
	r = f.Import("admin_a", "new-topic", next, true)
	if r.Counts.CreatedKnowledge != 0 || r.Counts.LinkedTopics != 1 {
		t.Fatal(r)
	}
	if f.count("SELECT count(*) FROM managed_knowledge WHERE external_id=$1", after.ExternalID) != 1 {
		t.Fatal("second entity")
	}
	conflict := cloneManagedSource(t, next)
	conflict.KnowledgePoints[0].Statement += " 从未接收的新正文。"
	p, e := f.repo.PreviewManagedImport(f.ctx, f.Access("admin_a", "conflict"), conflict, managedInputSHA(conflict))
	if e != nil || p.Counts.Conflicts != 1 {
		t.Fatal(p, e)
	}
}
func TestManagedCrossSourceConcurrentImport(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	d := cloneManagedSource(t, f.doc)
	d.KnowledgePoints = d.KnowledgePoints[:1]
	d.KnowledgePoints[0].Relations = []knowledgeadmin.Relation{}
	d.KnowledgePoints = append(d.KnowledgePoints, d.KnowledgePoints[0])
	other := cloneManagedSource(t, d)
	other.Source.SourceID = "another-source"
	other.DatasetID = "another-dataset"
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for n, doc := range []knowledgeadmin.SourceDocument{d, other} {
		wg.Add(1)
		go func(n int, doc knowledgeadmin.SourceDocument) {
			defer wg.Done()
			name := "admin_a"
			if n == 1 {
				name = "admin_b"
			}
			p, e := f.repo.PreviewManagedImport(f.ctx, f.Access(name, "parallel-preview"), doc, managedInputSHA(doc))
			if e != nil {
				errs <- e
				return
			}
			ix := []int{}
			for _, v := range p.Items {
				if v.Action == "create" || v.Action == "link" {
					ix = append(ix, v.Index)
				}
			}
			_, e = f.repo.ApplyManagedImport(f.ctx, f.Access(name, "parallel-apply"), p.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: ix, Publish: true, PreviewToken: p.PreviewToken})
			errs <- e
		}(n, doc)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if f.count("SELECT count(*) FROM managed_knowledge") != 1 || f.count("SELECT count(*) FROM managed_knowledge_topics") != 1 || f.count("SELECT count(*) FROM managed_knowledge_events WHERE action='import'") != 1 {
		t.Fatal("concurrent duplication")
	}
	d.KnowledgePoints = d.KnowledgePoints[:1]
	d.KnowledgePoints[0].ID = "same-title-new-id"
	r := f.Import("admin_a", "new-id", d, true)
	if r.Counts.CreatedKnowledge != 1 {
		t.Fatal("title treated as identity", r)
	}
}

func managedInputSHA(d knowledgeadmin.SourceDocument) string {
	b, _ := json.Marshal(d)
	return knowledgeadmin.DigestBytes(b)
}
func TestManagedCrossSourceSupplementAndPreviewCounts(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	d := cloneManagedSource(t, f.doc)
	d.KnowledgePoints = d.KnowledgePoints[:1]
	d.KnowledgePoints[0].Relations = []knowledgeadmin.Relation{}
	f.Import("admin_a", "original", d, true)
	other := cloneManagedSource(t, d)
	other.Source.SourceID = "second-book"
	other.DatasetID = "second-dataset"
	r := f.Import("admin_a", "supplement", other, true)
	if r.Counts.CreatedKnowledge != 0 || r.Counts.SkippedItems != 1 {
		t.Fatal(r)
	}
	if f.count("SELECT count(*) FROM managed_knowledge_sources") != 2 {
		t.Fatal("actual second source was lost on a duplicate")
	}
	d.KnowledgePoints[0].ID = "three-occurrences"
	first := d.KnowledgePoints[0]
	second := cloneManagedSource(t, d).KnowledgePoints[0]
	second.MSCCodes = []string{"97F50"}
	second.ClassificationEvidence[0].MSCCode = "97F50"
	d.KnowledgePoints = []knowledgeadmin.SourcePoint{first, second, first}
	p, e := f.repo.PreviewManagedImport(f.ctx, f.Access("admin_a", "three-preview"), d, managedInputSHA(d))
	if e != nil {
		t.Fatal(e)
	}
	if p.Counts.CreatedKnowledge != 1 || p.Counts.LinkedTopics != 2 || p.Counts.SkippedItems != 1 || p.Items[2].Action != "skip" {
		t.Fatal("preview overcounts repeated association", p)
	}
}
func TestManagedCrossSourceOtherMembership(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	d := cloneManagedSource(t, f.doc)
	d.KnowledgePoints = d.KnowledgePoints[:1]
	d.KnowledgePoints[0].Relations = []knowledgeadmin.Relation{}
	f.Import("admin_a", "msc", d, true)
	p := &d.KnowledgePoints[0]
	p.ClassificationMode = "project_other"
	p.MSCCodes = []string{}
	p.ClassificationEvidence = []knowledgeadmin.ClassificationEvidence{}
	p.ProjectOther = &knowledgeadmin.ProjectOther{GroupID: "project-other", Reason: "经过核实的项目其他归属，用于检验不同主题关联仍然共用同一知识。", EvidenceFields: []string{"statement"}}
	r := f.Import("admin_a", "other-link", d, true)
	if r.Counts.LinkedTopics != 1 || r.Counts.CreatedKnowledge != 0 {
		t.Fatal(r)
	}
	k, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), knowledgeadmin.KnowledgeID(p.ID))
	if e != nil {
		t.Fatal(e)
	}
	if e = knowledgeadmin.ValidateCurrent(knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}); e != nil {
		t.Fatal("membership corrupts normalized source body", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge_topics WHERE internal_id=$1 AND active", k.ID) != 2 {
		t.Fatal("membership lost")
	}
}
