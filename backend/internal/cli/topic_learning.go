package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"io"
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
	f.IntVar(&batches, "batches", 1, "")
	f.IntVar(&limit, "limit", 50, "")
	if e := f.Parse(args[1:]); e != nil || f.NArg() != 0 || batches < 1 || batches > 10 || limit < 1 || limit > 50 || operation != "migrate" && operation != "inspect" && operation != "verify" {
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
	repo := store.New(db)
	if operation != "migrate" {
		report, e := repo.InspectStudyMigration(ctx)
		if e != nil {
			return failure(stderr, "TOPIC_SCHEMA_NOT_READY")
		}
		if e = json.NewEncoder(stdout).Encode(report); e != nil {
			return failure(stderr, "OUTPUT_IO")
		}
		if operation == "verify" && !report.MigrationDone {
			return failure(stderr, "STUDY_MIGRATION_PENDING")
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
	write(stderr, map[string]string{"code": "INVALID_ARGUMENT", "message": "Use migrate --batches=1..10 --limit=1..50, inspect or verify."})
	return 2
}
