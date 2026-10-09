package knowledgeadmin

import (
	"bytes"
	"strings"
	"testing"
)

func TestSourceCoreAndCurrentDigest(t *testing.T) {
	d, e := DecodeSource(bytes.NewReader(sourceBytes(t)))
	if e != nil {
		t.Fatal(e)
	}
	p := d.KnowledgePoints[0]
	a, e := SourceCoreSHA(p)
	if e != nil {
		t.Fatal(e)
	}
	p.Version++
	b, _ := SourceCoreSHA(p)
	if a != b {
		t.Fatal("source version changes identity")
	}
	p.Statement += " 修正后的说明。"
	b, _ = SourceCoreSHA(p)
	if a == b {
		t.Fatal("body ignored")
	}
	i := CurrentInput{ExternalID: p.ID, Point: p, Sources: []PublicSource{}}
	a, _ = CurrentSHA(i, []string{"97F40"})
	b, _ = CurrentSHA(i, []string{"97F50"})
	if a == b {
		t.Fatal("topic ignored")
	}
	if id := KnowledgeID("相同原始id"); len(id) != 58 || id != KnowledgeID("相同原始id") || id == KnowledgeID("不同id") || !strings.HasPrefix(id, "k-") {
		t.Fatal(id)
	}
	golden := `{"a":1e+30,"b":0.002,"c":"中文😀","d":1}`
	got, e := CanonicalJSON([]byte(`{"d":1.0,"c":"中文😀","b":2e-3,"a":1e30}`))
	if e != nil || string(got) != golden {
		t.Fatalf("JCS: %s %v", got, e)
	}
}

func TestSourceCoreExcludesBinding(t *testing.T) {
	d, e := DecodeSource(bytes.NewReader(sourceBytes(t)))
	if e != nil {
		t.Fatal(e)
	}
	p := d.KnowledgePoints[0]
	a, _ := SourceCoreSHA(p)
	p.ContentOrigin = "adapted"
	p.OriginalType = "source-specific-definition"
	p.OriginalBinding = &OriginalBinding{RecordID: p.ID, LocalRelativePath: "different-book/source.json"}
	b, _ := SourceCoreSHA(p)
	if a != b {
		t.Fatal("private source binding changes mathematical content digest")
	}
}
