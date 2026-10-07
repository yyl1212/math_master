package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/buildmeta"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTopicCutoverCLIExplicitArguments(t *testing.T) {
	for _, a := range [][]string{{"activate"}, {"activate", "--code-sha=" + strings.Repeat("a", 40)}, {"activate", "--database-url=secret"}, {"inspect", "--code-sha=" + strings.Repeat("a", 40)}} {
		var out, err bytes.Buffer
		if n := RunTopicLearning(context.Background(), a, &out, &err); n != 2 || strings.Contains(err.String(), "secret") || out.Len() != 0 {
			t.Fatal("unsafe or implicit activate", n)
		}
	}
}
func TestTopicCutoverBackupRecordValidation(t *testing.T) {
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	dump := []byte("PGDMP Original isolated backup fixture only.")
	sha := fmt.Sprintf("%x", sha256.Sum256(dump))
	manifest := map[string]any{"schemaVersion": 1, "sourceCommit": strings.Repeat("a", 40), "createdAt": time.Now().UTC(), "migrationVersion": 12, "dumpSha256": sha, "database": map[string]string{"name": "math_master_test_fixture"}, "tables": map[string]any{}}
	raw, _ := json.Marshal(manifest)
	record := fmt.Sprintf("%x", sha256.Sum256(raw))
	for name, data := range map[string][]byte{"database.dump": dump, "manifest.json": raw, "SHA256SUMS": []byte(sha + "  database.dump\n" + record + "  manifest.json\n")} {
		if e = os.WriteFile(filepath.Join(root, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	got, e := validateTopicBackup(context.Background(), root, "math_master_test_fixture")
	if e != nil || got != record {
		t.Fatal("valid private record rejected", e)
	}
	if _, e = validateTopicBackup(context.Background(), root, "another_database"); e == nil {
		t.Fatal("wrong database backup accepted")
	}
	if e = os.WriteFile(filepath.Join(root, "database.dump"), []byte("PGDMP changed fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = validateTopicBackup(context.Background(), root, "math_master_test_fixture"); e == nil {
		t.Fatal("modified dump accepted")
	}
}

func TestTopicCutoverCLIActualActivationAndReplay(t *testing.T) {
	ctx := context.Background()
	db := testutil.Database(t)
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("CORRECTION_WORKER_ENABLED", "false")
	old := buildmeta.Revision
	buildmeta.Revision = strings.Repeat("a", 40)
	defer func() { buildmeta.Revision = old }()
	repo := store.NewWithTrustedCodeSHA(db, buildmeta.Revision)
	hasher := auth.NewArgon2Hasher(rand.Reader)
	accounts, e := auth.NewService(repo, hasher, rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	admin := auth.NewAdminService(repo, hasher, rand.Reader)
	password := "Original cutover-only 中文 password with spaces"
	if _, e = admin.Initialize(ctx, "cutover_admin", password, "isolated-cutover"); e != nil {
		t.Fatal(e)
	}
	view, delta, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		t.Fatal(e)
	}
	_, delta, e = accounts.Login(ctx, auth.Cookies{Preauth: delta.SetPreauth}, view.CSRFToken, auth.LoginInput{Username: "cutover_admin", Password: password}, "isolated-login")
	if e != nil {
		t.Fatal(e)
	}
	cookies := auth.Cookies{Session: delta.SetSession}
	view, _, e = accounts.Context(ctx, cookies)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = accounts.Reauthenticate(ctx, cookies, view.CSRFToken, auth.ReauthInput{Password: password}, "isolated-reauth"); e != nil {
		t.Fatal(e)
	}
	proof, e := auth.DecodeContentProof(cookies, view.CSRFToken, true)
	if e != nil {
		t.Fatal(e)
	}
	access := publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: "11111111-1111-4111-8111-111111111111", RequestID: "isolated-cli"}
	version, e := repo.InstallTaxonomyBatch(ctx, testutil.TaxonomyBatch())
	if e != nil {
		t.Fatal(e)
	}
	pair := taxonomy.PairRef{TaxonomyVersionID: version.ID}
	release, e := repo.PrepareTopicRelease(ctx, access, taxonomy.PrepareInput{ExpectedPair: pair, SubmissionIDs: []string{}, Reason: "Prepare original empty CLI fixture."})
	if e != nil {
		t.Fatal(e)
	}
	access.IdempotencyKey = "22222222-2222-4222-8222-222222222222"
	if _, e = repo.ActivateTopicRelease(ctx, access, release.ID, taxonomy.ActivateInput{ExpectedPair: pair, ManifestSHA: release.ManifestSHA, Reason: "Activate original empty CLI fixture."}); e != nil {
		t.Fatal(e)
	}
	pair.TaxonomyHead = &release.ID
	batch, e := repo.MigrateLegacyStudyBatch(ctx, 50, nil)
	if e != nil {
		t.Fatal(e)
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	var database string
	if e = db.QueryRow("SELECT current_database()").Scan(&database); e != nil {
		t.Fatal(e)
	}
	dump := []byte("PGDMP Original test-only metadata backup record.")
	dumpSHA := fmt.Sprintf("%x", sha256.Sum256(dump))
	manifest, _ := json.Marshal(map[string]any{"schemaVersion": 1, "sourceCommit": buildmeta.Revision, "createdAt": time.Now().UTC(), "migrationVersion": 12, "dumpSha256": dumpSHA, "database": map[string]string{"name": database}, "tables": map[string]any{}})
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256(manifest))
	for name, data := range map[string][]byte{"database.dump": dump, "manifest.json": manifest, "SHA256SUMS": []byte(dumpSHA + "  database.dump\n" + manifestSHA + "  manifest.json\n")} {
		if e = os.WriteFile(filepath.Join(root, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	raw, _ := json.Marshal(pair)
	args := []string{"activate", "--expected-pair=" + string(raw), "--code-sha=" + buildmeta.Revision, "--migration-batch=" + batch.BatchID, "--reason=Explicit isolated CLI activation only.", "--backup-record=" + root}
	var originalID string
	for n := 0; n < 2; n++ {
		var out, err bytes.Buffer
		if status := RunTopicLearning(ctx, args, &out, &err); status != 0 {
			t.Fatal("explicit CLI activation unavailable", status, err.String())
		}
		var report struct {
			Activated bool    `json:"activated"`
			CutoverID *string `json:"cutoverId"`
		}
		if json.Unmarshal(out.Bytes(), &report) != nil || report.CutoverID == nil || report.Activated != (n == 0) {
			t.Fatal("incorrect activation/replay report")
		}
		if n == 0 {
			originalID = *report.CutoverID
		} else if *report.CutoverID != originalID {
			t.Fatal("audit changed")
		}
	}
}
