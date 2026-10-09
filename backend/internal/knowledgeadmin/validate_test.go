package knowledgeadmin

import (
	"bytes"
	"testing"
)

func TestClassificationIndexAndOtherKind(t *testing.T) {
	ix, e := LoadClassificationIndex()
	if e != nil || len(ix.Specific) != 4969 || len(ix.Other) != 534 {
		t.Fatal("bad classification index", e)
	}
	d, e := DecodeSource(bytes.NewReader(sourceBytes(t)))
	if e != nil {
		t.Fatal(e)
	}
	p := d.KnowledgePoints[0]
	var other string
	for k := range ix.Other {
		other = k
		break
	}
	p.MSCCodes = []string{other}
	p.ClassificationEvidence = []ClassificationEvidence{{MSCCode: other, Kind: "other", Reason: "此知识已经核实适合本板块的其他主题。", EvidenceFields: []string{"statement"}}}
	if e = ValidatePoint(p, ix); e != nil {
		t.Fatal(e)
	}
	p.ClassificationEvidence[0].Kind = "specific"
	if ValidatePoint(p, ix) == nil {
		t.Fatal("other counted as specific")
	}
	for _, s := range []string{"../secret", "a/../secret", "/root", "C:/file", "a\\b", "a//b"} {
		if SafeRelativePath(s) {
			t.Fatal("unsafe relative path", s)
		}
	}
}
