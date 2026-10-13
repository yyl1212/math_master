package studyreview

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"reflect"
	"strings"
	"testing"
	"time"
)

func source() knowledgeadmin.SourcePoint {
	return knowledgeadmin.SourcePoint{Title: "Title", TitleZH: "标题", Type: "theorem", Statement: "下式成立", Conditions: []string{"x>0", "  "}, Scope: "实数", System: "ZFC", Equations: []string{"x^2>0"}, Proof: "由正数乘积为正。", ProofScope: "本命题"}
}
func TestReferenceTargetsAndLimit(t *testing.T) {
	p := source()
	m, e := BuildReference(p, Recall, Statement)
	if e != nil {
		t.Fatal(e)
	}
	want := []Section{{"statement", []string{"下式成立"}}, {"conditions", []string{"x>0"}}, {"scope", []string{"实数"}}, {"system", []string{"ZFC"}}, {"equations", []string{"x^2>0"}}}
	if len(m.Prompt) != 0 || !reflect.DeepEqual(m.Expected, want) || len(m.SHA256) != 64 {
		t.Fatalf("statement %+v", m)
	}
	proof, e := BuildReference(p, Recall, Proof)
	if e != nil || !reflect.DeepEqual(proof.Prompt, want) || !reflect.DeepEqual(proof.Expected, []Section{{"proof", []string{p.Proof}}, {"proof-scope", []string{p.ProofScope}}}) {
		t.Fatalf("proof %+v %v", proof, e)
	}
	p.Conditions[0] = "changed"
	if m.Expected[1].Items[0] != "x>0" {
		t.Fatal("snapshot aliases source")
	}
	p.Proof = " \n"
	if _, e := BuildReference(p, Recall, Proof); e == nil {
		t.Fatal("empty proof accepted")
	}
	p = source()
	p.Statement = strings.Repeat("a", 524288)
	p.Conditions = nil
	p.Scope = ""
	p.System = ""
	p.Equations = nil
	if _, e := BuildReference(p, Recall, Statement); e != nil {
		t.Fatalf("raw boundary: %v", e)
	}
	p.Statement += "a"
	if _, e := BuildReference(p, Recall, Statement); e != ErrReferenceTooLarge {
		t.Fatalf("raw excess: %v", e)
	}
	p = source()
	p.Title = strings.Repeat("😀", 1024)
	p.TitleZH = strings.Repeat("字", 1024)
	if _, e := BuildReference(p, Read, Statement); e != nil {
		t.Fatal(e)
	}
	p.Title += "a"
	if _, e := BuildReference(p, Read, Statement); e != ErrReferenceTooLarge {
		t.Fatal("overlong metadata")
	}
	for _, mode := range []Mode{Mode("exam"), Read} {
		target := Target("missing")
		if _, e := BuildReference(source(), mode, target); e == nil {
			t.Fatal("invalid mode/target")
		}
	}
}
func TestReferenceWireBudget(t *testing.T) {
	p := source()
	p.Statement = strings.Repeat("<", 100000)
	p.Conditions = nil
	p.Scope = ""
	p.System = ""
	p.Equations = nil
	if _, e := BuildReference(p, Recall, Statement); e != ErrReferenceTooLarge {
		t.Fatalf("HTML escapes exceed wire: %v", e)
	}
	p.Statement = strings.Repeat("<", 98000)
	m, e := BuildReference(p, Recall, Statement)
	if e != nil {
		t.Fatal(e)
	}
	// 最大元数据形状（所有时间/安排/当次评分非null），答案使用合法最坏转义字符。
	s := Session{ActorID: strings.Repeat("a", 36), SessionID: strings.Repeat("b", 36), KnowledgeID: testRef("b").ID, ReviewID: strings.Repeat("c", 36), Knowledge: testRef("b"), Mode: Recall, Target: Statement, Status: Completed, Sequence: MaxSequence, DraftRevision: MaxSequence, Title: strings.Repeat("<", 1024), TitleZH: strings.Repeat("😀", 1024), KnowledgeType: "theorem", CreatedAt: time.Now(), Available: true, Prompt: m.Prompt, Reference: &ExpectedReference{Sections: m.Expected, SHA256: m.SHA256}}
	answer := strings.Repeat("<", 16000)
	s.Answer = &answer
	n := s.CreatedAt
	s.ComparedAt = &n
	s.GradedAt = &n
	s.AbandonedAt = &n
	s.DraftUpdatedAt = &n
	s.Schedule = &Schedule{Stage: 5, DueAt: n, LastRef: testRef("b"), Sequence: MaxSequence, GradedAt: n, LastRating: Familiar}
	s.Grade = &GradeResult{Rating: Familiar, GradedAt: n, Schedule: *s.Schedule}
	if e := ValidateSessionBudget(s); e != nil {
		t.Fatalf("composite: %v", e)
	}
	b, e := json.Marshal(s)
	if e != nil || len(b) > 1048576 {
		t.Fatalf("response %d %v", len(b), e)
	}
	var ref map[string]any
	rb, _ := json.Marshal(s.Reference)
	_ = json.Unmarshal(rb, &ref)
	if _, ok := ref["prompt"]; ok {
		t.Fatal("prompt duplicated inside reference")
	}
	read, e := BuildReference(source(), Read, Statement)
	if e != nil || len(read.Prompt) != 0 || len(read.Expected) != 7 {
		t.Fatalf("read: %+v %v", read, e)
	}
}

func TestReferenceExistingKnowledgeTypes(t *testing.T) {
	for _, kind := range []string{"concept", "definition", "axiom", "theorem", "corollary", "method", "mathematical-thinking", "other"} {
		p := source()
		p.Type = kind
		if _, e := BuildReference(p, Read, Statement); e != nil {
			t.Fatalf("existing %s: %v", kind, e)
		}
	}
	p := source()
	p.Type = "lemma"
	if _, e := BuildReference(p, Read, Statement); e == nil {
		t.Fatal("non-schema type accepted")
	}
}

func TestReferenceCanonicalDigest(t *testing.T) {
	p := source()
	p.Statement = "A"
	p.Conditions = nil
	p.Scope = ""
	p.System = ""
	p.Equations = nil
	m, e := BuildReference(p, Recall, Statement)
	if e != nil || m.SHA256 != "aadc145ad236f35534c682ce51ef092d4e75865c343f3aa4293c4868420c19d7" {
		t.Fatalf("noncanonical reference digest %s %v", m.SHA256, e)
	}
}
