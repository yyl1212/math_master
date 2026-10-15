package store_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestTopicCutoverCapacityPrepare(t *testing.T) {
	path := os.Getenv(cutoverCapacityReceiptEnv)
	if path == "" {
		t.Fatal("capacity receipt path required")
	}
	ctx := context.Background()
	stageStarted := time.Now()
	f, _ := studyCapacityPublished(t, ctx)
	t.Log("capacity published fixture", time.Since(stageStarted))
	stageStarted = time.Now()
	f.exec(`WITH users AS(INSERT INTO auth_users(id,username,password_phc) SELECT gen_random_uuid(),'cutover_capacity_'||lpad(n::text,4,'0'),u.password_phc FROM generate_series(1,499) n CROSS JOIN auth_users u WHERE u.id=$1 RETURNING id) INSERT INTO auth_user_roles(user_id,role) SELECT id,'learner' FROM users`, f.ids["author_a"])
	rows, e := f.db.Query(`SELECT id::text FROM auth_users WHERE username LIKE 'cutover_capacity_%' OR id=$1 ORDER BY id`, f.ids["author_a"])
	if e != nil {
		t.Fatal(e)
	}
	owners := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			t.Fatal(e)
		}
		owners = append(owners, id)
	}
	rows.Close()
	if len(owners) != 500 {
		t.Fatal("owner fixture count")
	}
	var sourcePublication string
	if e = f.db.QueryRow(`SELECT p.id FROM publication_snapshots p JOIN content_publication_manifests m ON m.snapshot_id=p.id WHERE p.status='published' ORDER BY m.created_at,p.id LIMIT 1`).Scan(&sourcePublication); e != nil {
		t.Fatal(e)
	}
	rows, e = f.db.Query(`SELECT k.id,k.version,k.sha256 FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active' ORDER BY k.id LIMIT 200`, sourcePublication)
	if e != nil {
		t.Fatal(e)
	}
	points := []question.Identity{}
	for rows.Next() {
		var k question.Identity
		if e = rows.Scan(&k.ID, &k.Version, &k.SHA256); e != nil {
			t.Fatal(e)
		}
		points = append(points, k)
	}
	rows.Close()
	if len(points) != 200 {
		t.Fatal("point fixture count")
	}
	// Original immutable approvals supply each technical historical snapshot.
	// Its one selected member keeps source seeding independent of the current
	// 1,000-point manifest size; no review evidence or public head is changed.
	var manifestRaw []byte
	if e = f.db.QueryRow(`SELECT manifest_bytes FROM content_publication_manifests WHERE snapshot_id=$1`, sourcePublication).Scan(&manifestRaw); e != nil {
		t.Fatal(e)
	}
	var original publication.Manifest
	if e = json.Unmarshal(manifestRaw, &original); e != nil {
		t.Fatal(e)
	}
	publications := map[string]string{}
	for _, k := range points {
		m := original
		m.BaseHead = nil
		m.Members = nil
		m.Bindings = nil
		for _, member := range original.Members {
			if member.Identity.Kind == "knowledge" && member.Identity.ID == k.ID {
				member.Evidence.InheritedFrom = nil
				m.Members = []publication.ManifestMember{member}
				break
			}
		}
		if len(m.Members) != 1 {
			t.Fatal("approved source member missing")
		}
		raw, e := publication.ManifestBytes(m)
		if e != nil {
			t.Fatal(e)
		}
		sha, e := publication.ManifestDigest(m)
		if e != nil {
			t.Fatal(e)
		}
		id := f.ID()
		f.exec(`INSERT INTO publication_snapshots(id,catalogue_version,status) VALUES($1,$2,'draft')`, id, m.CatalogueVersion)
		f.exec(`INSERT INTO publication_members(snapshot_id,package_id,package_version,kind,id,version,availability) SELECT $1,package_id,package_version,kind,id,version,availability FROM publication_members WHERE snapshot_id=$2 AND kind='knowledge' AND id=$3`, id, sourcePublication, k.ID)
		f.exec(`INSERT INTO content_publication_manifests(snapshot_id,manifest,manifest_bytes,sha256,diff,creator_user_id) VALUES($1,$2,$3,$4,'{"added":1,"replaced":0,"removed":0,"changes":[]}', $5)`, id, string(raw), raw, sha, f.ids["admin_a"])
		f.exec(`UPDATE publication_snapshots SET status='published' WHERE id=$1`, id)
		publications[k.ID] = id
	}
	t.Log("capacity owners and historical snapshots", time.Since(stageStarted))
	base := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Microsecond)
	seedStarted := time.Now()
	// Two source transactions use the two bounded database CPUs. Knowledge
	// refs differ, every original seal/record/dependency guard stays enabled,
	// and only preparation is parallel; all 2,000 migration batches stay serial.
	var generateTime, eventTime, recordTime, dependencyTime, commitTime atomic.Int64
	seed := func(first int) error {
		measured := time.Now()

		values := make([]capacitySeedRow, 0, 2000)
		end := first + 4
		if end > len(points) {
			end = len(points)
		}
		for i := first; i < end; i++ {
			k := points[i]
			publication := publications[k.ID]
			for j, owner := range owners {
				id, e := auth.NewID(rand.Reader)
				if e != nil {
					return e
				}
				at := base.Add(time.Duration(i*500+j) * time.Microsecond)
				raw, sha, e := learning.CanonicalLearningEvent(learning.EventSeal{ID: id, ActorID: owner, Knowledge: k, KnowledgePublicationID: publication, Units: []question.Identity{}, Assets: []question.AssetRef{}, Kind: "started", RecordedAt: at})
				if e != nil {
					return e
				}
				values = append(values, capacitySeedRow{ID: id, Owner: owner, Knowledge: k.ID, Version: k.Version, SHA: k.SHA256, Publication: publication, Seal: json.RawMessage(raw), Bytes: base64.StdEncoding.EncodeToString(raw), Digest: sha, At: at})
			}
		}
		generateTime.Add(int64(time.Since(measured)))
		times, e := writeCapacitySeedBatch(ctx, f.db, values)
		eventTime.Add(int64(times.Events))
		recordTime.Add(int64(times.Records))
		dependencyTime.Add(int64(times.Dependencies))
		commitTime.Add(int64(times.Commit))
		if e != nil {
			return e
		}

		return nil
	}
	jobs := make(chan int)
	results := make(chan error, 2)
	for worker := 0; worker < 2; worker++ {
		go func() {
			var first error
			for i := range jobs {
				if first == nil {
					first = seed(i)
				}
			}
			results <- first
		}()
	}
	for i := 0; i < len(points); i += 4 {
		jobs <- i
	}
	close(jobs)
	var seedError error
	for worker := 0; worker < 2; worker++ {
		if e := <-results; e != nil && seedError == nil {
			seedError = e
		}
	}
	if seedError != nil {
		t.Fatal(seedError)
	}
	t.Log("sealed source100000", time.Since(seedStarted))
	t.Log("capacity summed worker durations", "generate", time.Duration(generateTime.Load()), "events", time.Duration(eventTime.Load()), "records", time.Duration(recordTime.Load()), "dependencies", time.Duration(dependencyTime.Load()), "commit", time.Duration(commitTime.Load()))
	if f.count("SELECT count(*) FROM learning_events") != 100000 {
		t.Fatal("source event count")
	}
	stageStarted = time.Now()
	persistCapacityFixture(t, f.db, path)
	t.Log("capacity private handoff", time.Since(stageStarted))
}

func TestTopicCutoverCapacityMigrate(t *testing.T) {
	path := os.Getenv(cutoverCapacityReceiptEnv)
	if path == "" {
		t.Fatal("capacity receipt path required")
	}
	// Register cleanup before opening, including identity or assertion failures.
	t.Cleanup(func() { cleanupCapacityFixture(t, path) })
	db, receipt := openCapacityFixture(t, path)
	if capacitySourceDigest(t, db) != receipt.SourceSHA {
		t.Fatal("capacity source changed between stages")
	}
	ctx := context.Background()
	f := &workflowFixture{authFixture: &authFixture{t: t, db: db, ctx: ctx, repo: store.New(db)}}
	if f.count("SELECT count(*) FROM learning_events") != 100000 || f.count("SELECT count(DISTINCT owner_user_id) FROM learning_events") != 500 || f.count("SELECT count(*) FROM study_events") != 0 || f.count("SELECT count(*) FROM study_legacy_event_links") != 0 {
		t.Fatal("capacity handoff incomplete or reused")
	}
	var e error
	changes := f.count("SELECT count(*) FROM study_content_changes")
	migrationStarted := time.Now()
	total := 0
	var cursor *study.LegacyCursor
	var report study.MigrationReport
	maxBatch := time.Duration(0)
	for n := 0; n < 2000; n++ {
		start := time.Now()
		report, e = f.repo.MigrateLegacyStudyBatch(ctx, 50, cursor)
		elapsed := time.Since(start)
		if e != nil || report.Processed > 50 || elapsed > 8*time.Second {
			t.Fatal("finite migration capacity", n, elapsed, e)
		}
		if elapsed > maxBatch {
			maxBatch = elapsed
		}
		total += report.CreatedEvents
		cursor = report.Cursor
	}
	if total != 100000 || !report.Done || f.count("SELECT count(*) FROM study_events WHERE kind='completed'") != 0 || f.count("SELECT count(*) FROM study_legacy_event_links") != 100000 || f.count("SELECT count(*) FROM study_content_changes") != changes {
		t.Fatal("capacity facts/done/fanout mismatch")
	}
	if capacitySourceDigest(t, db) != receipt.SourceSHA {
		t.Fatal("original source changed during migration")
	}
	t.Log("migrated100000", time.Since(migrationStarted), "max50 batch", maxBatch)
	replay, e := f.repo.MigrateLegacyStudyBatch(ctx, 50, nil)
	if e != nil || replay.CreatedEvents != 0 || !replay.Done {
		t.Fatal("capacity replay", e)
	}
}
