package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"strings"
	"testing"
)

func questionDraftInput(t *testing.T) question.DraftInput {
	t.Helper()
	raw, err := os.ReadFile("../../../content/questions/elementary-rationals.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := question.DecodePackage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	ref := question.Ref{ID: "workflow-fractions", Version: 1}
	for i := range p.Templates {
		p.Templates[i].Knowledge = ref
		for j := range p.Templates[i].Coverage {
			p.Templates[i].Coverage[j].Knowledge = ref
		}
	}
	for i := range p.Blueprints {
		p.Blueprints[i].Knowledge = ref
	}
	return question.DraftInput{CatalogueVersion: 1, QuestionPackage: p, SourceMap: []question.SourceLink{}}
}

func TestQuestionDraftOwnership(t *testing.T) {
	f := newWorkflowFixture(t)
	input := questionDraftInput(t)
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), input)
	if err != nil || d.Revision != 1 || d.Status != "editing" {
		t.Fatal("create", err)
	}
	if len(d.Gate.Generation) != 0 || d.Gate.ReadyToSubmit {
		t.Fatal("normal create generated or approved questions")
	}
	if _, err = f.repo.ReadQuestionDraft(f.ctx, f.Access("author_b", false), d.ID); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other editor read", err)
	}
	save := question.SaveDraftInput{DraftInput: input, ExpectedRevision: 1}
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, save); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("other editor wrote", err)
	}
	if _, err = f.repo.ReadQuestionDraft(f.ctx, f.Access("admin_a", false), d.ID); err != nil {
		t.Fatal("admin read", err)
	}
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("admin_a", false), d.ID, save); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("admin authored", err)
	}
	// Legal incomplete text is saved without running generators.
	save.QuestionPackage.Templates[0].PromptTemplate = ""
	changed, err := f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, save)
	if err != nil || changed.Revision != 2 || changed.Gate.ReadyToSubmit || len(changed.Gate.Generation) != 0 {
		t.Fatal("incomplete save", err)
	}
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, save); !errors.Is(err, question.ErrDraftConflict) {
		t.Fatal("stale save", err)
	}
	page, err := f.repo.ListQuestionDrafts(f.ctx, f.Access("author_a", false), question.ListQuery{Scope: "mine"})
	if err != nil || page.Total != 1 || page.Limit != 20 {
		t.Fatal("list", err)
	}
	raw, _ := json.Marshal(page)
	if strings.Contains(string(raw), "sourceMap") || strings.Contains(string(raw), "correctNumeric") {
		t.Fatal("summary disclosed question body")
	}
	if _, err = f.repo.ListQuestionDrafts(f.ctx, f.Access("author_a", false), question.ListQuery{Scope: "all"}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("editor all", err)
	}
	report, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 2})
	if err != nil || report.ReadyToSubmit {
		t.Fatal("unpublished references validated", err)
	}
	private := f.Input()
	private.Package.ID = "private-reference"
	private.Package.Knowledge[0].ID = "private-secret"
	private.Package.Knowledge[0].Title = "PRIVATE KNOWLEDGE TITLE"
	private.Package.Units[0].Knowledge.ID = "private-secret"
	private.Package.Assets[0].Knowledge.ID = "private-secret"
	private.SourceMap[0].Knowledge.ID = "private-secret"
	if _, err = f.repo.CreateDraft(f.ctx, f.Access("author_b", false), private); err != nil {
		t.Fatal(err)
	}
	save = question.SaveDraftInput{DraftInput: input, ExpectedRevision: 2}
	save.QuestionPackage.Templates[0].Knowledge.ID = "private-secret"
	save.QuestionPackage.Templates[0].Coverage[0].Knowledge.ID = "private-secret"
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, save); err != nil {
		t.Fatal("pending reference save", err)
	}
	report, err = f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 3})
	raw, _ = json.Marshal(report)
	if err != nil || report.ReadyToSubmit || strings.Contains(string(raw), "PRIVATE KNOWLEDGE TITLE") {
		t.Fatal("private knowledge exposed", err)
	}
}

func TestQuestionAdoptionPreservesResponsibility(t *testing.T) {
	f := newWorkflowFixture(t)
	// Technical SQL setup represents existing server-verified authorship; no CLI claim is trusted.
	p := question.QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "server-authored-empty", Version: 1, Templates: []question.Template{}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{}}
	raw, sha, err := question.CanonicalPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	var catalogueSHA string
	if err = f.db.QueryRow(`SELECT sha256 FROM catalogue_versions WHERE version=1`).Scan(&catalogueSHA); err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	authors, _ := json.Marshal([]string{f.ids["author_a"]})
	if _, err = tx.Exec(`INSERT INTO question_packages(id,version,sha256,body,body_bytes,catalogue_version,catalogue_sha256,instance_identities,author_ids,legacy_unattributed) VALUES($1,1,$2,$3,$4,1,$5,'[]',$6,true)`, p.ID, sha, string(raw), raw, catalogueSHA, string(authors)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_packages SET sealed=true WHERE id=$1`, p.ID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{9, 10, 2000, 2001} {
		input := question.AdoptInput{PackageID: p.ID, PackageVersion: 1, Reason: strings.Repeat("学", n)}
		d, err := f.repo.AdoptQuestionDraft(f.ctx, f.Access("author_b", false), input)
		if n == 9 || n == 2001 {
			if !errors.Is(err, auth.ErrInvalidInput) {
				t.Fatalf("reason %d: %v", n, err)
			}
			continue
		}
		if err != nil || !d.LegacyUnattributed || len(d.AuthorIDs) != 2 || d.Gate.ReadyToSubmit {
			t.Fatalf("reason %d lost responsibility: %v", n, err)
		}
		changed, err := f.repo.SaveQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.SaveDraftInput{DraftInput: question.DraftInput{CatalogueVersion: 1, QuestionPackage: p, SourceMap: []question.SourceLink{}}, ExpectedRevision: 1})
		if err != nil || !changed.LegacyUnattributed || len(changed.AuthorIDs) != 2 {
			t.Fatal("save cleared authors", err)
		}
	}
}

func TestQuestionDraftValidationUsesCurrentReferences(t *testing.T) {
	f := newWorkflowFixture(t)
	in := questionDraftInput(t)
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_b", false), in)
	if err != nil {
		t.Fatal(err)
	}
	report, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil || report.ReadyToSubmit {
		t.Fatal("private reference accepted", err)
	}
	active := f.Activate(f.Prepare(f.Approved("author_a", "reviewer_a"), nil), nil)
	_ = active
	var digest string
	for i := 0; i < 5; i++ {
		report, err = f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
		if err != nil || !report.ReadyToSubmit || len(report.Generation) != 3 {
			t.Fatal("current public references unavailable", err)
		}
		if i > 0 && digest != report.Digest {
			t.Fatal("unstable validation digest")
		}
		digest = report.Digest
	}
	read, err := f.repo.ReadQuestionDraft(f.ctx, f.Access("author_b", false), d.ID)
	if err != nil || len(read.AuthorIDs) != 1 || read.AuthorIDs[0] != f.ids["author_b"] {
		t.Fatal("knowledge author incorrectly became question author", err)
	}
	bad := f.Access("author_b", false)
	bad.CSRF[0]++
	if _, err = f.repo.ValidateQuestionDraft(f.ctx, bad, d.ID, question.ValidateInput{ExpectedRevision: 1}); !errors.Is(err, auth.ErrCSRF) {
		t.Fatal("validate ignored csrf", err)
	}
	if f.count(`SELECT count(*) FROM question_idempotency WHERE route='validateDraft'`) != 0 {
		t.Fatal("read validation remembered a mutation")
	}
}
