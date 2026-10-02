package question

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"testing"
)

func TestQuestionCanonicalIdentity(t *testing.T) {
	p := QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "contract-fixture", Version: 1, Templates: []Template{}, FixedQuestions: []FixedQuestion{}, Blueprints: []Blueprint{}}
	b, sha, e := CanonicalPackage(p)
	if e != nil {
		t.Fatal(e)
	}
	want := `{"purpose":"question-package-v1","body":` + contractJSON + `}`
	if string(b) != want || sha != "67a0f28c3f9b225341c40caba289789df64efefcee4523aefadc029306081498" {
		t.Fatalf("golden bytes or SHA mismatch: %s %s", b, sha)
	}
	b2, s2, _ := CanonicalPackage(p)
	if string(b2) != want || s2 != sha {
		t.Fatal("non deterministic")
	}
	id := "qi-" + strings.Repeat("a", 64)
	if len(id) != 67 || !ValidInstanceID(id) || publication.ValidMathID(id) || ValidInstanceID(id+"0") || ValidInstanceID(strings.ToUpper(id)) {
		t.Fatal("instance namespace leaks old ID rules")
	}
	i := Instance{Identity: Identity{ID: id, Version: 1, SHA256: strings.Repeat("b", 64)}, Origin: "template", Parameters: []ParameterValue{}, Body: QuestionBody{Type: "single_choice", Knowledge: Ref{ID: "fractions", Version: 1}, Coverage: []ObjectiveCoverage{{Knowledge: Ref{ID: "fractions", Version: 1}, ObjectiveIndices: []int{0, 1}}}, Units: []Ref{}, Choices: []Choice{{ID: "a", Text: "1"}, {ID: "b", Text: "2"}}, Assets: []AssetRef{}, Sources: nil}}
	_, h1, e := CanonicalInstance(i)
	if e != nil {
		t.Fatal(e)
	}
	i.Identity.SHA256 = strings.Repeat("c", 64)
	_, h2, _ := CanonicalInstance(i)
	if h1 != h2 {
		t.Fatal("self SHA entered hash")
	}
	i.Body.Choices[0], i.Body.Choices[1] = i.Body.Choices[1], i.Body.Choices[0]
	_, h3, _ := CanonicalInstance(i)
	if h1 == h3 {
		t.Fatal("choice semantic order erased")
	}
	i.Body.Coverage[0].ObjectiveIndices = []int{1, 0}
	_, h4, _ := CanonicalInstance(i)
	if h3 == h4 {
		t.Fatal("objective semantic order erased")
	}
	i.Body.Units = []Ref{{ID: "unit-b", Version: 1}, {ID: "unit-a", Version: 1}}
	_, h5, _ := CanonicalInstance(i)
	i.Body.Units[0], i.Body.Units[1] = i.Body.Units[1], i.Body.Units[0]
	_, h6, _ := CanonicalInstance(i)
	if h5 != h6 {
		t.Fatal("relation ordering unstable")
	}
	frozen := FrozenBody{AuthorIDs: []string{}, InstanceIdentities: []Identity{i.Identity}, FrozenDigest: ""}
	_, f1, _ := CanonicalFrozen(frozen, []Instance{i})
	i.Body.Prompt = "Changed complete body"
	_, f2, _ := CanonicalFrozen(frozen, []Instance{i})
	if f1 == f2 {
		t.Fatal("frozen hash covers only IDs")
	}
	var doc map[string]any
	if json.Unmarshal(b, &doc) != nil {
		t.Fatal("invalid canonical bytes")
	}
}
