package publication

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"strings"
	"testing"
)

func TestWorkflowDraftDigest(t *testing.T) {
	b := FrozenBody{CatalogueVersion: 1, CatalogueSHA256: strings.Repeat("a", 64), Package: content.Package{SchemaVersion: 1, ID: "draft", Version: 1, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}}, SourceMap: []SourceLink{}, AuthorIDs: []string{"22222222-2222-4222-8222-222222222222", "11111111-1111-4111-8111-111111111111"}, Assets: []content.AssetView{}}
	first, err := FrozenDigest(b)
	if err != nil {
		t.Fatal(err)
	}
	b.AuthorIDs[0], b.AuthorIDs[1] = b.AuthorIDs[1], b.AuthorIDs[0]
	b.FrozenDigest = "not part of hash"
	second, err := FrozenDigest(b)
	if err != nil || first != second {
		t.Fatal("author order or digest field changed frozen digest")
	}
	d := DraftView{ID: "33333333-3333-4333-8333-333333333333", Revision: 1, CatalogueVersion: b.CatalogueVersion, CatalogueSHA256: b.CatalogueSHA256, Package: b.Package, SourceMap: b.SourceMap, AuthorIDs: b.AuthorIDs, Assets: b.Assets}
	one, err := DraftDigest(d)
	if err != nil || one == first {
		t.Fatal("digest purposes not separated")
	}
	d.CreatedAt = "time does not enter content digest"
	d.Gate.Digest = "ignored"
	two, err := DraftDigest(d)
	if err != nil || one != two {
		t.Fatal("mutable gate/time entered digest")
	}
	d.Revision++
	two, err = DraftDigest(d)
	if err != nil || one == two {
		t.Fatal("revision excluded from draft digest")
	}
	d.Package.Knowledge = []content.Knowledge{{Statement: "é"}}
	one, _ = DraftDigest(d)
	d.Package.Knowledge[0].Statement = "é"
	two, _ = DraftDigest(d)
	if one == two {
		t.Fatal("Unicode mathematical text normalized")
	}
	raw, _ := json.Marshal(DraftInput{})
	if strings.Contains(string(raw), "authorIds") {
		t.Fatal("client input carries trusted authors")
	}
}

func TestWorkflowDraftNoteAndEncoding(t *testing.T) {
	for _, v := range []string{"1234567890", "a         ", strings.Repeat("中", 1000)} {
		if !ValidNote(v) {
			t.Fatal("valid codepoint/byte note rejected")
		}
	}
	for _, v := range []string{"123456789", strings.Repeat(" ", 10), "123456789\x00", string([]byte{0xff}) + "1234567890", strings.Repeat("😀", 1000)} {
		if ValidNote(v) {
			t.Fatal("invalid operation note accepted")
		}
	}
}

func TestContentRelativeSourcePathRejectsAbsolutePrefixes(t *testing.T) {
	for _, path := range []string{"/root.json", "C:/root.json", "c:root.json", "\\\\host\\root.json", "a//b.json", "a/../b.json", "a/./b.json"} {
		if RelativeSourcePath(path) {
			t.Fatal("absolute or non-relative source path accepted", path)
		}
	}
	if !RelativeSourcePath("数学基础/root.json") {
		t.Fatal("ordinary Unicode relative source path rejected")
	}
}
