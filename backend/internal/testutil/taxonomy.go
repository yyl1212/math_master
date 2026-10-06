package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"strings"
)

func TaxonomyBatch() taxonomy.CapturedBatch {
	var n []taxonomy.TopicNode
	ptr := func(s string) *string { return &s }
	for i := 0; i < 63; i++ {
		c := fmt.Sprintf("%02d-XX", i)
		n = append(n, taxonomy.TopicNode{ID: taxonomy.TopicID(c), Code: c, Name: "Original fixture theme", Kind: "primary", Level: 1})
	}
	for i := 0; i < 534; i++ {
		c := fmt.Sprintf("%02d%cxx", i%63, 'A'+i/63)
		n = append(n, taxonomy.TopicNode{ID: taxonomy.TopicID(c), Code: c, Name: "Original fixture subtheme", Kind: "primary", Level: 2, ParentID: ptr(taxonomy.TopicID(fmt.Sprintf("%02d-XX", i%63)))})
	}
	for i := 0; i < 4969; i++ {
		p := i % 534
		prefix := fmt.Sprintf("%02d%c", p%63, 'A'+p/63)
		c := fmt.Sprintf("%s%02d", prefix, i/534)
		n = append(n, taxonomy.TopicNode{ID: taxonomy.TopicID(c), Code: c, Name: "Original fixture specific", Kind: "primary", Level: 3, ParentID: ptr(taxonomy.TopicID(prefix + "xx"))})
	}
	for i := 0; i < 503; i++ {
		c := fmt.Sprintf("%02d-%02d", i%63, i/63)
		n = append(n, taxonomy.TopicNode{ID: taxonomy.TopicID(c), Code: c, Name: "Original auxiliary", Kind: "auxiliary", Level: 2, ParentID: ptr(taxonomy.TopicID(fmt.Sprintf("%02d-XX", i%63)))})
	}
	for i := 0; i < 534; i++ {
		p := fmt.Sprintf("%02d%c", i%63, 'A'+i/63)
		n = append(n, taxonomy.TopicNode{ID: taxonomy.TopicID(p + "99"), Code: p + "99", Name: "Original other", Kind: "other", Level: 3, ParentID: ptr(taxonomy.TopicID(p + "xx"))})
	}
	files := []taxonomy.SourceFile{{Path: "Materials/fixture.json", SizeBytes: 2, SHA256: strings.Repeat("a", 64)}}
	b, _ := json.Marshal(files)
	h := sha256.Sum256(b)
	return taxonomy.CapturedBatch{Manifest: taxonomy.TopicCaptureManifest{SchemaVersion: 1, Batch: 1, SourceFiles: files, SnapshotID: hex.EncodeToString(h[:]), Accepted: true, Diff: taxonomy.CaptureDiff{Added: []string{}, Changed: []string{}, Missing: []string{}}, Issues: []taxonomy.CaptureIssue{}}, Nodes: n, SourceRecordIndex: []taxonomy.SourceRecordRef{}, RawClassificationSHA: strings.Repeat("b", 64), Attribution: "Original technical fixture", License: "Original fixture"}
}
