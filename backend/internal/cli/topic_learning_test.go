package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
)

func TestStudyMigrationCLIFiniteEmptyBatch(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("CORRECTION_WORKER_ENABLED", "false")
	var out, err bytes.Buffer
	if n := RunTopicLearning(ctx, []string{"migrate", "--batches=2", "--limit=1"}, &out, &err); n != 0 {
		t.Fatal("finite migration unavailable", n)
	}
	var r struct {
		Batches   int `json:"batches"`
		LastBatch struct {
			Processed int  `json:"processed"`
			Done      bool `json:"done"`
		} `json:"lastBatch"`
	}
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Batches != 1 || !r.LastBatch.Done || r.LastBatch.Processed != 0 {
		t.Fatal("empty migration not explicit")
	}
}

func TestStudyMigrationCLIArgumentsAreFiniteAndSanitized(t *testing.T) {
	for _, args := range [][]string{{}, {"migrate", "--batches=0"}, {"migrate", "--batches=11"}, {"migrate", "--limit=0"}, {"migrate", "--limit=51"}, {"migrate", "--database-url=secret"}, {"migrate", "--loop"}, {"unknown"}} {
		var out, err bytes.Buffer
		if n := RunTopicLearning(context.Background(), args, &out, &err); n != 2 || out.Len() != 0 || !json.Valid(err.Bytes()) || strings.Contains(err.String(), "secret") {
			t.Fatal("invalid maintenance bounds", n)
		}
	}
}
func TestStudyMigrationCLIInspectionAndVerification(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("CORRECTION_WORKER_ENABLED", "false")
	for _, operation := range []string{"inspect", "verify"} {
		var out, err bytes.Buffer
		n := RunTopicLearning(ctx, []string{operation}, &out, &err)
		if !json.Valid(out.Bytes()) || operation == "inspect" && n != 0 || operation == "verify" && n == 0 {
			t.Fatal("no audited batch reported done", operation, n)
		}
	}
	var out, err bytes.Buffer
	if n := RunTopicLearning(ctx, []string{"migrate"}, &out, &err); n != 0 {
		t.Fatal(n)
	}
	out.Reset()
	err.Reset()
	if n := RunTopicLearning(ctx, []string{"verify"}, &out, &err); n == 0 || !json.Valid(out.Bytes()) {
		t.Fatal("actual empty migration verification unavailable", n)
	}
}
