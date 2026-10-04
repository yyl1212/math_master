package contentaudit

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentAuditReportGolden(t *testing.T) {
	r := Report{SchemaVersion: 1, Mode: Draft, FixtureOnly: true, CreatedAt: "2026-10-04T00:00:00Z", CodeSHA: strings.Repeat("a", 40), Context: ReportContext{CatalogueVersion: 1, CatalogueSHA: strings.Repeat("b", 64), Route: content.VersionRef{ID: "target", Version: 1}, RouteSHA: strings.Repeat("c", 64)}, Source: ReportSource{SnapshotID: strings.Repeat("d", 64), PolicyVersion: 1, ReportSHA: strings.Repeat("e", 64), Complete: true, UnresolvedCount: 8}, DraftCounts: Counts{Knowledge: 30, Templates: 24, FixedInstances: 450, GeneratedInstances: 384, EffectiveInstances: 834}, Nodes: []NodeReport{}, Conclusion: DraftReady, Reasons: []Reason{}}
	var j, m bytes.Buffer
	if e := WriteReport(&j, &m, r); e != nil {
		t.Fatal(e)
	}
	for ext, data := range map[string][]byte{"json": j.Bytes(), "md": m.Bytes()} {
		path := filepath.Join("testdata", "report."+ext)
		if os.Getenv("P6A_UPDATE_GOLDEN") == "1" {
			if e := os.WriteFile(path, data, 0600); e != nil {
				t.Fatal(e)
			}
		}
		want, e := os.ReadFile(path)
		if e != nil || !bytes.Equal(data, want) {
			t.Fatalf("golden %s mismatch: %v", ext, e)
		}
	}
	var decoded Report
	if json.Unmarshal(j.Bytes(), &decoded) != nil || decoded.DraftCounts.EffectiveInstances != 834 || decoded.FormalCounts.EffectiveInstances != 0 || !strings.Contains(m.String(), "| 草稿技术 | 30 | 24 | 450 | 384 | 834 | 0 | 0 |") {
		t.Fatal("count parity")
	}
}
