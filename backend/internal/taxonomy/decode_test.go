package taxonomy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestTaxonomyDecodeStrictBoundaries(t *testing.T) {
	in := AssignmentInput{}
	_ = in
	for _, raw := range []string{`{"manifest":{},"manifest":{}}`, `{"manifest":{},"extra":"unknown"}`, `{"manifest":{"schemaVersion":1},"nodes":"bad"}`, `{"manifest":{"snapshotId":"\ud800"}}`} {
		if _, e := DecodeCapturedBatch(strings.NewReader(raw)); e == nil {
			t.Fatal("invalid archive accepted")
		}
	}
}
func TestTaxonomyDecodeUsesBoundedReader(t *testing.T) {
	if _, e := DecodeCapturedBatch(bytes.NewReader(bytes.Repeat([]byte("x"), (32<<20)+1))); e == nil {
		t.Fatal("unbounded archive")
	}
	var b CapturedBatch
	raw, _ := json.Marshal(b)
	if _, e := DecodeCapturedBatch(bytes.NewReader(raw)); e == nil {
		t.Fatal("empty archive")
	}
}
func TestTaxonomyDecodeValidArchive(t *testing.T) {
	files := []SourceFile{}
	rawFiles, _ := json.Marshal(files)
	h := sha256.Sum256(rawFiles)
	c := CapturedBatch{Manifest: TopicCaptureManifest{SchemaVersion: 1, Batch: 1, SnapshotID: hex.EncodeToString(h[:]), Accepted: true, SourceFiles: files, Diff: CaptureDiff{Added: []string{}, Changed: []string{}, Missing: []string{}}, Issues: []CaptureIssue{}}, Nodes: fixtureNodes(), SourceRecordIndex: []SourceRecordRef{}, RawClassificationSHA: strings.Repeat("b", 64), Attribution: "Original fixture", License: "Original fixture"}
	raw, _ := json.Marshal(c)
	got, e := DecodeCapturedBatch(bytes.NewReader(raw))
	if e != nil || len(got.Nodes) != 6603 || !got.Manifest.Accepted {
		t.Fatal(e)
	}
}
