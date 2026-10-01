package content

import (
	"io"
	"os"
	"strings"
	"testing"
)

func draftJSON(t *testing.T) string {
	t.Helper()
	b, e := os.ReadFile("testdata/valid-draft.json")
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func TestDecodePreservesChineseAndRejectsPreviewState(t *testing.T) {
	valid := draftJSON(t)
	p, err := DecodePackage(strings.NewReader(valid))
	if err != nil || len(p.Knowledge) != 1 || p.Knowledge[0].TitleZh != "分数" {
		t.Fatal("valid Unicode draft was not preserved")
	}
	for _, field := range []string{"unlocked", "learningState", "contentStatus", "reviewedBy"} {
		changed := strings.Replace(valid, `"type": "definition"`, `"`+field+`": true, "type": "definition"`, 1)
		if _, e := DecodePackage(strings.NewReader(changed)); e == nil {
			t.Errorf("preview/audit field %s accepted", field)
		}
	}
}
func TestDecodeRejectsDuplicateKeysAndUnknownFields(t *testing.T) {
	valid := draftJSON(t)
	for _, changed := range []string{
		strings.Replace(valid, `"version": 1`, `"version": 1, "version": 2`, 1),
		strings.Replace(valid, `"title": "Fractions"`, `"title": "Fractions", "title": "Changed"`, 1),
		strings.Replace(valid, `"assets": []`, `"unknown": true, "assets": []`, 1),
		strings.Replace(valid, `"type": "definition"`, `"type": "arbitrary"`, 1),
	} {
		if _, err := DecodePackage(strings.NewReader(changed)); err == nil {
			t.Fatal("invalid structure accepted")
		}
	}
}
func TestDecodeLimitsBytesAndVersions(t *testing.T) {
	valid := draftJSON(t)
	for _, changed := range []string{
		strings.Replace(valid, `"version": 1`, `"version": -1`, 1),
		strings.Replace(valid, `"schemaVersion": 1`, `"schemaVersion": 2`, 1),
		valid + ` {}`, strings.Repeat(" ", 10*1024*1024) + valid,
	} {
		if _, err := DecodePackage(strings.NewReader(changed)); err == nil {
			t.Fatal("invalid size, version or trailing object accepted")
		}
	}
	if _, err := DecodePackage(io.LimitReader(strings.NewReader(valid), 20)); err == nil {
		t.Fatal("truncated JSON accepted")
	}
}

func TestDecodeFormalSeed(t *testing.T) {
	f, e := os.Open("../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c, e := DecodeCatalogue(f)
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for _, d := range c.Domains {
		n += len(d.Topics)
	}
	if len(c.Domains) != 16 || n != 56 {
		t.Fatalf("catalogue %d/%d", len(c.Domains), n)
	}
	f2, e := os.Open("../../../content/packages/elementary-fractions.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f2.Close()
	p, e := DecodePackage(f2)
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Knowledge) != 10 || len(p.Units) != 1 || len(p.Assets) != 1 {
		t.Fatal("seed shape changed")
	}
}
