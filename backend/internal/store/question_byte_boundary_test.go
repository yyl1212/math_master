package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestQuestionPackageExactByteBoundary(t *testing.T) {
	f := newQuestionFixture(t)
	p := &f.questionInput.QuestionPackage
	for ti := range p.Templates {
		for pi := range p.Templates[ti].Parameters {
			for vi, value := range p.Templates[ti].Parameters[pi].Values {
				r, e := question.ParseNumeric(value, "rational")
				if e != nil {
					t.Fatal(e)
				}
				p.Templates[ti].Parameters[pi].Values[vi] = r.Numerator + "/" + r.Denominator
			}
		}
	}
	p.Blueprints[0].CoverageNote = ""
	raw, _ := json.Marshal(p)
	p.Blueprints[0].CoverageNote = strings.Repeat("x", question.MaxPackageBytes-len(raw))
	raw, _ = json.Marshal(p)
	if len(raw) != question.MaxPackageBytes {
		t.Fatal("wrong test boundary", len(raw))
	}
	wrapped, _, _ := question.CanonicalPackage(*p)
	t.Log("payload bytes", len(raw), "wrapped bytes", len(wrapped))
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), f.questionInput)
	if err != nil {
		t.Fatal("exact-size create", err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil || !gate.ReadyToSubmit {
		t.Fatal("exact-size validation", err, gate.StructuralErrors, gate.CompletenessErrors)
	}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest})
	if err != nil {
		t.Fatal("exact-size submit", err)
	}
	archive, err := f.repo.ExportQuestionArchive(f.ctx, p.ID, p.Version)
	if err != nil {
		t.Fatal("exact-size export", err)
	}
	_, expected, _ := question.CanonicalPackage(sub.Frozen.QuestionPackage)
	if archive.PackageSHA != expected {
		t.Fatal("exact-size SHA changed")
	}
	// A new equal-length package identity forces an actual import INSERT, not a duplicate replay.
	archive.Envelope.QuestionPackage.ID = "z" + p.ID[1:]
	_, archive.PackageSHA, err = question.CanonicalPackage(archive.Envelope.QuestionPackage)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.ImportQuestionDraft(f.ctx, archive); err != nil {
		t.Fatal("exact-size import", err)
	}
	reexport, err := f.repo.ExportQuestionArchive(f.ctx, archive.Envelope.QuestionPackage.ID, p.Version)
	if err != nil || reexport.PackageSHA != archive.PackageSHA {
		t.Fatal("import/export SHA", err)
	}
	p.Blueprints[0].CoverageNote += "x"
	if _, err = f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), f.questionInput); !errors.Is(err, question.ErrLimitExceeded) {
		t.Fatal("over-size accepted", err)
	}
	archive.Envelope.QuestionPackage = *p
	if _, err = f.repo.ImportQuestionDraft(f.ctx, archive); !errors.Is(err, question.ErrLimitExceeded) {
		t.Fatal("over-size import accepted", err)
	}
}
func TestQuestionSourceMapExactByteBoundary(t *testing.T) {
	f := newQuestionFixture(t)
	f.questionInput.SourceMap[0].Note = ""
	raw, _ := json.Marshal(f.questionInput.SourceMap)
	f.questionInput.SourceMap[0].Note = strings.Repeat("x", question.MaxSourceMapBytes-len(raw))
	raw, _ = json.Marshal(f.questionInput.SourceMap)
	if len(raw) != question.MaxSourceMapBytes {
		t.Fatal("wrong source test boundary", len(raw))
	}
	var sqlBytes int
	if err := f.db.QueryRow(`SELECT octet_length($1::jsonb::text)`, string(raw)).Scan(&sqlBytes); err != nil {
		t.Fatal(err)
	}
	t.Log("canonical source bytes", len(raw), "jsonb text bytes", sqlBytes)
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), f.questionInput)
	if err != nil {
		t.Fatal("legal sourceMap create", err)
	}
	saved, err := f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SaveDraftInput{DraftInput: f.questionInput, ExpectedRevision: 1})
	if err != nil || saved.Revision != 2 {
		t.Fatal("legal sourceMap save", err)
	}
	f.questionInput.SourceMap[0].Note += "x"
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SaveDraftInput{DraftInput: f.questionInput, ExpectedRevision: 2}); !errors.Is(err, question.ErrLimitExceeded) {
		t.Fatal("over-size sourceMap accepted", err)
	}
}

func TestQuestionNormalizedPackageByteBudget(t *testing.T) {
	f := newQuestionFixture(t)
	p := &f.questionInput.QuestionPackage
	p.Blueprints[0].CoverageNote = ""
	raw, _ := json.Marshal(p)
	p.Blueprints[0].CoverageNote = strings.Repeat("x", question.MaxPackageBytes-len(raw))
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), f.questionInput)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReadyToSubmit {
		t.Fatal("parameter normalization grew beyond2MiB but was marked ready")
	}
	found := false
	for _, issue := range gate.StructuralErrors {
		if issue.Code == "QUESTION_LIMIT_EXCEEDED" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing actual budget error", gate.StructuralErrors)
	}
}
