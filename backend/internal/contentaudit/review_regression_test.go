package contentaudit

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

// These are in-memory facts for testing the final gate, never an actual acceptance report.
func reviewedPublishedFacts(t *testing.T) (Request, PublishedFacts, SourceBundle, AcceptanceEvidence) {
	t.Helper()
	f := questionBatch(t, "numbers", "operations", "fractions", "decimals", "ratios")
	m, err := ReadSourceMap("../../../content/source-maps/elementary-foundations.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	s := SourceBundle{SnapshotID: m.SnapshotID, ReportSHA: m.SourceReportSHA256, Mapping: m, Report: SourceReport{SchemaVersion: 1, PolicyVersion: 1, Ready: true, Issues: []SourceIssue{}}}
	req := testRequest(Published)
	req.Route = content.VersionRef{ID: f.Path.ID, Version: f.Path.Version}
	p := PublishedFacts{CatalogueVersion: 1, CatalogueSHA: f.References.CatalogueSHA256, Path: f.Path, PathSHA: content.Digest(f.Path), Content: f.Content, KnowledgeHead: &Head{ID: "synthetic-knowledge", SHA256: strings.Repeat("a", 64)}, QuestionHead: &Head{ID: "synthetic-questions", SHA256: strings.Repeat("b", 64)}}
	for _, sealed := range f.Sealed {
		p.Bank.Templates = append(p.Bank.Templates, sealed.Package.Templates...)
		p.Bank.Blueprints = append(p.Bank.Blueprints, sealed.Package.Blueprints...)
		p.Bank.Instances = append(p.Bank.Instances, sealed.Instances...)
	}
	for _, in := range p.Bank.Instances {
		p.EligibleInstances = append(p.EligibleInstances, in.Identity)
	}
	ev := AcceptanceEvidence{SchemaVersion: 1, CodeSHA: req.CodeSHA, RouteSHA: p.PathSHA, KnowledgeHead: p.KnowledgeHead, QuestionHead: p.QuestionHead}
	for _, o := range m.Objects {
		decision := "synthetic-" + o.Kind + "-" + o.ID
		p.Approvals = append(p.Approvals, ApprovalFact{Object: ObjectIdentity{Kind: o.Kind, ID: o.ID, Version: o.Version, SHA256: o.SHA256}, Evidence: publication.MemberEvidence{DecisionID: decision, FrozenDigest: strings.Repeat("e", 64)}, AuthorIDs: []string{"synthetic-author"}, ReviewerID: "synthetic-reviewer", ChecksComplete: true, FrozenMatches: true})
		ev.ReviewAttestations = append(ev.ReviewAttestations, ReviewAttestation{DecisionID: decision, IndependenceVerified: true})
		if o.Kind == "blueprint" {
			p.Bank.Manifest.Members = append(p.Bank.Manifest.Members, question.ManifestMember{Identity: question.MemberIdentity{Kind: o.Kind, ID: o.ID, Version: *o.Version, SHA256: o.SHA256}})
		}
	}
	for _, name := range []string{"reading", "pass", "fail", "prerequisites", "review", "practice_exposure", "retake", "feedback_correction"} {
		ev.LearningChecks = append(ev.LearningChecks, LearningCheck{Name: name, Result: "passed", EvidenceSHA256: strings.Repeat("f", 64)})
	}
	return req, p, s, ev
}

func TestContentAuditEvidenceConflicts(t *testing.T) {
	req, p, s, ev := reviewedPublishedFacts(t)
	r, e := EvaluatePublished(context.Background(), req, p, s, ev)
	if e != nil || r.Conclusion != Accepted || r.FormalCounts.Knowledge != 30 || r.FormalCounts.Templates != 24 || r.FormalCounts.EffectiveInstances != 834 {
		t.Fatal("complete unique evidence rejected", r.Conclusion, r.FormalCounts, e)
	}
	for _, results := range [][2]string{{"not_run", "passed"}, {"passed", "not_run"}, {"passed", "passed"}, {"failed", "passed"}, {"passed", "failed"}} {
		t.Run("learning-"+results[0]+"-"+results[1], func(t *testing.T) {
			copyEv := ev
			copyEv.LearningChecks = append([]LearningCheck{}, ev.LearningChecks...)
			first := copyEv.LearningChecks[0]
			first.Result = results[0]
			second := first
			second.Result = results[1]
			copyEv.LearningChecks = append([]LearningCheck{first, second}, copyEv.LearningChecks[1:]...)
			if _, e := EvaluatePublished(context.Background(), req, p, s, copyEv); !errors.Is(e, ErrInvalid) {
				t.Fatal("duplicate learning evidence must be rejected", e)
			}
		})
	}
	for _, values := range [][2]bool{{false, true}, {true, false}, {true, true}, {false, false}} {
		name := "attestation-"
		if values[0] {
			name += "true"
		} else {
			name += "false"
		}
		if values[1] {
			name += "-true"
		} else {
			name += "-false"
		}
		t.Run(name, func(t *testing.T) {
			copyEv := ev
			first := ev.ReviewAttestations[0]
			first.IndependenceVerified = values[0]
			second := first
			second.IndependenceVerified = values[1]
			copyEv.ReviewAttestations = append([]ReviewAttestation{first, second}, ev.ReviewAttestations[1:]...)
			if _, e := EvaluatePublished(context.Background(), req, p, s, copyEv); !errors.Is(e, ErrInvalid) {
				t.Fatal("duplicate review attestation must be rejected", e)
			}
		})
	}
	t.Run("unknown-check", func(t *testing.T) {
		copyEv := ev
		copyEv.LearningChecks = append(append([]LearningCheck{}, ev.LearningChecks...), LearningCheck{Name: "unknown", Result: "passed", EvidenceSHA256: strings.Repeat("f", 64)})
		if _, e := EvaluatePublished(context.Background(), req, p, s, copyEv); !errors.Is(e, ErrInvalid) {
			t.Fatal("unknown learning check accepted", e)
		}
	})
	t.Run("failed-check", func(t *testing.T) {
		copyEv := ev
		copyEv.LearningChecks = append([]LearningCheck{}, ev.LearningChecks...)
		copyEv.LearningChecks[0].Result = "failed"
		r, e := EvaluatePublished(context.Background(), req, p, s, copyEv)
		if e != nil || r.Conclusion != NotReady {
			t.Fatal(r.Conclusion, e)
		}
	})
	t.Run("unattested-independence", func(t *testing.T) {
		copyEv := ev
		copyEv.ReviewAttestations = append([]ReviewAttestation{}, ev.ReviewAttestations...)
		copyEv.ReviewAttestations[0].IndependenceVerified = false
		r, e := EvaluatePublished(context.Background(), req, p, s, copyEv)
		if e != nil || r.Conclusion == Accepted {
			t.Fatal(r.Conclusion, e)
		}
	})
}

func TestContentAuditTemplateEligibility(t *testing.T) {
	req, p, s, ev := reviewedPublishedFacts(t)
	dead := map[question.Identity]bool{}
	for _, tpl := range p.Bank.Templates[:5] {
		_, sha, e := question.CanonicalTemplate(tpl)
		if e != nil {
			t.Fatal(e)
		}
		dead[question.Identity{ID: tpl.ID, Version: tpl.Version, SHA256: sha}] = true
	}
	p.EligibleInstances = nil
	for _, in := range p.Bank.Instances {
		if in.Template == nil || !dead[*in.Template] {
			p.EligibleInstances = append(p.EligibleInstances, in.Identity)
		}
	}
	r, e := EvaluatePublished(context.Background(), req, p, s, ev)
	if e != nil || r.Conclusion != NotReady || r.DraftCounts.Templates != 19 || r.DraftCounts.FixedInstances != 450 || r.DraftCounts.EffectiveInstances != 754 {
		t.Fatal("five unusable templates counted despite fixed coverage", r.Conclusion, r.DraftCounts, e)
	}
	for _, n := range r.Nodes {
		if !n.Ready {
			t.Fatal("fixed question coverage should remain ready", n)
		}
	}
}

func TestContentAuditSourceReadBudget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := LoadSources(ctx, t.TempDir(), "missing-report", "missing-map"); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation must precede file IO", e)
	}
}
func TestContentAuditSourceSpecialFiles(t *testing.T) {
	for _, which := range []string{"report", "corpus"} {
		t.Run(which, func(t *testing.T) {
			root, rp, mp := sourceFixture(t)
			pipe := rp
			if which == "corpus" {
				pipe = filepath.Join(root, "files/main.json")
			}
			if e := os.Remove(pipe); e != nil {
				t.Fatal(e)
			}
			if b, e := exec.Command("mkfifo", pipe).CombinedOutput(); e != nil {
				t.Fatalf("create FIFO: %v %s", e, b)
			}
			t.Cleanup(func() {
				f, e := os.OpenFile(pipe, os.O_RDWR, 0600)
				if e == nil {
					f.Close()
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() { _, e := LoadSources(ctx, root, rp, mp); done <- e }()
			select {
			case e := <-done:
				if !errors.Is(e, ErrInvalid) {
					t.Fatal("special file must fail closed", e)
				}
			case <-time.After(250 * time.Millisecond):
				t.Fatal("FIFO escaped source read deadline")
			}
		})
	}
}
func TestContentAuditSourceDeclaredBytes(t *testing.T) {
	root, rp, mp := sourceFixture(t)
	path := filepath.Join(root, "files/main.json")
	raw, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	// Changed bytes are refused, including a same-length replacement and growth beyond the declared snapshot size.
	for _, changed := range [][]byte{append(append([]byte{}, raw...), []byte(" ")...), []byte(strings.ReplaceAll(string(raw), "r1", "r2"))} {
		if e := os.WriteFile(path, changed, 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := LoadSources(context.Background(), root, rp, mp); e == nil {
			t.Fatal("changed snapshot accepted")
		}
	}
	// The corpus has its own budget: valid raw source over the metadata limit must remain usable.
	large := append(raw, []byte(strings.Repeat(" ", MaxMetadataBytes))...)
	if e := os.WriteFile(path, large, 0600); e != nil {
		t.Fatal(e)
	}
	files := []snapshotFile{{Path: "main.json", SizeBytes: int64(len(large)), SHA256: hashBytes(large)}}
	fb, _ := json.Marshal(files)
	sid := hashBytes(fb)
	mb, _ := json.Marshal(snapshotManifest{SchemaVersion: 1, SnapshotID: sid, Files: fb})
	os.WriteFile(filepath.Join(root, "manifest.json"), mb, 0600)
	var report SourceReport
	b, _ := os.ReadFile(rp)
	json.Unmarshal(b, &report)
	report.SnapshotID = sid
	report.SelectedFiles[0].SHA256 = hashBytes(large)
	b, _ = json.Marshal(report)
	os.WriteFile(rp, b, 0600)
	var mapping SourceMap
	b2, _ := os.ReadFile(mp)
	json.Unmarshal(b2, &mapping)
	mapping.SnapshotID = sid
	mapping.SourceReportSHA256 = hashBytes(b)
	mapping.Sources[0].FileSHA256 = hashBytes(large)
	b2, _ = json.Marshal(mapping)
	os.WriteFile(mp, b2, 0600)
	if _, e := LoadSources(context.Background(), root, rp, mp); e != nil {
		t.Fatal("source corpus incorrectly shares metadata limit", e)
	}
}

func TestContentAuditSourceCorpusCapacity(t *testing.T) {
	root, rp, mp := sourceFixture(t)
	size := int64(64<<20) + 1
	path := filepath.Join(root, "files/main.json")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(size); e != nil {
		t.Fatal(e)
	}
	f.Close()
	files := []snapshotFile{{Path: "main.json", SizeBytes: size, SHA256: strings.Repeat("a", 64)}}
	fb, _ := json.Marshal(files)
	sid := hashBytes(fb)
	mb, _ := json.Marshal(snapshotManifest{SchemaVersion: 1, SnapshotID: sid, Files: fb})
	os.WriteFile(filepath.Join(root, "manifest.json"), mb, 0600)
	var report SourceReport
	b, _ := os.ReadFile(rp)
	json.Unmarshal(b, &report)
	report.SnapshotID = sid
	report.SelectedFiles[0].SHA256 = files[0].SHA256
	b, _ = json.Marshal(report)
	os.WriteFile(rp, b, 0600)
	var mapping SourceMap
	b2, _ := os.ReadFile(mp)
	json.Unmarshal(b2, &mapping)
	mapping.SnapshotID = sid
	mapping.SourceReportSHA256 = hashBytes(b)
	mapping.Sources[0].FileSHA256 = files[0].SHA256
	b2, _ = json.Marshal(mapping)
	os.WriteFile(mp, b2, 0600)
	if _, e := LoadSources(context.Background(), root, rp, mp); !errors.Is(e, ErrLimit) {
		t.Fatal("oversized corpus must fail its separate memory budget before hashing", e)
	}
}

func TestContentAuditDraftSpecialAsset(t *testing.T) {
	input, e := LoadDraft(context.Background(), "../../..")
	if e != nil {
		t.Fatal(e)
	}
	root := t.TempDir()
	asset := input.Content.Assets[0]
	path := filepath.Join(root, asset.Path)
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if b, e := exec.Command("mkfifo", path).CombinedOutput(); e != nil {
		t.Fatalf("create asset FIFO: %v %s", e, b)
	}
	input.AssetsRoot = root
	t.Cleanup(func() {
		f, e := os.OpenFile(path, os.O_RDWR, 0600)
		if e == nil {
			f.Close()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := CheckDraft(ctx, input); done <- e }()
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("non-regular asset accepted")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("asset FIFO escaped draft read deadline")
	}
}

func TestContentAuditDraftCancellation(t *testing.T) {
	input, e := LoadDraft(context.Background(), "../../..")
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = CheckDraft(ctx, input); !errors.Is(e, context.Canceled) {
		t.Fatal("draft cancellation must remain a budget error", e)
	}
}
