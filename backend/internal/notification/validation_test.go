package notification

import (
	"github.com/yyl1212/math_master/backend/internal/correction"
	"testing"
)

func TestNotificationValidation(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	if ValidateQuery(Query{Limit: 20}) != nil {
		t.Fatal("default valid query")
	}
	for _, q := range []Query{{Limit: 51}, {Limit: -1}, {Limit: 20, Cursor: "bad+cursor"}} {
		if ValidateQuery(q) == nil {
			t.Fatal("bad pagination accepted")
		}
	}
	for _, kind := range []string{"assessment", "practice", "learning-event", "enrollment"} {
		if ValidateSource(Source{DedupKey: "result:unique", Type: Checking, Evidence: correction.EvidenceRef{Kind: correction.EvidenceKind(kind), ID: id}, CaseID: id}) != nil {
			t.Fatal("valid source denied", kind)
		}
	}
	if ValidateSource(Source{DedupKey: "x", Type: "free_text", CaseID: id}) == nil {
		t.Fatal("arbitrary notification text type accepted")
	}
}
