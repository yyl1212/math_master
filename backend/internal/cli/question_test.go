package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestQuestionCLIArchiveRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if err := store.Up(ctx, db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	setURL(t, db)
	repo := store.New(db)
	cf, err := os.Open(args()[1])
	if err != nil {
		t.Fatal(err)
	}
	cat, err := content.DecodeCatalogue(cf)
	cf.Close()
	if err != nil {
		t.Fatal(err)
	}
	pf, err := os.Open(args()[3])
	if err != nil {
		t.Fatal(err)
	}
	p, err := content.DecodePackage(pf)
	pf.Close()
	if err != nil {
		t.Fatal(err)
	}
	p.ID = "question-reference-fixture"
	for i := range p.Knowledge {
		if p.Knowledge[i].ID == "fractions" {
			p.Knowledge[i].Objectives = []string{"Add exact fractions."}
		}
	}
	cv, report := content.ValidateAndSeal(cat, p, args()[5])
	if len(report.Errors) > 0 {
		t.Fatal("technical content fixture invalid")
	}
	if _, err = repo.ImportDraft(ctx, cv); err != nil {
		t.Fatal(err)
	}
	head := "draft-" + cv.SHA256()
	if _, err = db.Exec(`UPDATE publication_snapshots SET status='published' WHERE id=$1`, head); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO publication_heads VALUES(true,$1)`, head); err != nil {
		t.Fatal(err)
	}
	qpRaw, err := os.ReadFile("../../../content/questions/elementary-rationals.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	qp, err := question.DecodePackage(bytes.NewReader(qpRaw))
	if err != nil {
		t.Fatal(err)
	}
	for i := range qp.Templates {
		qp.Templates[i].Knowledge.Version = 1
		for j := range qp.Templates[i].Coverage {
			qp.Templates[i].Coverage[j].Knowledge.Version = 1
		}
	}
	for i := range qp.Blueprints {
		qp.Blueprints[i].Knowledge.Version = 1
	}
	refs := question.ReferenceSnapshot{KnowledgeHead: &head, CatalogueVersion: 1, CatalogueSHA256: cv.CatalogueSHA256(), Knowledge: []question.FixedKnowledge{}, Units: []question.FixedUnit{}, Assets: []content.AssetView{}}
	for _, k := range p.Knowledge {
		if k.ID == "fractions" {
			refs.Knowledge = append(refs.Knowledge, question.FixedKnowledge{Identity: question.Identity{ID: k.ID, Version: k.Version, SHA256: content.Digest(k)}, Title: k.Title, TitleZh: k.TitleZh, Objectives: k.Objectives})
		}
	}
	input := question.DraftInput{CatalogueVersion: 1, QuestionPackage: qp, SourceMap: []question.SourceLink{{Knowledge: question.Ref{ID: "fractions", Version: 1}, BatchSHA256: strings.Repeat("a", 64), SHA256: strings.Repeat("b", 64), RelativePath: "elementary/rationals.json", LegacyID: "technical-rationals", Note: "An original technical source mapping."}}}
	sealed, gate, err := question.ValidateAndSeal(ctx, input, refs)
	if err != nil || !gate.ReadyToSubmit {
		t.Fatal("technical question fixture invalid", err)
	}
	archive := question.Archive{Envelope: question.DraftInput{CatalogueVersion: 1, QuestionPackage: sealed.Package, SourceMap: input.SourceMap}, PackageSHA: sealed.PackageSHA, Instances: sealed.Instances, GeneratorVersions: []int{1}, VerifierVersions: []int{1}, SourceResponsibility: question.SourceResponsibility{AuthorIDs: []string{}, LegacyUnattributed: true}}
	root := t.TempDir()
	archiveFile := filepath.Join(root, "archive.json")
	refFile := filepath.Join(root, "references.json")
	writeFixture := func(path string, v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(archiveFile, archive)
	writeFixture(refFile, refs)
	var out, stderr bytes.Buffer
	run := func(command string, arguments ...string) int {
		out.Reset()
		stderr.Reset()
		return RunQuestion(ctx, command, arguments, &out, &stderr)
	}
	if code := run("check", "--archive", archiveFile, "--references", refFile); code != 0 {
		t.Fatalf("check %d %s", code, stderr.String())
	}
	if code := run("import", "--archive", archiveFile); code != 0 {
		t.Fatalf("import %d %s", code, stderr.String())
	}
	var imported question.QuestionImportResult
	if err = json.Unmarshal(out.Bytes(), &imported); err != nil || imported.PackageSHA != archive.PackageSHA || imported.Status != "draft" || imported.ImportedInstances != 28 {
		t.Fatal("wrong unapproved import", err)
	}
	dir := filepath.Join(root, "export")
	if code := run("export", "--id", qp.ID, "--version", "1", "--out", dir); code != 0 {
		t.Fatalf("export %d %s", code, stderr.String())
	}
	exported, err := repo.ExportQuestionArchive(ctx, qp.ID, 1)
	if err != nil || exported.PackageSHA != archive.PackageSHA || len(exported.Instances) != 28 || !exported.SourceResponsibility.LegacyUnattributed || len(exported.SourceResponsibility.AuthorIDs) != 0 {
		t.Fatal("archive changed", err)
	}
	if !reflect.DeepEqual(exported.Envelope.SourceMap, archive.Envelope.SourceMap) || !reflect.DeepEqual(exported.GeneratorVersions, archive.GeneratorVersions) || !reflect.DeepEqual(exported.VerifierVersions, archive.VerifierVersions) {
		t.Fatal("source map or engines changed")
	}
	for i := range archive.Instances {
		if exported.Instances[i].Identity != archive.Instances[i].Identity {
			t.Fatal("instance identity changed")
		}
	}
	if code := run("check", "--archive", filepath.Join(dir, "archive.json"), "--references", refFile); code != 0 {
		t.Fatal("export check failed", stderr.String())
	}
	if code := run("export", "--id", qp.ID, "--version", "1", "--out", dir); code != 1 {
		t.Fatal("existing directory overwritten")
	}
	for _, table := range []string{"question_submissions", "question_review_decisions", "question_publications", "question_heads"} {
		var n int
		if err = db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatal("CLI imported approval")
		}
	}
	// Claiming a platform author is never proof: the immutable local record decides.
	archive.SourceResponsibility.AuthorIDs = []string{"11111111-1111-4111-8111-111111111111"}
	archive.SourceResponsibility.LegacyUnattributed = false
	writeFixture(archiveFile, archive)
	if code := run("import", "--archive", archiveFile); code != 0 {
		t.Fatal("same-library replay failed")
	}
	exported, err = repo.ExportQuestionArchive(ctx, qp.ID, 1)
	if err != nil || !exported.SourceResponsibility.LegacyUnattributed || len(exported.SourceResponsibility.AuthorIDs) != 0 {
		t.Fatal("trusted a claimed author")
	}

	const knownAuthor = "88888888-8888-4888-8888-888888888888"
	known := exported
	known.Envelope.QuestionPackage.ID = "server-authored-archive"
	pkgBytes, pkgSHA, err := question.CanonicalPackage(known.Envelope.QuestionPackage)
	if err != nil {
		t.Fatal(err)
	}
	identities := []question.Identity{}
	for _, i := range known.Instances {
		identities = append(identities, i.Identity)
	}
	idsRaw, _ := json.Marshal(identities)
	sourceRaw, _ := json.Marshal(known.Envelope.SourceMap)
	knownAuthors, _ := json.Marshal([]string{knownAuthor})
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO auth_users(id,username,password_phc) VALUES($1,'question_original_author','isolated-test-only')`, knownAuthor); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO auth_user_roles VALUES($1,'learner')`, knownAuthor); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO question_packages(id,version,sha256,body,body_bytes,catalogue_version,catalogue_sha256,instance_identities,source_map,author_ids,legacy_unattributed) VALUES('server-authored-archive',1,$1,$2,$3,1,$4,$5,$6,$7,true)`, pkgSHA, string(pkgBytes), pkgBytes, refs.CatalogueSHA256, string(idsRaw), string(sourceRaw), string(knownAuthors)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_packages SET sealed=true WHERE id='server-authored-archive'`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	knownDir := filepath.Join(root, "known-author-export")
	if code := run("export", "--id", "server-authored-archive", "--version", "1", "--out", knownDir); code != 0 {
		t.Fatal("known author export failed", stderr.String())
	}
	if code := run("import", "--archive", filepath.Join(knownDir, "archive.json")); code != 0 {
		t.Fatal("known author import failed", stderr.String())
	}
	known, err = repo.ExportQuestionArchive(ctx, "server-authored-archive", 1)
	if err != nil || len(known.SourceResponsibility.AuthorIDs) != 1 || known.SourceResponsibility.AuthorIDs[0] != knownAuthor {
		t.Fatal("same-library author not inherited", err)
	}

	conflict := exported
	conflict.Envelope.QuestionPackage.ID = "atomic-conflict-copy"
	oldID := conflict.Envelope.QuestionPackage.Templates[0].ID
	conflict.Envelope.QuestionPackage.Templates[0].ID = "aaaa-fresh-template"
	conflict.Envelope.QuestionPackage.Templates[1].ExplanationTemplate += " This revised explanation requires a new version."
	for i := range conflict.Envelope.QuestionPackage.Blueprints {
		for j := range conflict.Envelope.QuestionPackage.Blueprints[i].Sources {
			if conflict.Envelope.QuestionPackage.Blueprints[i].Sources[j].Ref.ID == oldID {
				conflict.Envelope.QuestionPackage.Blueprints[i].Sources[j].Ref.ID = "aaaa-fresh-template"
			}
		}
	}
	conflictSealed, _, err := question.ValidateAndSeal(ctx, conflict.Envelope, refs)
	if err != nil {
		t.Fatal("conflict fixture invalid", err)
	}
	conflict.Envelope.QuestionPackage = conflictSealed.Package
	conflict.PackageSHA = conflictSealed.PackageSHA
	conflict.Instances = conflictSealed.Instances
	conflictFile := filepath.Join(root, "conflict.json")
	writeFixture(conflictFile, conflict)
	var packagesBefore, templatesBefore int
	if err = db.QueryRow(`SELECT count(*) FROM question_packages`).Scan(&packagesBefore); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM question_templates`).Scan(&templatesBefore); err != nil {
		t.Fatal(err)
	}
	if code := run("import", "--archive", conflictFile); code != 2 || !strings.Contains(stderr.String(), "VERSION_CONFLICT") {
		t.Fatal("fixed conflict accepted", stderr.String())
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM question_packages`).Scan(&n); err != nil || n != packagesBefore {
		t.Fatal("partial package committed")
	}
	if err = db.QueryRow(`SELECT count(*) FROM question_templates`).Scan(&n); err != nil || n != templatesBefore {
		t.Fatal("partial fixed template committed")
	}
	badPath := filepath.Join(root, "bad.json")
	b, _ := json.Marshal(archive)
	b = bytes.Replace(b, []byte(`"envelope":`), []byte(`"reviewedBy":"someone","envelope":`), 1)
	if err = os.WriteFile(badPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	if code := run("import", "--archive", badPath); code != 2 || strings.Contains(stderr.String(), root) {
		t.Fatal("claim/path leaked")
	}
}
