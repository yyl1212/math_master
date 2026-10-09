package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func (f *knowledgeAdminFixture) ActivateManaged() {
	f.exec("UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp() WHERE singleton")
	f.exec("UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0")
}
func TestManagedPublicCurrentVisibility(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "public-seed", f.doc, true)
	f.ActivateManaged()
	id := knowledgeadmin.KnowledgeID(f.doc.KnowledgePoints[0].ID)
	page, e := f.repo.ListCurrentKnowledge(f.ctx, knowledgeadmin.Query{})
	if e != nil || page.Total != 2 || len(page.Items) != 2 {
		t.Fatal(page, e)
	}
	pub, e := f.repo.ReadCurrentKnowledge(f.ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if _, exists := pub.Point["version"]; exists {
		t.Fatal("public business version")
	}
	if _, exists := pub.Point["original_binding"]; exists {
		t.Fatal("private source binding")
	}
	k, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), id)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}
	in.Point.Statement += " 更新立即可读。"
	k, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "update"), id, k.EditToken, in)
	if e != nil {
		t.Fatal(e)
	}
	pub, e = f.repo.ReadCurrentKnowledge(f.ctx, id)
	if e != nil || pub.Point["statement"] != in.Point.Statement {
		t.Fatal("stale body", e)
	}
	_, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "hide"), id, k.EditToken, "unpublish")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReadCurrentKnowledge(f.ctx, id); !errors.Is(e, knowledgeadmin.ErrNotFound) {
		t.Fatal("unpublished body disclosed", e)
	}
	page, e = f.repo.ListCurrentKnowledge(f.ctx, knowledgeadmin.Query{})
	if e != nil || page.Total != 1 {
		t.Fatal(page, e)
	}
	if f.count("SELECT count(*) FROM managed_study_records") != 0 {
		t.Fatal("public read wrote learning")
	}
}
func TestManagedPublicDistinctTopicCount(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	d := cloneManagedSource(t, f.doc)
	d.KnowledgePoints = d.KnowledgePoints[:1]
	d.KnowledgePoints[0].Relations = []knowledgeadmin.Relation{}
	f.Import("admin_a", "first", d, true)
	d.KnowledgePoints[0].MSCCodes = []string{"97F50"}
	d.KnowledgePoints[0].ClassificationEvidence[0].MSCCode = "97F50"
	f.Import("admin_a", "link", d, true)
	f.ActivateManaged()
	p, e := f.repo.ListCurrentKnowledge(f.ctx, knowledgeadmin.Query{TopicKey: "97F"})
	if e != nil || p.Total != 1 || len(p.Items[0].TopicKeys) != 2 {
		t.Fatal("ancestor double count", p, e)
	}
	if _, e = f.repo.ListCurrentKnowledge(f.ctx, knowledgeadmin.Query{Limit: 101}); !errors.Is(e, knowledgeadmin.ErrInvalid) {
		t.Fatal("unbounded public list", e)
	}
}

func TestManagedPublicDirectoryAndOtherGroups(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	batch := testutil.TaxonomyBatch()
	for n := range batch.Nodes {
		if batch.Nodes[n].Code == "97F00" {
			batch.Nodes[n].Code = "97F40"
			batch.Nodes[n].ID = taxonomy.TopicID("97F40")
		}
	}
	v, e := f.repo.InstallTaxonomyBatch(f.ctx, batch)
	if e != nil {
		t.Fatal(e)
	}
	const release = "f1205495-cf1d-4a43-8dcf-f895e126b10a"
	f.exec(`INSERT INTO taxonomy_releases(id,taxonomy_version_id,status,manifest_sha,assignments_sha,body,creator_user_id) VALUES($1,$2,'draft',repeat('c',64),repeat('d',64),'{}',$3)`, release, v.ID, f.ids["admin_a"])
	f.exec(`UPDATE taxonomy_releases SET status='published' WHERE id=$1`, release)
	f.exec(`INSERT INTO taxonomy_heads(singleton,release_id) VALUES(true,$1)`, release)
	f.Import("admin_a", "directory-points", f.doc, true)
	f.ActivateManaged()
	roots, e := f.repo.ListCurrentTopics(f.ctx, knowledgeadmin.Query{Limit: 100})
	if e != nil || roots.Total != 64 || len(roots.Items) != 64 {
		t.Fatal("formal roots plus explicit project other", roots.Total, e)
	}
	parent, e := f.repo.ReadCurrentTopic(f.ctx, "97-XX", knowledgeadmin.Query{})
	if e != nil || parent.KnowledgeCount != 2 {
		t.Fatal("ancestor count", parent, e)
	}
	leaf, e := f.repo.ReadCurrentTopic(f.ctx, "97F40", knowledgeadmin.Query{})
	if e != nil || leaf.KnowledgeCount != 2 {
		t.Fatal("leaf", leaf, e)
	}
	other, e := f.repo.ReadCurrentTopic(f.ctx, "project-other", knowledgeadmin.Query{})
	if e != nil || other.Kind != "project-other" || other.KnowledgeCount != 0 {
		t.Fatal(other, e)
	}
}
