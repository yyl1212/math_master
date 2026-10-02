package feedback

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

const testUUID = "11111111-1111-4111-8111-111111111111"
const otherUUID = "22222222-2222-4222-8222-222222222222"

func testPtr[T any](v T) *T { return &v }
func testIdentity(id string) *question.Identity {
	return &question.Identity{ID: id, Version: 1, SHA256: strings.Repeat("a", 64)}
}
func siteInput() CreateInput {
	return CreateInput{Target: Target{Kind: "site", Area: testPtr(Area("home"))}, Source: Source{Kind: "site"}, Category: "suggestion", Title: "Improve navigation", Message: "Please add a clearer learning entry.", Location: ""}
}
func TestFeedbackModelMetadataHasNoFreeText(t *testing.T) {
	m := Metadata{ID: testUUID, Target: siteInput().Target, Label: "Website: home", Category: "suggestion", Status: "new", Sequence: 1, TargetValidity: "not_applicable"}
	b, e := json.Marshal(Envelope[Metadata]{ActorID: otherUUID, Data: m})
	if e != nil {
		t.Fatal(e)
	}
	var v map[string]json.RawMessage
	if json.Unmarshal(b, &v) != nil || len(v) != 2 {
		t.Fatal(string(b))
	}
	var data map[string]json.RawMessage
	if json.Unmarshal(v["data"], &data) != nil {
		t.Fatal(string(b))
	}
	for _, field := range []string{"title", "message", "location", "reply", "ownerId", "username", "source", "binding", "duplicateOf"} {
		if _, ok := data[field]; ok {
			t.Fatal("unsafe metadata field", field)
		}
	}
	if len(data) != 11 {
		t.Fatal("strict metadata", string(b))
	}
	if string(data["resolutionKind"]) != "null" {
		t.Fatal("required nullable field")
	}
}
