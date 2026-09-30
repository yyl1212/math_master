package content

import (
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"os"
	"testing"
)

func seed(t *testing.T) (catalogue.Catalogue, Package, string) {
	t.Helper()
	f, e := os.Open("../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c, e := DecodeCatalogue(f)
	if e != nil {
		t.Fatal(e)
	}
	f2, e := os.Open("../../../content/packages/elementary-fractions.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f2.Close()
	p, e := DecodePackage(f2)
	if e != nil {
		t.Fatal(e)
	}
	return c, p, "../../../content/assets"
}
func blocked(t *testing.T, c catalogue.Catalogue, p Package, r string) {
	t.Helper()
	if len(ValidateStructure(c, p, r).Errors) == 0 {
		t.Fatal("invalid package accepted")
	}
}
func TestReferencesRequireExactPackageVersion(t *testing.T) {
	for _, kind := range []string{"wrong", "missing", "closure"} {
		t.Run(kind, func(t *testing.T) {
			c, p, r := seed(t)
			switch kind {
			case "wrong":
				p.Knowledge[2].Relations[0].Target.Version = 2
			case "missing":
				p.Knowledge[2].Relations[0].Target.ID = "unknown"
			case "closure":
				p.Paths[0].Nodes = p.Paths[0].Nodes[1:]
			}
			blocked(t, c, p, r)
		})
	}
}
func TestPrerequisiteDAGIgnoresRelatedCycles(t *testing.T) {
	c, p, r := seed(t)
	p.Knowledge[0].Relations = append(p.Knowledge[0].Relations, Relation{Kind: "related", Target: VersionRef{ID: p.Knowledge[1].ID, Version: 1}})
	p.Knowledge[1].Relations = append(p.Knowledge[1].Relations, Relation{Kind: "related", Target: VersionRef{ID: p.Knowledge[0].ID, Version: 1}})
	if v := ValidateStructure(c, p, r); len(v.Errors) > 0 {
		t.Fatal(v.Errors)
	}
	p.Knowledge[0].Relations = append(p.Knowledge[0].Relations, Relation{Kind: "prerequisite", Target: VersionRef{ID: p.Knowledge[2].ID, Version: 1}})
	blocked(t, c, p, r)
}
func TestIDsAndDomainMembership(t *testing.T) {
	for _, kind := range []string{"duplicate", "topic", "membership", "invalid"} {
		t.Run(kind, func(t *testing.T) {
			c, p, r := seed(t)
			switch kind {
			case "duplicate":
				p.Knowledge = append(p.Knowledge, p.Knowledge[0])
			case "topic":
				p.Knowledge[0].TopicIDs = []string{"unknown"}
			case "membership":
				p.Knowledge[0].DomainIDs = []string{c.Domains[0].ID}
			case "invalid":
				p.Knowledge[0].ID = "Old_ID"
			}
			blocked(t, c, p, r)
		})
	}
}
func TestFormalSeedHasReviewItems(t *testing.T) {
	c, p, r := seed(t)
	report := ValidateStructure(c, p, r)
	if len(report.Errors) > 0 || len(report.ReviewItems) == 0 {
		t.Fatal(report)
	}
}
