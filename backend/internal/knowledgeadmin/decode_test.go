package knowledgeadmin

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func sourceBytes(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("testdata/valid-source.json")
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func sourceMap(t *testing.T) map[string]any {
	t.Helper()
	var d map[string]any
	if e := json.Unmarshal(sourceBytes(t), &d); e != nil {
		t.Fatal(e)
	}
	return d
}
func mutatedSource(t *testing.T, mutate func(map[string]any, map[string]any)) []byte {
	t.Helper()
	d := sourceMap(t)
	p := d["knowledge_points"].([]any)[0].(map[string]any)
	mutate(d, p)
	b, e := json.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestDecodeSourceStrictParity(t *testing.T) {
	doc, e := DecodeSource(bytes.NewReader(sourceBytes(t)))
	if e != nil || len(doc.KnowledgePoints) != 2 {
		t.Fatalf("valid example: %v", e)
	}
	cases := []struct {
		name   string
		mutate func(map[string]any, map[string]any)
	}{
		{"unsupported-type", func(d, p map[string]any) { p["type"] = "lemma" }},
		{"null-type", func(d, p map[string]any) { p["type"] = nil }},
		{"pending-type", func(d, p map[string]any) { p["type_status"] = "pending" }},
		{"level-six", func(d, p map[string]any) { p["learning_difficulty"].(map[string]any)["difficulty_level"] = 6 }},
		{"level-zero", func(d, p map[string]any) { p["learning_difficulty"].(map[string]any)["difficulty_level"] = 0 }},
		{"null-level", func(d, p map[string]any) { p["learning_difficulty"].(map[string]any)["difficulty_level"] = nil }},
		{"pending-level", func(d, p map[string]any) { p["learning_difficulty"].(map[string]any)["review_status"] = "pending" }},
		{"no-prerequisites", func(d, p map[string]any) {
			v := p["learning_difficulty"].(map[string]any)
			v["difficulty_level"] = 2
			v["prerequisites"] = []any{}
		}},
		{"full-proof-empty", func(d, p map[string]any) { p["proof_scope"] = "full" }},
		{"sketch-proof-empty", func(d, p map[string]any) { p["proof_scope"] = "sketch" }},
		{"theorem-inapplicable", func(d, p map[string]any) { p["type"] = "theorem" }},
		{"parent-topic", func(d, p map[string]any) { p["msc_codes"] = []any{"97F"} }},
		{"unknown-specific", func(d, p map[string]any) {
			p["msc_codes"] = []any{"97Z99"}
			p["classification_evidence"].([]any)[0].(map[string]any)["msc_code"] = "97Z99"
		}},
		{"evidence-mismatch", func(d, p map[string]any) { p["msc_codes"] = []any{"97F50"} }},
		{"empty-evidence-field", func(d, p map[string]any) { p["conditions"] = []any{} }},
		{"duplicate-evidence", func(d, p map[string]any) {
			v := p["classification_evidence"].([]any)
			p["classification_evidence"] = append(v, v[0])
		}},
		{"unexpected-learning-state", func(d, p map[string]any) { p["learning_state"] = "completed" }},
		{"unsafe-source-url", func(d, p map[string]any) { d["source"].(map[string]any)["url"] = "https://user:password@example.com/" }},
		{"nul-text", func(d, p map[string]any) { p["statement"] = "valid\x00text" }},
		{"condition-shape", func(d, p map[string]any) { p["conditions"] = map[string]any{"a": "b"} }},
		{"id-byte-limit", func(d, p map[string]any) { d["source"].(map[string]any)["source_id"] = strings.Repeat("中", 60) }},
		{"null-extension", func(d, p map[string]any) { p["extensions"] = map[string]any{"metadata": nil} }},
		{"placeholder-core", func(d, p map[string]any) { p["scope"] = "待核实" }},
		{"proof-scope", func(d, p map[string]any) { p["proof_scope"] = "unknown" }},
		{"empty-title", func(d, p map[string]any) { p["title_zh"] = " " }},
		{"other-without-reason", func(d, p map[string]any) { p["type"] = "other" }},
		{"empty-classification", func(d, p map[string]any) { p["msc_codes"] = []any{}; p["classification_evidence"] = []any{} }},
		{"dangerous-markup", func(d, p map[string]any) { p["statement"] = "<script>alert(1)</script>" }},
		{"leading-id-space", func(d, p map[string]any) { p["id"] = " leading" }},
		{"trailing-id-space", func(d, p map[string]any) { p["id"] = "trailing " }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, e := DecodeSource(bytes.NewReader(mutatedSource(t, c.mutate)))
			if e == nil {
				t.Fatal("accepted invalid input")
			}
			de, ok := e.(*DecodeError)
			if !ok || de.Path == "" {
				t.Fatalf("missing safe location: %v", e)
			}
		})
	}
	for _, raw := range [][]byte{[]byte(`{"format":1,"format":2}`), []byte("{\"a\":\"\xff\"}"), []byte(`{"a":"\ud800"}`), append(sourceBytes(t), []byte(` {}`)...)} {
		if _, e := DecodeSource(bytes.NewReader(raw)); e == nil {
			t.Fatal("accepted malformed JSON")
		}
	}
}
func TestDecodeSourceExplicitOtherAndRawIDs(t *testing.T) {
	raw := mutatedSource(t, func(d, p map[string]any) {
		p["type"] = "other"
		p["type_other_reason"] = "这是明确归入其他主类型的格式测试内容。"
		p["id"] = "unknown"
		for _, q := range d["knowledge_points"].([]any) {
			for _, r := range q.(map[string]any)["relations"].([]any) {
				rr := r.(map[string]any)
				if rr["target_id"] == "demo-rational-fraction" {
					rr["target_id"] = "unknown"
				}
			}
		}
	})
	if _, e := DecodeSource(bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	raw = mutatedSource(t, func(d, p map[string]any) {
		p["classification_mode"] = "project_other"
		p["msc_codes"] = []any{}
		p["classification_evidence"] = []any{}
		p["project_other"] = map[string]any{"group_id": "project-other", "reason": "暂不属于官方具体分类，但已经核实内容并明确归入项目其他组。", "evidence_fields": []any{"statement"}}
	})
	if _, e := DecodeSource(bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
}

func TestDecodeSourceRetainsDeclaredRelationsAcrossSplitFiles(t *testing.T) {
	var d map[string]any
	if e := json.Unmarshal(sourceBytes(t), &d); e != nil {
		t.Fatal(e)
	}
	points := d["knowledge_points"].([]any)
	p := points[0].(map[string]any)
	target := points[1].(map[string]any)
	p["relations"] = []any{map[string]any{"kind": "related", "target_source_id": d["source"].(map[string]any)["source_id"], "target_id": target["id"], "target_version": target["version"], "status": "confirmed", "reason": "已核实的关系，目标知识位于同一规范快照的另一文件。"}}
	d["knowledge_points"] = points[:1]
	b, e := json.Marshal(d)
	if e != nil {
		t.Fatal(e)
	}
	v, e := DecodeSource(bytes.NewReader(b))
	if e != nil || len(v.KnowledgePoints[0].Relations) != 1 {
		t.Fatal("a valid split file rejected an external declaration", e)
	}
}
