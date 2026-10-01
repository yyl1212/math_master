package publication

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"strings"
	"testing"
)

func withdrawalPackage() content.Package {
	p := releasePackage()
	child := p.Knowledge[0]
	child.ID = "child"
	child.Relations = []content.Relation{{Kind: "prerequisite", Target: content.VersionRef{ID: "root", Version: 1}}}
	related := p.Knowledge[0]
	related.ID = "related"
	related.Relations = []content.Relation{{Kind: "related", Target: content.VersionRef{ID: "root", Version: 1}}}
	p.Knowledge = append(p.Knowledge, child, related)
	for _, id := range []string{"child", "related"} {
		u := p.Units[0]
		u.ID = id + "-unit"
		u.Knowledge.ID = id
		u.AssetIDs = []string{}
		p.Units = append(p.Units, u)
	}
	p.Paths = []content.Path{{ID: "root-path", Version: 1, DomainIDs: p.Knowledge[0].DomainIDs, Title: "Root to child", TitleZh: "路线", Nodes: []content.VersionRef{{ID: "root", Version: 1}, {ID: "child", Version: 1}}}}
	return p
}
func TestWithdrawalClosure(t *testing.T) {
	p := withdrawalPackage()
	base, err := BuildCandidate(Candidate{}, []ReviewedBatch{reviewedBatch(t, p, "44444444-4444-4444-8444-444444444444")})
	if err != nil {
		t.Fatal(err)
	}
	base.PublicationID = "66666666-6666-4666-8666-666666666666"
	before, _ := json.Marshal(base)
	for _, target := range []WithdrawalTarget{{Kind: "knowledge", ID: "root", Version: 1}, {Kind: "unit", ID: "root-unit", Version: 1}, {Kind: "asset", SHA256: strings.Repeat("a", 64)}} {
		out, err := WithdrawCandidate(base, target)
		if err != nil {
			t.Fatal(target, err)
		}
		if out.Diff.Removed != 6 || len(out.Snapshot.Knowledge) != 1 || out.Snapshot.Knowledge[0].ID != "related" || len(out.Snapshot.Knowledge[0].Relations) != 1 || len(out.Snapshot.Assets) != 0 || len(out.Snapshot.Paths) != 0 {
			t.Fatal("incorrect closure or immutable non-prerequisite edge", target, out.Diff)
		}
		for _, m := range out.Manifest.Members {
			if m.Evidence.InheritedFrom == nil || *m.Evidence.InheritedFrom != base.PublicationID {
				t.Fatal("historical evidence lost")
			}
		}
	}
	out, err := WithdrawCandidate(base, WithdrawalTarget{Kind: "path", ID: "root-path", Version: 1})
	if err != nil || out.Diff.Removed != 1 || len(out.Snapshot.Knowledge) != 3 {
		t.Fatal("path withdrawal removed knowledge", err)
	}
	after, _ := json.Marshal(base)
	if string(before) != string(after) {
		t.Fatal("pure withdrawal changed immutable input")
	}
	p.Assets = append(p.Assets, p.Assets[0])
	p.Assets[1].ID = "related-asset"
	p.Assets[1].Knowledge.ID = "related"
	p.Units[2].AssetIDs = []string{"related-asset"}
	shared, err := BuildCandidate(Candidate{}, []ReviewedBatch{reviewedBatch(t, p, "44444444-4444-4444-8444-444444444444")})
	if err != nil {
		t.Fatal(err)
	}
	shared.PublicationID = base.PublicationID
	out, err = WithdrawCandidate(shared, WithdrawalTarget{Kind: "asset", SHA256: strings.Repeat("a", 64)})
	if err != nil || len(out.Manifest.Members) != 0 {
		t.Fatal("exact shared SVG bytes left a knowledge node active", err)
	}
}
