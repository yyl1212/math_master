package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"io"

	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/store"
)

type correctionSummary struct {
	Batches   int               `json:"batches"`
	Claimed   int               `json:"claimed"`
	Processed int               `json:"processed"`
	LastBatch *correction.Batch `json:"lastBatch"`
}

func RunCorrection(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("correction-maintenance", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	batches, limit := 1, 50
	f.IntVar(&batches, "batches", 1, "")
	f.IntVar(&limit, "limit", 50, "")
	if e := f.Parse(args); e != nil || f.NArg() != 0 || batches < 1 || batches > 10 || limit < 1 || limit > 50 {
		write(stderr, map[string]string{"code": "INVALID_ARGUMENT", "message": "Use --batches=1..10 and --limit=1..50."})
		return 2
	}
	c, e := config.Load()
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	db, e := sql.Open("pgx", c.DatabaseURL)
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(2)
	// Explicit finite maintenance ignores only the resident worker switch, never
	// the transactional guards, ownership, lease token or statement deadlines.
	r, e := correction.NewRunner(store.New(db), correction.Options{Enabled: true, Limit: limit})
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	sum := correctionSummary{}
	for n := 0; n < batches; n++ {
		v, e := r.RunOnce(ctx)
		sum.Batches++
		sum.Processed += v.Processed
		if v.Claimed {
			sum.Claimed++
			copy := v
			sum.LastBatch = &copy
		}
		if e != nil {
			code := "CORRECTION_BATCH_FAILED"
			if errors.Is(e, correction.ErrNotConfigured) {
				code = "CORRECTION_NOT_CONFIGURED"
			}
			return failure(stderr, code)
		}
		if !v.Claimed {
			break
		}
	}
	if e = json.NewEncoder(stdout).Encode(sum); e != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	return 0
}
