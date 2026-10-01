package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestSnapshotPublicViewIncludesOwnedUnusedAssets(t *testing.T) {
	c, p, reader := workflowSeed(t)
	p.Units[0].AssetIDs = []string{}
	p.Units[0].Angles[0].Body = "An illustration can belong to the knowledge without being linked by this unit."
	p.Assets[0].Attribution = strings.Repeat("x", 10<<20)
	snapshot := Snapshot{CatalogueVersion: c.Version, Knowledge: p.Knowledge, Units: p.Units, Paths: p.Paths, Assets: p.Assets, Bindings: []AssetBinding{}}
	if _, err := ValidateSnapshot(context.Background(), c, snapshot, reader); err == nil {
		t.Fatal("owned unused asset metadata bypassed public view limit")
	}
}

func TestSnapshotPublicResponseEnvelope(t *testing.T) {
	c, p, reader := workflowSeed(t)
	base := p.Units[0]
	p.Units = []Unit{}
	for i := 0; i < 6; i++ {
		u := clone(base)
		u.ID = fmt.Sprintf("large-unit-%d", i)
		p.Units = append(p.Units, u)
	}
	a := p.Assets[0]
	view := KnowledgeView{Knowledge: p.Knowledge[0], Units: p.Units, Assets: []AssetView{{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge}}}
	raw, _ := json.Marshal(map[string]any{"data": view})
	padding := (10 << 20) - len(raw) - 1
	for i := range p.Units {
		n := padding / 6
		if i < padding%6 {
			n++
		}
		p.Units[i].Angles[1].Body += strings.Repeat("x", n)
	}
	// Each immutable unit can be submitted with this same small knowledge owner.
	for _, u := range p.Units {
		fragment := p
		fragment.Units = []Unit{u}
		fragment.Paths = []Path{}
		if _, r := ValidateWorkflow(context.Background(), c, fragment, reader); !r.ReadyToSubmit {
			t.Fatal("public boundary cannot originate from legal batches")
		}
	}
	snapshot := Snapshot{CatalogueVersion: c.Version, Knowledge: p.Knowledge, Units: p.Units, Paths: []Path{}, Assets: p.Assets, Bindings: []AssetBinding{}}
	for _, u := range p.Units {
		snapshot.Bindings = append(snapshot.Bindings, AssetBinding{Unit: VersionRef{ID: u.ID, Version: u.Version}, AssetID: a.ID, SHA256: a.SHA256})
	}
	if r, err := ValidateSnapshot(context.Background(), c, snapshot, reader); err != nil || !r.ReadyToSubmit {
		t.Fatal("exact 10 MiB public response rejected", err)
	}
	snapshot.Units[0].Angles[1].Body += "x"
	if _, err := ValidateSnapshot(context.Background(), c, snapshot, reader); !errors.Is(err, ErrLimit) {
		t.Fatal("public envelope above 10 MiB accepted")
	}
}
