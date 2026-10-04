package contentreview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var learningNames = []string{"reading", "pass", "fail", "prerequisites", "review", "practice_exposure", "retake", "feedback_correction"}

func withLearning(t *testing.T, f *evidenceFixtureData) {
	t.Helper()
	for _, name := range learningNames {
		a := evidenceSave(t, f.Root, "learning/"+name+"-observation.json", map[string]string{"observation": "Isolated technical fixture observation, no real learner."})
		record := LearningRecord{SchemaVersion: 1, Name: name, Result: "passed", Context: f.Release.ReleaseIdentity, ExecutedAt: "2026-10-04T00:00:00Z", Steps: []LearningStep{{Action: "Execute the exact technical scenario", Expected: "Observe the specified acceptance or refusal", Observed: "The expected behavior was observed in this fixture"}}, Attachments: []FileRef{a}, AttestedBy: fixtureReviewer, Attestation: f.Release.Attestation}
		f.Input.LearningChecks = append(f.Input.LearningChecks, LearningFile{name, evidenceSave(t, f.Root, "learning/"+name+".json", record)})
	}
}
func mutateLearning(t *testing.T, f *evidenceFixtureData, index int, mutate func(*LearningRecord)) {
	t.Helper()
	ref := f.Input.LearningChecks[index].File
	var record LearningRecord
	evidenceRead(t, *f, ref, &record)
	mutate(&record)
	f.Input.LearningChecks[index].File = evidenceSave(t, f.Root, ref.Path, record)
}
func TestReviewEvidenceContext(t *testing.T) {
	f := evidenceFixture(t)
	withLearning(t, &f)
	v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
	if e != nil || v.Conclusion != "evidence_ready" || v.Evidence == nil {
		t.Fatal("complete files not ready", e, v.Conclusion)
	}
	var legacy contentaudit.AcceptanceEvidence
	raw := bundleVerificationFile(t, v, "acceptance-evidence.json")
	if e = contentaudit.DecodeEvidence(bytes.NewReader(raw), &legacy); e != nil {
		t.Fatal(e)
	}
	if legacy.CodeSHA != f.Release.CodeSHA || legacy.FixtureOnly != f.Release.FixtureOnly || !legacy.FixtureOnly {
		t.Fatal("绑定丢失")
	}
	cases := []struct {
		name   string
		mutate func(*ReleaseIdentity)
	}{
		{"code", func(c *ReleaseIdentity) { c.CodeSHA = strings.Repeat("b", 40) }},
		{"knowledge-id", func(c *ReleaseIdentity) { c.KnowledgeHead.ID = "99999999-9999-4999-8999-999999999999" }},
		{"knowledge-sha", func(c *ReleaseIdentity) { c.KnowledgeHead.SHA256 = strings.Repeat("9", 64) }},
		{"question-id", func(c *ReleaseIdentity) { c.QuestionHead.ID = "99999999-9999-4999-8999-999999999999" }},
		{"question-sha", func(c *ReleaseIdentity) { c.QuestionHead.SHA256 = strings.Repeat("9", 64) }},
		{"route-id", func(c *ReleaseIdentity) { c.Route.ID = "old-route" }},
		{"route-version", func(c *ReleaseIdentity) { c.Route.Version++ }},
		{"route-sha", func(c *ReleaseIdentity) { c.Route.SHA256 = strings.Repeat("9", 64) }},
		{"catalogue-version", func(c *ReleaseIdentity) { c.CatalogueVersion++ }},
		{"catalogue-sha", func(c *ReleaseIdentity) { c.CatalogueSHA256 = strings.Repeat("9", 64) }},
		{"manifest-sha", func(c *ReleaseIdentity) { c.ManifestSHA256 = strings.Repeat("9", 64) }},
		{"fixture-upgrade", func(c *ReleaseIdentity) { c.FixtureOnly = false }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := evidenceFixture(t)
			withLearning(t, &f)
			mutateLearning(t, &f, 0, func(r *LearningRecord) { c.mutate(&r.Context) })
			if _, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input); e == nil {
				t.Fatal("mixed final contexts accepted")
			}
		})
	}
}
func TestReviewEvidenceFiles(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *evidenceFixtureData)
	}{
		{"changed-attachment", func(t *testing.T, f *evidenceFixtureData) {
			fixtureWrite(t, filepath.Join(f.Root, "learning/reading-observation.json"), []byte("changed"))
		}},
		{"missing-attachment", func(t *testing.T, f *evidenceFixtureData) {
			if e := os.Remove(filepath.Join(f.Root, "learning/reading-observation.json")); e != nil {
				t.Fatal(e)
			}
		}},
		{"wrong-file-sha", func(t *testing.T, f *evidenceFixtureData) {
			f.Input.LearningChecks[0].File.SHA256 = strings.Repeat("0", 64)
		}},
		{"duplicate-key", func(t *testing.T, f *evidenceFixtureData) {
			ref := f.Input.LearningChecks[0].File
			b, e := os.ReadFile(filepath.Join(f.Root, ref.Path))
			if e != nil {
				t.Fatal(e)
			}
			b = append([]byte(`{"result":"passed",`), b[1:]...)
			fixtureWrite(t, filepath.Join(f.Root, ref.Path), b)
			f.Input.LearningChecks[0].File.SHA256 = sha(b)
		}},
		{"unknown-name", func(t *testing.T, f *evidenceFixtureData) { f.Input.LearningChecks[0].Name = "unknown" }},
		{"duplicate-check", func(t *testing.T, f *evidenceFixtureData) {
			f.Input.LearningChecks = append(f.Input.LearningChecks, f.Input.LearningChecks[0])
		}},
		{"wrong-record-name", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.Name = "pass" })
		}},
		{"invalid-result", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.Result = "approved" })
		}},
		{"escape", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.Attachments[0].Path = "../secret" })
		}},
		{"attachment-limit", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) {
				for i := 0; i < 65; i++ {
					r.Attachments = append(r.Attachments, r.Attachments[0])
				}
			})
		}},
		{"missing-step", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.Steps = []LearningStep{} })
		}},
		{"missing-observed", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.Steps[0].Observed = "" })
		}},
		{"bad-time", func(t *testing.T, f *evidenceFixtureData) {
			mutateLearning(t, f, 0, func(r *LearningRecord) { r.ExecutedAt = "not-time" })
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := evidenceFixture(t)
			withLearning(t, &f)
			c.mutate(t, &f)
			v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
			if e == nil && c.name != "missing-step" && c.name != "missing-observed" {
				t.Fatal("invalid contract or file not rejected")
			}
			if e == nil && v.Conclusion == "evidence_ready" {
				t.Fatal("invalid evidence file accepted")
			}
			if v.Evidence != nil {
				t.Fatal("invalid ready file produced")
			}
		})
	}
}
func TestReviewEvidenceNoReadyFile(t *testing.T) {
	for _, name := range []string{"missing", "not-run", "unsigned", "release-unsigned", "build-missing"} {
		t.Run(name, func(t *testing.T) {
			f := evidenceFixture(t)
			withLearning(t, &f)
			switch name {
			case "missing":
				f.Input.LearningChecks = f.Input.LearningChecks[1:]
			case "not-run":
				mutateLearning(t, &f, 0, func(r *LearningRecord) { r.Result = "not_run" })
			case "unsigned":
				mutateLearning(t, &f, 0, func(r *LearningRecord) { r.Attestation = FileRef{} })
			case "release-unsigned":
				f.Release.Attestation = FileRef{}
				f.Input.ReleaseContext = evidenceSave(t, f.Root, "release-context.json", f.Release)
			case "build-missing":
				f.Release.BuildRecord = FileRef{}
				f.Input.ReleaseContext = evidenceSave(t, f.Root, "release-context.json", f.Release)
			}
			v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
			if e != nil {
				t.Fatal(e)
			}
			if v.Conclusion != "awaiting_review" || v.Evidence != nil {
				t.Fatal("unfinished evidence accepted")
			}
			out := filepath.Join(outputRoot(t), "verification")
			if e = WriteVerification(context.Background(), out, v); e != nil {
				t.Fatal(e)
			}
			if _, e = os.Stat(filepath.Join(out, "acceptance-evidence.json")); !os.IsNotExist(e) {
				t.Fatal("pending ready file created")
			}
		})
	}
	f := evidenceFixture(t)
	withLearning(t, &f)
	mutateLearning(t, &f, 1, func(r *LearningRecord) {
		r.Result = "failed"
		r.Steps[0].Observed = "The fixture observed an incorrect behavior"
	})
	v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
	if e != nil || v.Conclusion != "not_ready" || v.Evidence == nil {
		t.Fatal("actual failure not preserved", e)
	}
	if v.Evidence.LearningChecks[1].Result != "failed" {
		t.Fatal("failed changed to passed")
	}
}
func TestReviewEvidenceReadBudgets(t *testing.T) {
	root := outputRoot(t)
	block := bytes.Repeat([]byte("a"), MaxFileBytes)
	fixtureWrite(t, filepath.Join(root, "block"), block)
	reader, e := newEvidenceReader(context.Background(), root)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 32; i++ {
		p := fmt.Sprintf("block-%02d", i)
		if e = os.Link(filepath.Join(root, "block"), filepath.Join(root, p)); e != nil {
			t.Fatal(e)
		}
		if _, e = reader.read(FileRef{p, sha(block)}, MaxFileBytes); e != nil {
			t.Fatal("legal total refused", i, e)
		}
	}
	fixtureWrite(t, filepath.Join(root, "extra"), []byte("a"))
	if _, e = reader.read(FileRef{"extra", sha([]byte("a"))}, MaxFileBytes); !errors.Is(e, ErrLimit) {
		t.Fatal("total+1 accepted", e)
	}
	fixtureWrite(t, filepath.Join(root, "oversized"), append(block, 'a'))
	reader, e = newEvidenceReader(context.Background(), root)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = reader.read(FileRef{"oversized", sha(append(block, 'a'))}, MaxFileBytes); e == nil {
		t.Fatal("file+1 accepted")
	}
}
func bundleVerificationFile(t *testing.T, v Verification, path string) []byte {
	t.Helper()
	for _, f := range v.Files {
		if f.Path == path {
			return f.Bytes
		}
	}
	t.Fatal("missing verification file", path)
	return nil
}

func TestReviewEvidenceAttachmentBoundary(t *testing.T) {
	f := evidenceFixture(t)
	withLearning(t, &f)
	for index := range f.Input.LearningChecks {
		mutateLearning(t, &f, index, func(r *LearningRecord) {
			r.Attachments = []FileRef{}
			for i := 0; i < 8; i++ {
				r.Attachments = append(r.Attachments, evidenceSave(t, f.Root, fmt.Sprintf("attachments/%s-%02d.json", r.Name, i), map[string]string{"observation": "technical fixture only"}))
			}
		})
	}
	v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
	if e != nil || v.Conclusion != "evidence_ready" {
		t.Fatal("64 actual attachments refused", e)
	}
	mutateLearning(t, &f, 7, func(r *LearningRecord) {
		r.Attachments = append(r.Attachments, evidenceSave(t, f.Root, "attachments/extra.json", map[string]string{"observation": "technical fixture only"}))
	})
	if _, e = VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input); !errors.Is(e, ErrLimit) {
		t.Fatal("attachment count+1 accepted", e)
	}
}
func TestReviewEvidenceAbsentReleaseHeads(t *testing.T) {
	f := evidenceFixture(t)
	f.Release.KnowledgeHead = contentaudit.Head{}
	f.Release.QuestionHead = contentaudit.Head{}
	f.Input.ReleaseContext = evidenceSave(t, f.Root, "release-context.json", f.Release)
	v, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
	if e != nil || v.Conclusion != "awaiting_review" || v.Evidence != nil {
		t.Fatal("missing real published heads accepted", e)
	}
}
