package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"runtime"
	"testing"
	"time"
)

func TestWorkflowCapacityEnvelope(t *testing.T) {
	capacity, err := testutil.CapacityContent("../content/testdata", "../../../content/catalogue/domains.json")
	if err != nil {
		t.Fatal(err)
	}
	s := capacity.Snapshot
	raw, _ := json.Marshal(s)
	if len(raw) != 32<<20 || len(s.Knowledge) != 1000 || len(s.Units) != 4000 || len(s.Paths) != 200 || len(s.Assets) != 1000 || capacity.SVGBytes != 10<<20 {
		t.Fatal("capacity fixture is not simultaneously maximal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	start := time.Now()
	report, err := content.ValidateSnapshot(ctx, capacity.Catalogue, s, capacity.Reader)
	cancel()
	if err != nil || !report.ReadyToSubmit {
		t.Fatalf("legal maximum rejected: %v; structural=%d completeness=%d", err, report.StructuralTotal, report.CompletenessTotal)
	}
	t.Logf("maximum pure validation: %s, canonical=%d SVG=%d", time.Since(start), len(raw), capacity.SVGBytes)
	t.Run("json-over-limit", func(t *testing.T) {
		var over content.Snapshot
		json.Unmarshal(raw, &over)
		over.Knowledge[0].Statement += "x"
		if _, err := content.ValidateSnapshot(context.Background(), capacity.Catalogue, over, capacity.Reader); !errors.Is(err, content.ErrLimit) {
			t.Fatal("32 MiB + 1 accepted")
		}
	})
	t.Run("svg-over-limit", func(t *testing.T) {
		var over content.Snapshot
		if err := json.Unmarshal(raw, &over); err != nil {
			t.Fatal(err)
		}
		original := over.Assets[0]
		data, err := capacity.Reader(context.Background(), original)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.Replace(data, []byte("--></svg>"), []byte("x--></svg>"), 1)
		over.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		for i := range over.Bindings {
			if over.Bindings[i].AssetID == original.ID {
				over.Bindings[i].SHA256 = over.Assets[0].SHA256
			}
		}
		reader := func(ctx context.Context, a content.Asset) ([]byte, error) {
			if a.ID == original.ID {
				return data, nil
			}
			return capacity.Reader(ctx, a)
		}
		report, err := content.ValidateSnapshot(context.Background(), capacity.Catalogue, over, reader)
		found := false
		for _, issue := range report.StructuralErrors {
			if issue.Code == "ASSETS_TOO_LARGE" {
				found = true
			}
		}
		if err == nil || !found {
			t.Fatal("10 MiB + 1 safe unique SVG bytes accepted")
		}
	})
	// Slots stay held while two real maximal validations run, not just until the
	// caller cancels. Reject a third operation without queueing it.
	t.Run("concurrent-maximum-validation", func(t *testing.T) {
		service := publication.NewService(nil)
		releaseA, err := service.AcquireValidation(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		releaseB, err := service.AcquireValidation(context.Background())
		if err != nil {
			releaseA()
			t.Fatal(err)
		}
		if _, err := service.AcquireValidation(context.Background()); err == nil {
			t.Fatal("third maximum validation queued")
		}
		results := make(chan error, 2)
		for _, release := range []func(){releaseA, releaseB} {
			go func(release func()) {
				defer release()
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				_, err := content.ValidateSnapshot(ctx, capacity.Catalogue, s, capacity.Reader)
				results <- err
			}(release)
		}
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal("concurrent maximum rejected", err)
			}
		}
	})
	t.Run("count-over-limit", func(t *testing.T) {
		over := s
		over.Knowledge = append(append([]content.Knowledge{}, s.Knowledge...), s.Knowledge[0])
		if _, err := content.ValidateSnapshot(context.Background(), capacity.Catalogue, over, capacity.Reader); !errors.Is(err, content.ErrLimit) {
			t.Fatal("1001 knowledge accepted")
		}
	})
	f := newWorkflowFixture(t)
	whole, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
	f.ctx = whole
	var submissions []string
	var longest time.Duration
	timed := func(start time.Time) {
		if elapsed := time.Since(start); elapsed > longest {
			longest = elapsed
		}
	}
	for _, input := range capacity.Inputs {
		start = time.Now()
		draft, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), input)
		timed(start)
		if err != nil {
			t.Fatal("maximum package create", err)
		}
		start = time.Now()
		submission, err := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
		timed(start)
		if err != nil {
			t.Fatal("maximum package submit", err)
		}
		start = time.Now()
		submission, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), submission.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Independent isolated capacity fixture reviewer.", Note: "Technical capacity fixture checks only; no production mathematical approval."})
		timed(start)
		if err != nil {
			t.Fatal("maximum package review", err)
		}
		submissions = append(submissions, submission.ID)
	}
	var head *string
	var final publication.PublicationView
	var before, after runtime.MemStats
	for at := 0; at < len(submissions); at += 20 {
		end := at + 20
		if end > len(submissions) {
			end = len(submissions)
		}
		runtime.ReadMemStats(&before)
		start = time.Now()
		prepared, err := f.repo.PrepareRelease(f.ctx, f.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: submissions[at:end], ExpectedHead: head, Reason: "Prepare maximal isolated capacity fixture batches."})
		timed(start)
		if err != nil {
			t.Fatal("maximum candidate prepare", err)
		}
		runtime.ReadMemStats(&after)
		t.Logf("prepare group %d: %s, Go allocated=%d", at/20+1, time.Since(start), after.TotalAlloc-before.TotalAlloc)
		start = time.Now()
		final, err = f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), prepared.ID, publication.ActivateInput{ExpectedHead: head, ExpectedManifestSHA: prepared.ManifestSHA, Reason: "Activate isolated capacity fixture only."})
		timed(start)
		if err != nil {
			t.Fatal("maximum candidate activate", err)
		}
		id := final.ID
		head = &id
	}
	if len(final.Manifest.Members) != 6200 {
		t.Fatal("maximum manifest member count")
	}
	encoded, _ := json.Marshal(final)
	if len(encoded) > 4<<20 {
		t.Fatal("private response exceeds 4 MiB")
	}
	t.Logf("maximum private PublicationView=%d, longest repository operation=%s", len(encoded), longest)
	if longest >= 8*time.Second {
		t.Fatal("repository exceeded content deadline")
	}
	for _, read := range []struct {
		name   string
		budget time.Duration
		call   func(context.Context) error
	}{{"knowledge", 3 * time.Second, func(ctx context.Context) error {
		_, err := f.repo.GetPublishedKnowledge(ctx, s.Knowledge[0].ID)
		return err
	}}, {"path", 5 * time.Second, func(ctx context.Context) error { _, err := f.repo.GetPublishedPath(ctx, s.Paths[0].ID); return err }}} {
		ctx, cancel := context.WithTimeout(context.Background(), read.budget)
		start = time.Now()
		err := read.call(ctx)
		cancel()
		if err != nil {
			t.Fatal("maximum public read", read.name, err)
		}
		t.Logf("public %s: %s", read.name, time.Since(start))
	}
	rows, err := f.db.Query("EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT) SELECT kind,id,version FROM publication_members WHERE snapshot_id=$1", final.ID)
	if err != nil {
		t.Fatal("query plan failed")
	}
	defer rows.Close()
	for rows.Next() {
		var line string
		if rows.Scan(&line) != nil {
			t.Fatal("query plan scan")
		}
		t.Log(line)
	}
	if rows.Err() != nil {
		t.Fatal("query plan incomplete")
	}
	t.Run("two-slots-and-cancellation", func(t *testing.T) {
		service := publication.NewService(f.repo)
		release, err := service.AcquireValidation(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer release()
		release2, err := service.AcquireValidation(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = service.AcquireValidation(context.Background()); err == nil {
			t.Fatal("third slot queued")
		}
		release2()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err = service.AcquireValidation(ctx); err == nil {
			t.Fatal("canceled slot acquired")
		}
	})
	t.Run("lock-wait-and-cancellation", func(t *testing.T) {
		gate := holdAdminLock(t, f.db)
		defer gate.Rollback()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		actor := f.Access("admin_a", false)
		go func() {
			_, err := f.repo.PrepareRelease(ctx, actor, publication.PrepareInput{SubmissionIDs: submissions[:1], ExpectedHead: head, Reason: "Cancel a real blocked capacity operation."})
			done <- err
		}()
		waitForAdminLockWaiters(t, f.db, 1)
		cancel()
		if err := <-done; err == nil {
			t.Fatal("canceled locked operation committed")
		}
	})
}
