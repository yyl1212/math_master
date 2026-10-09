package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKnowledgeCLIExplicitPrivateInputs(t *testing.T) {
	for _, args := range [][]string{{}, {"activate"}, {"plan", "--database-url=secret"}, {"clean-old", "--input=missing"}, {"unknown"}} {
		var out, err bytes.Buffer
		if code := RunKnowledge(context.Background(), args, &out, &err); code != 2 || out.Len() != 0 || strings.Contains(err.String(), "secret") {
			t.Fatal(code, out.String(), err.String())
		}
	}
}
func TestKnowledgeBackupAcceptsPortableSnapshotOnlyWithMatchingProof(t *testing.T) {
	root, pathError := filepath.EvalSymlinks(t.TempDir())
	if pathError != nil {
		t.Fatal(pathError)
	}
	if e := os.Chmod(root, 0700); e != nil {
		t.Fatal(e)
	}
	dump := []byte("PGDMP Test-only portable snapshot fixture.")
	h := sha256.Sum256(dump)
	dumpSHA := hex.EncodeToString(h[:])
	at := time.Now().UTC()
	manifest := map[string]any{"schemaVersion": 1, "sourceCommit": strings.Repeat("a", 64)[:40], "createdAt": at, "migrationVersion": 13, "dumpSha256": dumpSHA, "database": map[string]any{"encoding": "UTF8"}, "tables": map[string]any{}}
	write := func(database map[string]any) string {
		manifest["database"] = database
		raw, e := json.Marshal(manifest)
		if e != nil {
			t.Fatal(e)
		}
		hash := sha256.Sum256(raw)
		record := hex.EncodeToString(hash[:])
		for name, body := range map[string][]byte{"database.dump": dump, "manifest.json": raw, "SHA256SUMS": []byte(dumpSHA + "  database.dump\n" + record + "  manifest.json\n")} {
			if e = os.WriteFile(filepath.Join(root, name), body, 0600); e != nil {
				t.Fatal(e)
			}
		}
		proof := KnowledgeBackupVerification{RecordSHA: record, DumpSHA: dumpSHA, Database: "math_master_preview", BackupCreatedAt: at, CheckedAt: at, RestoreVerified: true, OffsiteVerified: true, OffsiteHost: "independent-test-host"}
		raw, e = json.Marshal(proof)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(root, "verification.json"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		return record
	}
	write(map[string]any{"encoding": "UTF8"})
	if _, e := validateKnowledgeBackup(context.Background(), root, filepath.Join(root, "verification.json"), "math_master_preview"); e != nil {
		t.Fatal("portable production snapshot rejected", e)
	}
	manifest["migrationVersion"] = 12
	write(map[string]any{"encoding": "UTF8"})
	if _, e := validateTopicBackup(context.Background(), root, "math_master_preview"); e == nil {
		t.Fatal("old topic backup accepted unnamed snapshot")
	}
	manifest["migrationVersion"] = 13
	for _, database := range []map[string]any{{"name": "another_database"}, {"name": nil}, {"name": ""}} {
		write(database)
		if _, e := validateKnowledgeBackup(context.Background(), root, filepath.Join(root, "verification.json"), "math_master_preview"); e == nil {
			t.Fatal("invalid explicit database accepted", database)
		}
	}
	write(map[string]any{"encoding": "UTF8"})
	if _, e := validateKnowledgeBackup(context.Background(), root, filepath.Join(root, "verification.json"), "another_database"); e == nil {
		t.Fatal("proof target database ignored")
	}
	var wrongProof KnowledgeBackupVerification
	proofPath := filepath.Join(root, "verification.json")
	proofBytes, e := os.ReadFile(proofPath)
	if e != nil || json.Unmarshal(proofBytes, &wrongProof) != nil {
		t.Fatal("invalid proof fixture")
	}
	wrongProof.Database = "another_database"
	proofBytes, e = json.Marshal(wrongProof)
	if e != nil || os.WriteFile(proofPath, proofBytes, 0600) != nil {
		t.Fatal("cannot write wrong proof fixture")
	}
	if _, e = validateKnowledgeBackup(context.Background(), root, proofPath, "math_master_preview"); e == nil || e.Error() != "unverified backup" {
		t.Fatal("wrong proof was not rejected at the proof boundary", e)
	}
}
