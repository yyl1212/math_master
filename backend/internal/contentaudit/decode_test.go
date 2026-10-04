package contentaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentAuditMetadataBoundaries(t *testing.T) {
	base := SourceMap{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: strings.Repeat("a", 64), SourceReportSHA256: strings.Repeat("b", 64), Sources: []MappedSource{}, Objects: []SourceObject{}}
	raw, _ := json.Marshal(base)
	padded := append(raw, bytes.Repeat([]byte(" "), MaxSourceMapBytes-len(raw))...)
	var out SourceMap
	if e := DecodeSourceMap(bytes.NewReader(padded), &out); e != nil {
		t.Fatal(e)
	}
	if e := DecodeSourceMap(bytes.NewReader(append(padded, ' ')), &out); e == nil {
		t.Fatal("size+1 accepted")
	}
	for _, raw := range []string{`{"schemaVersion":1,"schemaVersion":1}`, `{"schemaVersion":2147483648}`, `{"unknown":true}`, string([]byte{0xff}), `{"schemaVersion":1,"policyVersion":1,"snapshotId":"\u0000","sourceReportSha256":"","sources":[],"objects":[]}`} {
		if DecodeSourceMap(strings.NewReader(raw), &out) == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}
func sourceFixture(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	os.Mkdir(filepath.Join(root, "files"), 0700)
	corpus := []byte(`{"schema_version":"1","dataset_id":"set","knowledge_points":[{"id":"r1","conditions":["n>=0"]}]}`)
	os.WriteFile(filepath.Join(root, "files", "main.json"), corpus, 0600)
	files := []map[string]any{{"path": "main.json", "sizeBytes": len(corpus), "sha256": hashBytes(corpus)}}
	raw, _ := json.Marshal(files)
	snapshot := hashBytes(raw)
	manifest := map[string]any{"schemaVersion": 1, "snapshotId": snapshot, "files": files, "createdAt": "2026-10-04T00:00:00Z", "sourceRoot": "/unused", "packageCount": 1, "primaryFiles": []string{"main.json"}, "changes": map[string]any{"added": []string{"main.json"}, "modified": []string{}, "missing": []string{}}, "sourceIndexMismatches": []string{}}
	m, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(root, "manifest.json"), m, 0600)
	report := SourceReport{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshot, SelectedFiles: []SelectedSource{{Path: "main.json", SHA256: hashBytes(corpus), DatasetID: "set", RecordIDs: []string{"r1"}}}, Issues: []SourceIssue{}, Ready: true}
	r, _ := json.Marshal(report)
	rp := filepath.Join(root, "report.json")
	os.WriteFile(rp, r, 0600)
	mapping := SourceMap{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshot, SourceReportSHA256: hashBytes(r), Sources: []MappedSource{{ID: "s1", Path: "main.json", RecordID: "r1", DatasetID: "set", FileSHA256: hashBytes(corpus), PublicSources: []content.Source{}, Use: "fact_check", LegacyIDs: []string{}, ReviewStatus: "AI_checked", ConditionsNote: "n>=0"}}, Objects: []SourceObject{}}
	mb, _ := json.Marshal(mapping)
	mp := filepath.Join(root, "map.json")
	os.WriteFile(mp, mb, 0600)
	return root, rp, mp
}
func TestContentAuditSourceIdentity(t *testing.T) {
	root, rp, mp := sourceFixture(t)
	s, e := LoadSources(context.Background(), root, rp, mp)
	if e != nil || s.Report.PublicationApproved {
		t.Fatal(s, e)
	}
	for _, mutate := range []func(*SourceMap){func(m *SourceMap) { m.SnapshotID = strings.Repeat("f", 64) }, func(m *SourceMap) { m.SourceReportSHA256 = strings.Repeat("f", 64) }, func(m *SourceMap) { m.Sources[0].RecordID = "missing" }, func(m *SourceMap) { m.Sources[0].FileSHA256 = strings.Repeat("f", 64) }} {
		var m SourceMap
		b, _ := os.ReadFile(mp)
		json.Unmarshal(b, &m)
		mutate(&m)
		b, _ = json.Marshal(m)
		bad := filepath.Join(root, "bad.json")
		os.WriteFile(bad, b, 0600)
		if _, e := LoadSources(context.Background(), root, rp, bad); e == nil {
			t.Fatal("untrusted identity accepted")
		}
	}
	f, s := auditFixture(30)
	s.Mapping.Objects[0].SHA256 = strings.Repeat("0", 64)
	if _, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s); e == nil {
		t.Fatal("wrong object digest accepted")
	}
}
