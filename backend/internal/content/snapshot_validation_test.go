package content

import (
	"context"
	"strings"
	"testing"
)

func TestSnapshotGraph(t *testing.T) {
	c, p, reader := workflowSeed(t)
	ctx := context.Background()
	snapshot := func(p Package) Snapshot {
		bindings := []AssetBinding{}
		for _, u := range p.Units {
			for _, id := range u.AssetIDs {
				bindings = append(bindings, AssetBinding{Unit: VersionRef{u.ID, u.Version}, AssetID: id, SHA256: p.Assets[0].SHA256})
			}
		}
		return Snapshot{CatalogueVersion: c.Version, Knowledge: p.Knowledge, Units: p.Units, Paths: p.Paths, Assets: p.Assets, Bindings: bindings}
	}
	if r, err := ValidateSnapshot(ctx, c, snapshot(p), reader); err != nil || !r.ReadyToSubmit {
		t.Fatal("valid snapshot rejected")
	}
	for _, kind := range []string{"missing", "cycle", "version", "binding"} {
		q := clone(p)
		switch kind {
		case "missing":
			q.Knowledge[0].Relations = []Relation{{"prerequisite", VersionRef{"missing", 1}}}
		case "cycle":
			q.Knowledge[0].Relations = []Relation{{"prerequisite", VersionRef{q.Knowledge[0].ID, 1}}}
		case "version":
			q.Units[0].Knowledge.Version = 2
		case "binding":
			q.Assets[0].SHA256 = strings.Repeat("a", 64)
		}
		if _, err := ValidateSnapshot(ctx, c, snapshot(q), reader); err == nil {
			t.Fatalf("%s snapshot accepted", kind)
		}
	}
	q := clone(p)
	q.Knowledge[0].Relations = []Relation{{"related", VersionRef{"missing", 1}}}
	if _, err := ValidateSnapshot(ctx, c, snapshot(q), reader); err != nil {
		t.Fatal("non-prerequisite reference blocked snapshot")
	}
	second := clone(p.Knowledge[0])
	second.ID = "later"
	second.Relations = []Relation{{"prerequisite", VersionRef{q.Knowledge[0].ID, 1}}}
	q = clone(p)
	q.Knowledge = append(q.Knowledge, second)
	u := clone(q.Units[0])
	u.ID = "later-unit"
	u.Knowledge.ID = "later"
	u.AssetIDs = []string{}
	u.Angles[0].Body = "Two equal parts form a whole."
	q.Units = append(q.Units, u)
	q.Paths = []Path{{ID: "ordered", Version: 1, DomainIDs: q.Knowledge[0].DomainIDs, Title: "An ordered path", TitleZh: "学习路线", Nodes: []VersionRef{{"later", 1}, {q.Knowledge[0].ID, 1}}}}
	if _, err := ValidateSnapshot(ctx, c, snapshot(q), reader); err == nil {
		t.Fatal("out-of-order path accepted")
	}
	q = clone(p)
	q.Knowledge[0].Statement = strings.Repeat("x", 10<<20)
	if _, err := ValidateSnapshot(ctx, c, snapshot(q), reader); err == nil {
		t.Fatal("oversized knowledge view accepted")
	}
	q = clone(p)
	q.Knowledge[0].Statement = strings.Repeat("x", 6<<20)
	second = clone(q.Knowledge[0])
	second.ID = "second"
	q.Knowledge = append(q.Knowledge, second)
	u = clone(q.Units[0])
	u.ID = "second-unit"
	u.Knowledge.ID = "second"
	u.AssetIDs = []string{}
	u.Angles[0].Body = "No illustration required."
	q.Units = append(q.Units, u)
	q.Paths = []Path{{ID: "large-path", Version: 1, DomainIDs: q.Knowledge[0].DomainIDs, Title: "Large path", TitleZh: "大路线", Nodes: []VersionRef{{q.Knowledge[0].ID, 1}, {"second", 1}}}}
	if _, err := ValidateSnapshot(ctx, c, snapshot(q), reader); err == nil {
		t.Fatal("oversized path view accepted")
	}
}
