package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type KnowledgeBackupVerification struct {
	RecordSHA       string    `json:"recordSha"`
	DumpSHA         string    `json:"dumpSha"`
	Database        string    `json:"database"`
	BackupCreatedAt time.Time `json:"backupCreatedAt"`
	CheckedAt       time.Time `json:"checkedAt"`
	RestoreVerified bool      `json:"restoreVerified"`
	OffsiteVerified bool      `json:"offsiteVerified"`
	OffsiteHost     string    `json:"offsiteHost"`
}

func privateKnowledgeJSON(path string, value any) error {
	abs, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	resolved, e := filepath.EvalSymlinks(abs)
	if e != nil || resolved != abs || !privateBackupPath(abs, 0600, false) {
		return errors.New("invalid private input")
	}
	info, e := os.Stat(abs)
	if e != nil || info.Size() > 2<<20 {
		return errors.New("invalid private input")
	}
	raw, e := os.ReadFile(abs)
	if e != nil {
		return e
	}
	if _, e = knowledgeadmin.StrictJSON(raw); e != nil {
		return e
	}
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	if e = d.Decode(value); e != nil {
		return e
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return errors.New("invalid private input")
	}
	return nil
}
func validateKnowledgeBackup(ctx context.Context, path, proof, database string) (KnowledgeBackupVerification, error) {
	var v KnowledgeBackupVerification
	record, e := validateBackup(ctx, path, database, 13)
	if e != nil {
		return v, e
	}
	if e = privateKnowledgeJSON(proof, &v); e != nil {
		return v, e
	}
	var manifest struct {
		CreatedAt time.Time `json:"createdAt"`
		DumpSHA   string    `json:"dumpSha256"`
	}
	raw, e := os.ReadFile(filepath.Join(path, "manifest.json"))
	if e != nil || json.Unmarshal(raw, &manifest) != nil {
		return v, errors.New("invalid backup")
	}
	host, _ := os.Hostname()
	if v.RecordSHA != record || v.DumpSHA != manifest.DumpSHA || v.Database != database || !v.BackupCreatedAt.Equal(manifest.CreatedAt) || !v.RestoreVerified || !v.OffsiteVerified || strings.TrimSpace(v.OffsiteHost) == "" || v.OffsiteHost == host || time.Since(v.CheckedAt) > 30*time.Minute || v.CheckedAt.After(time.Now().Add(time.Minute)) || time.Since(manifest.CreatedAt) > 30*time.Minute {
		return v, errors.New("unverified backup")
	}
	return v, nil
}
func RunKnowledge(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	invalid := func() int { return failure(stderr, "INVALID_ARGUMENT") + 1 }
	if len(args) == 0 {
		return invalid()
	}
	operation := args[0]
	if operation != "plan" && operation != "activate" && operation != "clean-old" {
		return invalid()
	}
	f := flag.NewFlagSet("knowledge-maintenance", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var input, backup, verification string
	f.StringVar(&input, "input", "", "")
	f.StringVar(&backup, "backup", "", "")
	f.StringVar(&verification, "verification", "", "")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 || input == "" || backup == "" || verification == "" {
		return invalid()
	}
	var in knowledgeadmin.CutoverInput
	var plan knowledgeadmin.CutoverPlan
	if operation == "clean-old" {
		if privateKnowledgeJSON(input, &plan) != nil {
			return invalid()
		}
		in = plan.Input
	} else if privateKnowledgeJSON(input, &in) != nil {
		return invalid()
	}
	cfg, e := config.Load()
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	target, e := url.Parse(cfg.DatabaseURL)
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute+30*time.Second)
	defer cancel()
	v, e := validateKnowledgeBackup(ctx, backup, verification, strings.TrimPrefix(target.Path, "/"))
	if e != nil {
		return failure(stderr, "BACKUP_RECORD_INVALID")
	}
	if in.BackupRecord != v.RecordSHA || !in.BackupCreatedAt.Equal(v.BackupCreatedAt) || !in.RestoreVerified || !in.OffsiteVerified || in.CodeSHA != cfg.CodeSHA {
		return failure(stderr, "CUTOVER_INPUT_INVALID")
	}
	db, e := sql.Open("pgx", cfg.DatabaseURL)
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	repo := store.NewWithTrustedCodeSHA(db, cfg.CodeSHA)
	var result any
	switch operation {
	case "plan":
		result, e = repo.PlanKnowledgeCutover(ctx, in)
	case "activate":
		result, e = repo.ActivateManagedKnowledge(ctx, in)
	case "clean-old":
		result, e = repo.ApplyKnowledgeCutover(ctx, plan)
	}
	if e != nil {
		return failure(stderr, "KNOWLEDGE_CUTOVER_REFUSED")
	}
	if json.NewEncoder(stdout).Encode(result) != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	return 0
}
