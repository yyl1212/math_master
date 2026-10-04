package contentreview

import (
	"bytes"
	"context"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

// Catches omitted generated combinations, source records and inline proofs in a ready export.
func TestReviewScopeFullBaseline(t *testing.T) {
	in := reviewFixture(t)
	s, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Objects) != 958 || len(s.MappedObjects) != 574 || len(s.DerivedInstances) != 384 || len(s.Sources) != 205 {
		t.Fatalf("incomplete scope: objects=%d mapped=%d derived=%d sources=%d", len(s.Objects), len(s.MappedObjects), len(s.DerivedInstances), len(s.Sources))
	}
	counts := map[string]int{}
	proofs := 0
	for _, o := range s.Objects {
		counts[o.Kind]++
		for _, c := range s.RequiredChecks[ObjectKey(o)] {
			if c == "proof" {
				proofs++
			}
		}
	}
	want := map[string]int{"knowledge": 30, "unit": 30, "path": 1, "asset": 9, "template": 24, "instance": 834, "blueprint": 30}
	for k, v := range want {
		if counts[k] != v {
			t.Fatalf("%s=%d want=%d", k, counts[k], v)
		}
	}
	if proofs != 5 {
		t.Fatalf("proofs=%d want=5", proofs)
	}
	used := 0
	for _, s := range s.Sources {
		if len(s.MappingIDs) > 0 {
			used++
		}
	}
	if used != 51 {
		t.Fatalf("used records=%d want=51", used)
	}
	for _, d := range s.DerivedInstances {
		if d.Template.ID == "" || len(d.SourceIDs) == 0 {
			t.Fatal("generated instance has no exact template/provenance")
		}
	}
}
func TestReviewScopeIdentityMismatch(t *testing.T) {
	base := reviewFixture(t)
	cases := []struct {
		name   string
		mutate func(*PrepareInput)
	}{
		{"sha", func(in *PrepareInput) {
			in.Sources.Mapping.Objects[0].SHA256 = strings.Repeat("0", 64)
			fixtureSyncMap(t, in)
		}},
		{"version", func(in *PrepareInput) { v := 999; in.Sources.Mapping.Objects[0].Version = &v; fixtureSyncMap(t, in) }},
		{"duplicate-object", func(in *PrepareInput) {
			in.Sources.Mapping.Objects = append(in.Sources.Mapping.Objects, in.Sources.Mapping.Objects[0])
			fixtureSyncMap(t, in)
		}},
		{"missing-fixed", func(in *PrepareInput) {
			for i, o := range in.Sources.Mapping.Objects {
				if o.Kind == "instance" {
					in.Sources.Mapping.Objects = append(in.Sources.Mapping.Objects[:i], in.Sources.Mapping.Objects[i+1:]...)
					break
				}
			}
			fixtureSyncMap(t, in)
		}},
		{"missing-blueprint", func(in *PrepareInput) {
			for i, o := range in.Sources.Mapping.Objects {
				if o.Kind == "blueprint" {
					in.Sources.Mapping.Objects = append(in.Sources.Mapping.Objects[:i], in.Sources.Mapping.Objects[i+1:]...)
					break
				}
			}
			fixtureSyncMap(t, in)
		}},
		{"missing-asset", func(in *PrepareInput) {
			for i, o := range in.Sources.Mapping.Objects {
				if o.Kind == "asset" {
					in.Sources.Mapping.Objects = append(in.Sources.Mapping.Objects[:i], in.Sources.Mapping.Objects[i+1:]...)
					break
				}
			}
			fixtureSyncMap(t, in)
		}},
		{"unknown-source", func(in *PrepareInput) {
			in.Sources.Mapping.Objects[0].SourceIDs = []string{"not-a-source"}
			fixtureSyncMap(t, in)
		}},
		{"duplicate-record", func(in *PrepareInput) {
			r := &in.Sources.Report
			r.SelectedFiles[0].RecordIDs = append(r.SelectedFiles[0].RecordIDs, r.SelectedFiles[0].RecordIDs[0])
			in.SourceReportRaw = fixtureJSON(t, *r)
			in.Sources.ReportSHA = sha(in.SourceReportRaw)
			in.Sources.Mapping.SourceReportSHA256 = in.Sources.ReportSHA
			fixtureSyncMap(t, in)
		}},
		{"wrong-route", func(in *PrepareInput) { in.Route.Version = 2 }},
		{"typed-input-change", func(in *PrepareInput) { in.Selected.Input.Content.Knowledge[0].ID = "unrelated-point" }},
		{"manifest-file-change", func(in *PrepareInput) {
			in.Manifest.QuestionPaths[0] = "content/questions/unselected.v2.json"
			in.ManifestRaw = fixtureJSON(t, in.Manifest)
		}},
		{"raw-map-change", func(in *PrepareInput) {
			in.SourceMapRaw = bytes.Replace(in.SourceMapRaw, []byte(`"origin":"original"`), []byte(`"origin":"changed"`), 1)
		}},
		{"map-limit", func(in *PrepareInput) {
			in.SourceMapRaw = append(in.SourceMapRaw, bytes.Repeat([]byte(" "), 256<<10)...)
		}},
		{"fixture-upgrade", func(in *PrepareInput) { in.FixtureOnly = false }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := cloneFixture(t, base)
			c.mutate(&in)
			if _, e := BuildScope(context.Background(), in); e == nil {
				t.Fatal("incomplete or conflicting input accepted")
			}
		})
	}
}
func TestReviewScopeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := BuildScope(ctx, reviewFixture(t)); e == nil {
		t.Fatal("canceled preparation accepted")
	}
}

func TestReviewScopeRevisedSelection(t *testing.T) {
	in := reviewFixture(t)
	old := in.Selected.Input.Questions[0].Templates[0]
	_, oldSHA, e := question.CanonicalTemplate(old)
	if e != nil {
		t.Fatal(e)
	}
	revised := &in.Selected.Input.Questions[0]
	revised.Version = 2
	revised.Templates[0].ExplanationTemplate += " Check the result using the inverse operation."
	_, newSHA, e := question.CanonicalTemplate(revised.Templates[0])
	if e != nil {
		t.Fatal(e)
	}
	in.Manifest.QuestionPaths[0] = "content/questions/elementary-foundations-numbers.v2.json"
	in.Selected.Files[2].Path = in.Manifest.QuestionPaths[0]
	in.Selected.Files[2].Bytes = fixtureJSON(t, *revised)
	in.ManifestRaw = fixtureJSON(t, in.Manifest)
	for i, o := range in.Sources.Mapping.Objects {
		if o.Kind == "template" && o.ID == old.ID {
			in.Sources.Mapping.Objects[i].SHA256 = newSHA
		}
	}
	fixtureSyncMap(t, &in)
	s, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, d := range s.DerivedInstances {
		if d.Template.ID == old.ID {
			found = true
			if d.Template.SHA256 == oldSHA || d.Template.SHA256 != newSHA {
				t.Fatal("reused old template SHA")
			}
		}
	}
	if !found {
		t.Fatal("revised derived instances omitted")
	}
}

func TestReviewScopeMissingParameterTail(t *testing.T) {
	in := reviewFixture(t)
	template := &in.Selected.Input.Questions[0].Templates[0]
	values := template.Parameters[0].Values
	template.Parameters[0].Values = values[:len(values)-1]
	_, hash, e := question.CanonicalTemplate(*template)
	if e != nil {
		t.Fatal(e)
	}
	in.Selected.Files[2].Bytes = fixtureJSON(t, in.Selected.Input.Questions[0])
	for i, o := range in.Sources.Mapping.Objects {
		if o.Kind == "template" && o.ID == template.ID {
			in.Sources.Mapping.Objects[i].SHA256 = hash
		}
	}
	fixtureSyncMap(t, &in)
	if _, e = BuildScope(context.Background(), in); e == nil {
		t.Fatal("missing parameter tail accepted")
	}
}
