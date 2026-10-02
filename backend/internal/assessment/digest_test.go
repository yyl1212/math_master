package assessment

import (
	"crypto/sha256"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func testSeal() Seal {
	return Seal{Kind: "assessment", Knowledge: testIdentity("fractions"), Mode: func() *Mode { x := Mode("diagnostic"); return &x }(), Blueprint: func() *question.Identity { x := testIdentity("test-blueprint"); return &x }(), KnowledgePublicationID: "snapshot", QuestionPublicationID: "11111111-1111-4111-8111-111111111111", RuleVersion: 1, Core: []int{0}, Seed: strings.Repeat("a", 64), Items: []ItemBinding{{Position: 1, Instance: testIdentity("item-1"), Approval: question.MemberEvidence{}}}}
}
func TestLearningSealPurposeAndCompleteBudget(t *testing.T) {
	s := testSeal()
	raw, sha, e := CanonicalSeal(s)
	if e != nil || sha != fmt.Sprintf("%x", sha256.Sum256(raw)) || !strings.HasPrefix(string(raw), `{"purpose":"assessment-attempt-v1","body":`) {
		t.Fatal(string(raw), sha, e)
	}
	if strings.Contains(string(raw), `:null`) && strings.Contains(string(raw), `"assets":null`) {
		t.Fatal("nil arrays", string(raw))
	}
	other := s
	other.Kind = "practice"
	_, p, e := CanonicalSeal(other)
	if e != nil || p == sha {
		t.Fatal("purpose collision", e)
	}
	padding := ""
	s.Items[0].Approval.InheritedFrom = &padding
	empty, _, _ := CanonicalSeal(s)
	padding = strings.Repeat("x", 4194304-len(empty))
	raw, _, e = CanonicalSeal(s)
	if e != nil || len(raw) != 4194304 {
		t.Fatal("exact complete envelope", len(raw), e)
	}
	padding += "x"
	if _, _, e = CanonicalSeal(s); e == nil {
		t.Fatal("complete envelope overflow accepted")
	}
	s.Kind = "unknown"
	if _, _, e = CanonicalSeal(s); e == nil {
		t.Fatal("unknown purpose")
	}
}
