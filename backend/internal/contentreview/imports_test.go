package contentreview

import (
	"bytes"
	"context"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestReviewImportSourceCoalescing(t *testing.T) {
	in := reviewFixture(t)
	a := in.Sources.Mapping.Sources[0]
	b := a
	b.ID = "additional-source"
	b.RecordID = in.Sources.Report.SelectedFiles[0].RecordIDs[204]
	b.LegacyIDs = []string{"记录B"}
	b.ConditionsNote = "记录B: n is nonnegative"
	in.Sources.Mapping.Sources[0].LegacyIDs = []string{"记录A"}
	in.Sources.Mapping.Sources[0].ConditionsNote = "记录A: zero is allowed"
	in.Sources.Mapping.Sources = append(in.Sources.Mapping.Sources, b)
	for i := range in.Sources.Mapping.Objects {
		o := &in.Sources.Mapping.Objects[i]
		for _, s := range o.SourceIDs {
			if s == a.ID {
				o.SourceIDs = append(o.SourceIDs, b.ID)
				break
			}
		}
	}
	fixtureSyncMap(t, &in)
	scope, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	imports, e := BuildImports(context.Background(), in, scope)
	if e != nil {
		t.Fatal(e)
	}
	keys := map[string]bool{}
	found := false
	for _, l := range imports.Knowledge.SourceMap {
		key := linkKey(l)
		if keys[key] {
			t.Fatal("duplicate source link")
		}
		keys[key] = true
		if strings.Contains(l.Note, "记录A") {
			found = true
			if !strings.Contains(l.Note, "记录B") || !strings.Contains(l.Note, "zero is allowed") || !strings.Contains(l.Note, "nonnegative") {
				t.Fatal("共同出处信息丢失")
			}
		}
	}
	if !found {
		t.Fatal("coalesced source missing")
	}
	ref := content.VersionRef{ID: "natural-numbers", Version: 1}
	l := publication.SourceLink{Knowledge: ref, BatchSHA256: strings.Repeat("a", 64), RelativePath: "safe.json", SHA256: strings.Repeat("b", 64)}
	l2 := l
	l2.SHA256 = strings.Repeat("c", 64)
	if linkKey(l) == linkKey(l2) {
		t.Fatal("different source SHA merged")
	}
	l2 = l
	l2.Knowledge.Version++
	if linkKey(l) == linkKey(l2) {
		t.Fatal("different version merged")
	}
}
func TestReviewImportNoWorkflowIdentity(t *testing.T) {
	in := reviewFixture(t)
	scope, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	out, e := BuildImports(context.Background(), in, scope)
	if e != nil {
		t.Fatal(e)
	}
	raw := fixtureJSON(t, out.Knowledge)
	t.Logf("knowledge import bytes=%d source links=%d", len(raw), len(out.Knowledge.SourceMap))
	var knowledge publication.DraftInput
	if e = question.DecodeStrictJSON(bytes.NewReader(raw), question.MaxEnvelopeBytes, &knowledge); e != nil {
		t.Fatal(e)
	}
	for _, q := range out.Questions {
		t.Logf("question %s import bytes=%d", q.QuestionPackage.ID, len(fixtureJSON(t, q)))
		raw = append(raw, fixtureJSON(t, q)...)
		if _, e = question.DecodeDraft(bytes.NewReader(fixtureJSON(t, q))); e != nil {
			t.Fatal(e)
		}
	}
	for _, key := range []string{"authorIds", "reviewer", "approval", "qualified", "head", "submissionId", "decisionId", "frozenDigest"} {
		if bytes.Contains(raw, []byte(`"`+key+`"`)) {
			t.Fatal("workflow identity leaked", key)
		}
	}
	if len(out.Questions) != 5 || len(out.Knowledge.AssetBytes) != 9 || len(out.Knowledge.SourceMap) != 30 {
		t.Fatal("incomplete imports")
	}
	for _, q := range out.Questions {
		used := map[content.VersionRef]bool{}
		for _, b := range q.QuestionPackage.Blueprints {
			used[b.Knowledge] = true
		}
		for _, l := range q.SourceMap {
			if !used[l.Knowledge] {
				t.Fatal("unreferenced knowledge source")
			}
		}
	}
}
func TestReviewImportBudgets(t *testing.T) {
	base := reviewFixture(t)
	scope, e := BuildScope(context.Background(), base)
	if e != nil {
		t.Fatal(e)
	}
	tests := []struct {
		name   string
		mutate func(*PrepareInput)
	}{
		{"bad-svg", func(in *PrepareInput) {
			for id := range in.Selected.Assets {
				in.Selected.Assets[id] = []byte("<script/>")
				break
			}
		}},
		{"missing-asset", func(in *PrepareInput) {
			for id := range in.Selected.Assets {
				delete(in.Selected.Assets, id)
				break
			}
		}},
		{"external-source", func(in *PrepareInput) { in.Sources.Mapping.Sources[0].Path = "../escape.json"; fixtureSyncMap(t, in) }},
		{"missing-use", func(in *PrepareInput) { in.Sources.Mapping.Sources[0].Use = ""; fixtureSyncMap(t, in) }},
		{"map-limit", func(in *PrepareInput) {
			in.SourceMapRaw = append(in.SourceMapRaw, bytes.Repeat([]byte(" "), contentaudit.MaxSourceMapBytes)...)
		}},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			in := cloneFixture(t, base)
			c.mutate(&in)
			if _, e := BuildImports(context.Background(), in, scope); e == nil {
				t.Fatal("invalid import accepted")
			}
		})
	}
	bad := scope
	bad.Objects = bad.Objects[:len(bad.Objects)-1]
	if _, e := BuildImports(context.Background(), base, bad); e == nil {
		t.Fatal("incomplete scope accepted")
	}
}
