package contentreview

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
)

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func fixtureWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func fixtureJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

// Uses original application mathematics and a synthetic provenance corpus, never book prose.
func reviewFixture(t *testing.T) PrepareInput {
	t.Helper()
	root, e := filepath.Abs("../../..")
	if e != nil {
		t.Fatal(e)
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "content/review/elementary-foundations.input.v1.json"))
	if e != nil {
		t.Fatal(e)
	}
	var manifest InputManifest
	if e = json.Unmarshal(raw, &manifest); e != nil {
		t.Fatal(e)
	}
	selected, e := contentaudit.LoadSelectedDraft(context.Background(), root, manifest.Selection())
	if e != nil {
		t.Fatal("selected", e)
	}
	mapping, e := contentaudit.ReadSourceMap(filepath.Join(root, "content/source-maps/elementary-foundations.v1.json"))
	if e != nil {
		t.Fatal("map", e)
	}
	ids := map[string]bool{}
	for _, s := range mapping.Sources {
		ids[s.RecordID] = true
	}
	for i := 1; len(ids) < 205; i++ {
		ids[fmt.Sprintf("review-unused-%03d", i)] = true
	}
	recordIDs := []string{}
	for id := range ids {
		recordIDs = append(recordIDs, id)
	}
	sort.Strings(recordIDs)
	records := []map[string]string{}
	for _, id := range recordIDs {
		records = append(records, map[string]string{"id": id})
	}
	corpus := fixtureJSON(t, map[string]any{"dataset_id": mapping.Sources[0].DatasetID, "knowledge_points": records})
	sourcePath := mapping.Sources[0].Path
	files := []map[string]any{{"path": sourcePath, "sizeBytes": len(corpus), "sha256": sha(corpus)}}
	snapshotSHA := sha(fixtureJSON(t, files))
	snapshot, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	fixtureWrite(t, filepath.Join(snapshot, "files", sourcePath), corpus)
	fixtureWrite(t, filepath.Join(snapshot, "manifest.json"), fixtureJSON(t, map[string]any{"schemaVersion": 1, "snapshotId": snapshotSHA, "files": files, "createdAt": nil, "sourceRoot": nil, "packageCount": nil, "primaryFiles": nil, "changes": nil, "sourceIndexMismatches": nil}))
	report := contentaudit.SourceReport{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshotSHA, SelectedFiles: []contentaudit.SelectedSource{{Path: sourcePath, SHA256: sha(corpus), DatasetID: mapping.Sources[0].DatasetID, RecordIDs: recordIDs}}, Issues: []contentaudit.SourceIssue{}, Ready: true}
	reportRaw := fixtureJSON(t, report)
	mapping.SnapshotID = snapshotSHA
	mapping.SourceReportSHA256 = sha(reportRaw)
	for i := range mapping.Sources {
		mapping.Sources[i].FileSHA256 = sha(corpus)
	}
	mapRaw := fixtureJSON(t, mapping)
	sources, e := contentaudit.LoadSourcesFromBytes(context.Background(), snapshot, reportRaw, mapRaw)
	if e != nil {
		t.Fatal("sources", e)
	}
	return PrepareInput{CodeSHA: strings.Repeat("a", 40), Route: content.VersionRef{ID: "elementary-foundations", Version: 1}, Manifest: manifest, ManifestRaw: raw, Selected: selected, Sources: sources, SourceReportRaw: reportRaw, SourceMapRaw: mapRaw, FixtureOnly: true}
}
func cloneFixture(t *testing.T, in PrepareInput) PrepareInput {
	t.Helper()
	var out PrepareInput
	if e := json.Unmarshal(fixtureJSON(t, in), &out); e != nil {
		t.Fatal(e)
	}
	return out
}
func fixtureSyncMap(t *testing.T, in *PrepareInput) {
	t.Helper()
	in.SourceMapRaw = fixtureJSON(t, in.Sources.Mapping)
}
