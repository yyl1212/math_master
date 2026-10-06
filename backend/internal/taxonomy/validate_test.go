package taxonomy

import (
	"fmt"
	"testing"
)

func fixtureNodes() []TopicNode {
	var out []TopicNode
	ptr := func(s string) *string { return &s }
	for i := 0; i < 63; i++ {
		c := fmt.Sprintf("%02d-XX", i)
		out = append(out, TopicNode{ID: TopicID(c), Code: c, Name: "Fixture topic", Kind: "primary", Level: 1})
	}
	for i := 0; i < 534; i++ {
		c := fmt.Sprintf("%02d%cxx", i%63, 'A'+i/63)
		out = append(out, TopicNode{ID: TopicID(c), Code: c, Name: "Fixture subtopic", Kind: "primary", Level: 2, ParentID: ptr(TopicID(fmt.Sprintf("%02d-XX", i%63)))})
	}
	for i := 0; i < 4969; i++ {
		p := i % 534
		prefix := fmt.Sprintf("%02d%c", p%63, 'A'+p/63)
		c := fmt.Sprintf("%s%02d", prefix, i/534)
		out = append(out, TopicNode{ID: TopicID(c), Code: c, Name: "Fixture specific", Kind: "primary", Level: 3, ParentID: ptr(TopicID(prefix + "xx"))})
	}
	for i := 0; i < 503; i++ {
		c := fmt.Sprintf("%02d-%02d", i%63, i/63)
		out = append(out, TopicNode{ID: TopicID(c), Code: c, Name: "Fixture auxiliary", Kind: "auxiliary", Level: 2, ParentID: ptr(TopicID(fmt.Sprintf("%02d-XX", i%63)))})
	}
	for i := 0; i < 534; i++ {
		prefix := fmt.Sprintf("%02d%c", i%63, 'A'+i/63)
		c := prefix + "99"
		out = append(out, TopicNode{ID: TopicID(c), Code: c, Name: "Fixture other", Kind: "other", Level: 3, ParentID: ptr(TopicID(prefix + "xx"))})
	}
	return out
}
func TestTaxonomyCountsAndParents(t *testing.T) {
	n := fixtureNodes()
	if err := ValidateCatalogue(n); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCatalogue(n[:len(n)-1]); err == nil {
		t.Fatal("partial catalogue accepted")
	}
}
func TestTaxonomyDuplicateCode(t *testing.T) {
	n := fixtureNodes()
	n[1] = n[0]
	if ValidateCatalogue(n) == nil {
		t.Fatal("duplicate accepted")
	}
}
func TestTaxonomyMissingParent(t *testing.T) {
	n := fixtureNodes()
	p := "msc-missing"
	n[63].ParentID = &p
	if ValidateCatalogue(n) == nil {
		t.Fatal("missing parent accepted")
	}
}
func TestTaxonomyCycle(t *testing.T) {
	n := fixtureNodes()
	p := n[63].ID
	n[0].ParentID = &p
	if ValidateCatalogue(n) == nil {
		t.Fatal("cycle accepted")
	}
}
func TestTaxonomyWrongLevel(t *testing.T) {
	n := fixtureNodes()
	n[63].Level = 3
	if ValidateCatalogue(n) == nil {
		t.Fatal("wrong hierarchy accepted")
	}
}
func TestTaxonomyIDKeepsOfficialCode(t *testing.T) {
	if got := TopicID("13C60"); got != "msc-13c60" {
		t.Fatal(got)
	}
	if got := TopicID("13Cxx"); got != "msc-13c" {
		t.Fatal(got)
	}
	if got := TopicID("13-XX"); got != "msc-13" {
		t.Fatal(got)
	}
}
