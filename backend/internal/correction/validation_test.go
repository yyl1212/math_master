package correction

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

const testUUID = "11111111-1111-4111-8111-111111111111"

func validCase() CaseInput {
	return CaseInput{Kind: GradingRuleCase, Rule: &RuleScope{RuleVersion: 1, Kind: "all"}}
}
func validPlan() PlanInput {
	return PlanInput{AlgorithmVersion: 1, Mappings: []Mapping{}, Reason: "Correct the independently verified answer."}
}
func TestCorrectionValidation(t *testing.T) {
	for name, in := range map[string]CaseInput{"valid": validCase(), "unknown": {Kind: "unknown"}, "wrong-null": {Kind: GradingRuleCase, Rule: &RuleScope{RuleVersion: 1, Kind: "all"}, Withdrawal: &WithdrawalRef{Space: "content", ID: testUUID}}, "all-with-knowledge": {Kind: GradingRuleCase, Rule: &RuleScope{RuleVersion: 1, Kind: "all", Knowledge: &question.Identity{ID: "addition", Version: 1, SHA256: strings.Repeat("a", 64)}}}, "zero-rule": {Kind: GradingRuleCase, Rule: &RuleScope{Kind: "all"}}, "bad-withdrawal": {Kind: WithdrawalCase, Withdrawal: &WithdrawalRef{Space: "content", ID: "not-a-uuid"}}} {
		t.Run(name, func(t *testing.T) {
			e := ValidateCase(in)
			if (e == nil) != (name == "valid") {
				t.Fatalf("case %s: %v", name, e)
			}
		})
	}
	for name, change := range map[string]func(*PlanInput){"valid": func(*PlanInput) {}, "unknown-algorithm": func(p *PlanInput) { p.AlgorithmVersion = 2 }, "empty": func(p *PlanInput) { p.Reason = "" }, "blank": func(p *PlanInput) { p.Reason = " \n\t" }, "nul": func(p *PlanInput) { p.Reason = "bad\x00text" }, "utf8": func(p *PlanInput) { p.Reason = string([]byte{255}) }, "too-long": func(p *PlanInput) { p.Reason = strings.Repeat("数", 4001) }, "too-many-mappings": func(p *PlanInput) { p.Mappings = make([]Mapping, 51) }, "unsafe-sequence": func(p *PlanInput) { n := int64(9007199254740992); p.ExpectedSequence = &n }, "bad-parent": func(p *PlanInput) { p.Parent = &PlanRef{ID: testUUID, Version: 0} }} {
		t.Run(name, func(t *testing.T) {
			in := validPlan()
			change(&in)
			e := ValidatePlan(in)
			if (e == nil) != (name == "valid") {
				t.Fatalf("plan %s: %v", name, e)
			}
		})
	}
	p := validPlan()
	p.Reason = strings.Repeat("数", 4000)
	if e := ValidatePlan(p); e != nil {
		t.Fatalf("4000 scalar boundary: %v", e)
	}
	for _, body := range []string{`{"kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all","knowledge":null},"cutoff":"2026-10-03T00:00:00Z"}`, `{"kind":"grading_rule","rule":{"ruleVersion":1,"kind":"all","knowledge":null}}`, `{"kind":"withdrawal","kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all","knowledge":null}}`} {
		var in CaseInput
		if json.Unmarshal([]byte(body), &in) == nil {
			t.Fatal("untrusted case shape accepted", body)
		}
	}
	var in CaseInput
	if e := json.Unmarshal([]byte(`{"kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all","knowledge":null}}`), &in); e != nil || ValidateCase(in) != nil {
		t.Fatalf("valid case JSON rejected: %v", e)
	}
	for _, q := range []Query{{Limit: 51}, {Limit: -1}, {Limit: 20, Cursor: strings.Repeat("a", 513)}, {Limit: 20, Cursor: "é"}} {
		if ValidateQuery(q) == nil {
			t.Fatalf("invalid query accepted: %+v", q)
		}
	}
}

func TestCorrectionValidationSurrogates(t *testing.T) {
	for _, reason := range []string{`"\ud800"`, `"\udc00"`, `"\ud800\u0041"`} {
		var in PlanInput
		raw := []byte(`{"expectedSequence":null,"parent":null,"algorithmVersion":1,"mappings":[],"reason":` + reason + `}`)
		if json.Unmarshal(raw, &in) == nil {
			t.Fatalf("isolated UTF16 surrogate accepted: %s", reason)
		}
	}
	for _, reason := range []string{`"\ud83d\ude00"`, `"literal \\ud800"`} {
		var in PlanInput
		raw := []byte(`{"expectedSequence":null,"parent":null,"algorithmVersion":1,"mappings":[],"reason":` + reason + `}`)
		if e := json.Unmarshal(raw, &in); e != nil || ValidatePlan(in) != nil {
			t.Fatalf("Unicode scalar/literal escape rejected: %s %v", reason, e)
		}
	}
}
