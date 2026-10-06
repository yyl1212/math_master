package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"strings"
	"testing"
)

type taxonomyFixture struct {
	*workflowFixture
	version     taxonomy.Version
	releaseID   string
	knowledgeID string
}

func newTaxonomyFixture(t *testing.T) *taxonomyFixture {
	w := newWorkflowFixture(t)
	sub := w.Approved("author_a", "reviewer_a")
	w.Activate(w.Prepare(sub, nil), nil)
	v, e := w.repo.InstallTaxonomyBatch(w.ctx, taxonomyFixtureBatch())
	if e != nil {
		t.Fatal(e)
	}
	return &taxonomyFixture{workflowFixture: w, version: v, releaseID: w.ID(), knowledgeID: w.Input().Package.Knowledge[0].ID}
}
func (f *taxonomyFixture) publishSQLFixture(topics []string) {
	f.t.Helper()
	var head string
	if e := f.db.QueryRowContext(f.ctx, "SELECT snapshot_id FROM publication_heads WHERE singleton").Scan(&head); e != nil {
		f.t.Fatal(e)
	}
	f.exec("INSERT INTO taxonomy_releases(id,taxonomy_version_id,knowledge_publication_id,status,manifest_sha,assignments_sha,body,creator_user_id) VALUES($1,$2,$3,'draft',$4,$5,'{}',$6)", f.releaseID, f.version.ID, head, strings.Repeat("a", 64), strings.Repeat("b", 64), f.ids["admin_a"])
	for _, topic := range topics {
		f.exec("INSERT INTO taxonomy_release_assignments(release_id,taxonomy_version_id,knowledge_id,knowledge_version,knowledge_sha,topic_id) SELECT $1,$2,id,version,sha256,$3 FROM knowledge_versions WHERE id=$4 AND version=1", f.releaseID, f.version.ID, topic, f.knowledgeID)
	}
	f.exec("UPDATE taxonomy_releases SET status='published' WHERE id=$1", f.releaseID)
	f.exec("INSERT INTO taxonomy_heads VALUES(true,$1)", f.releaseID)
}
func TestTaxonomyAncestorCounts(t *testing.T) {
	f := newTaxonomyFixture(t)
	f.publishSQLFixture([]string{"msc-00a00", "msc-00a01"})
	root, e := f.repo.ReadTopic(f.ctx, "msc-00")
	if e != nil || root.Summary.PublishedKnowledgeCount != 1 {
		t.Fatal(root, e)
	}
	middle, e := f.repo.ReadTopic(f.ctx, "msc-00a")
	if e != nil || middle.Summary.PublishedKnowledgeCount != 1 {
		t.Fatal(middle, e)
	}
	leaf, e := f.repo.ReadTopic(f.ctx, "msc-00a00")
	if e != nil || len(leaf.Summary.Ancestors) != 2 {
		t.Fatal(leaf, e)
	}
	page, e := f.repo.ListTopicKnowledge(f.ctx, "msc-00", taxonomy.Query{})
	if e != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
	if strings.Contains(string(mustJSON(page)), "sourceRefs") {
		t.Fatal("private references leaked")
	}
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func TestTaxonomyQueriesIgnoreUnpublished(t *testing.T) {
	f := newTaxonomyFixture(t)
	if _, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Level: 1}); !errors.Is(e, taxonomy.ErrNotConfigured) {
		t.Fatal(e)
	}
	f.publishSQLFixture(nil)
	p, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Level: 1, Limit: 100})
	if e != nil || p.Total != 63 || len(p.Items) != 63 {
		t.Fatal(p.Total, e)
	}
	for _, n := range p.Items {
		if n.PublishedKnowledgeCount != 0 {
			t.Fatal("unassigned knowledge counted")
		}
	}
}
func TestTaxonomyBodySHAUnchanged(t *testing.T) {
	f := newTaxonomyFixture(t)
	var before, after string
	if e := f.db.QueryRow("SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1", f.knowledgeID).Scan(&before); e != nil {
		t.Fatal(e)
	}
	f.publishSQLFixture([]string{"msc-00a00"})
	if _, e := f.repo.ReadTopic(f.ctx, "msc-00"); e != nil {
		t.Fatal(e)
	}
	if e := f.db.QueryRow("SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1", f.knowledgeID).Scan(&after); e != nil || before != after {
		t.Fatal("mathematics rewritten", e)
	}
}
func TestTaxonomyQueryBoundaries(t *testing.T) {
	f := newTaxonomyFixture(t)
	f.publishSQLFixture(nil)
	p, e := f.repo.ListTopics(f.ctx, taxonomy.Query{Level: 1})
	if e != nil || p.Limit != 20 {
		t.Fatal(p, e)
	}
	for _, q := range []taxonomy.Query{{Limit: 101}, {Q: strings.Repeat("汉", 171)}, {Q: "a\x00b"}, {ParentID: "missing"}, {Offset: -1}} {
		if _, e = f.repo.ListTopics(f.ctx, q); e == nil {
			t.Fatal("invalid query accepted")
		}
	}
	p, e = f.repo.ListTopics(f.ctx, taxonomy.Query{Q: "%_"})
	if e != nil || p.Total != 0 {
		t.Fatal("wildcards used as search syntax", e)
	}
	_, e = f.repo.ReadTopic(f.ctx, "msc-nonexistent")
	if !errors.Is(e, store.ErrNotFound) {
		t.Fatal(e)
	}
}
