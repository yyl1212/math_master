package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"io"
	"net/url"
	"strings"
	"time"
)

func RunTopicLearning(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return topicLearningArgumentError(stderr)
	}
	operation := args[0]
	f := flag.NewFlagSet("topic-learning-maintenance", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	batches, limit := 1, 50
	var expectedPair, codeSHA, batchID, reason, backupPath string
	f.StringVar(&expectedPair, "expected-pair", "", "")
	f.StringVar(&codeSHA, "code-sha", "", "")
	f.StringVar(&batchID, "migration-batch", "", "")
	f.StringVar(&reason, "reason", "", "")
	f.StringVar(&backupPath, "backup-record", "", "")
	f.IntVar(&batches, "batches", 1, "")
	f.IntVar(&limit, "limit", 50, "")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 || batches < 1 || batches > 10 || limit < 1 || limit > 50 || operation != "migrate" && operation != "inspect" && operation != "verify" && operation != "activate" {
		return topicLearningArgumentError(stderr)
	}
	if operation != "activate" && (expectedPair != "" || codeSHA != "" || batchID != "" || reason != "" || backupPath != "") {
		return topicLearningArgumentError(stderr)
	}
	if operation == "activate" && (expectedPair == "" || codeSHA == "" || batchID == "" || reason == "" || backupPath == "" || len(expectedPair) > 8192) {
		return topicLearningArgumentError(stderr)
	}
	cfg, e := config.Load()
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	db, e := sql.Open("pgx", cfg.DatabaseURL)
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	ctx, cancel := context.WithTimeout(ctx, 540*time.Second)
	defer cancel()
	repo := store.NewWithTrustedCodeSHA(db, cfg.CodeSHA)
	if operation == "activate" {
		var pair taxonomy.PairRef
		decoder := json.NewDecoder(bytes.NewBufferString(expectedPair))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&pair); e != nil {
			return topicLearningArgumentError(stderr)
		}
		var trailing any
		if decoder.Decode(&trailing) != io.EOF {
			return topicLearningArgumentError(stderr)
		}
		target, e := url.Parse(cfg.DatabaseURL)
		if e != nil {
			return failure(stderr, "INVALID_CONFIG")
		}
		record, e := validateTopicBackup(ctx, backupPath, strings.TrimPrefix(target.Path, "/"))
		if e != nil {
			return failure(stderr, "BACKUP_RECORD_INVALID")
		}
		report, e := repo.ActivateTopicExperience(ctx, study.CutoverInput{ExpectedPair: pair, CodeSHA: codeSHA, ExpectedMigrationBatchID: batchID, Reason: reason, BackupRecord: record})
		if e != nil {
			return failure(stderr, "TOPIC_CUTOVER_REFUSED")
		}
		if json.NewEncoder(stdout).Encode(report) != nil {
			return failure(stderr, "OUTPUT_IO")
		}
		return 0
	}
	if operation != "migrate" {
		report, e := repo.InspectTopicCutover(ctx)
		if e != nil {
			return failure(stderr, "TOPIC_SCHEMA_NOT_READY")
		}
		if e = json.NewEncoder(stdout).Encode(report); e != nil {
			return failure(stderr, "OUTPUT_IO")
		}
		if operation == "verify" && (!report.SchemaReady || !report.MigrationDone || !report.HistoryReady || !report.CodeCompatible || report.Pair == nil) {
			return failure(stderr, "TOPIC_CUTOVER_NOT_READY")
		}
		return 0
	}
	var last study.MigrationReport
	var cursor *study.LegacyCursor
	count := 0
	for count < batches {
		last, e = repo.MigrateLegacyStudyBatch(ctx, limit, cursor)
		if e != nil {
			return failure(stderr, "STUDY_MIGRATION_FAILED")
		}
		count++
		cursor = last.Cursor
		if last.Done || last.Processed == 0 {
			break
		}
	}
	if e = json.NewEncoder(stdout).Encode(struct {
		Batches   int                   `json:"batches"`
		LastBatch study.MigrationReport `json:"lastBatch"`
	}{count, last}); e != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	return 0
}
func topicLearningArgumentError(stderr io.Writer) int {
	write(stderr, map[string]string{"code": "INVALID_ARGUMENT", "message": "Use migrate --batches=1..10 --limit=1..50, inspect, verify, or activate with explicit pair/code/migration/reason/backup parameters."})
	return 2
}
