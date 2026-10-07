package store_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
	"time"
)

func TestTopicCutoverCapacity(t *testing.T) {
	ctx := context.Background()
	f, _ := studyCapacityPublished(t, ctx)
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
	base := time.Now().UTC().Add(-24 * time.Hour).Truncate(time.Microsecond)
	seedStarted := time.Now()
	// Capacity rows are original sealed started facts with all original constraints,
	// never invented completed states or assessment-derived completions.
	for i, k := range points {
		publication := publications[k.ID]
		values := []map[string]any{}
		for j, owner := range owners {
			id := f.ID()
			at := base.Add(time.Duration(i*500+j) * time.Microsecond)
			raw, sha, e := learning.CanonicalLearningEvent(learning.EventSeal{ID: id, ActorID: owner, Knowledge: k, KnowledgePublicationID: publication, Units: []question.Identity{}, Assets: []question.AssetRef{}, Kind: "started", RecordedAt: at})
			if e != nil {
				t.Fatal(e)
			}
			values = append(values, map[string]any{"id": id, "owner": owner, "kid": k.ID, "version": k.Version, "sha": k.SHA256, "publication": publication, "seal": json.RawMessage(raw), "bytes": base64.StdEncoding.EncodeToString(raw), "digest": sha, "at": at})
		}
		raw, _ := json.Marshal(values)
		tx, e := f.db.BeginTx(ctx, nil)
		if e != nil {
			t.Fatal(e)
		}
		const selected = `jsonb_to_recordset($1::jsonb) x(id uuid,owner uuid,kid text,version integer,sha text,publication text,seal jsonb,bytes text,digest text,at timestamptz)`
		if _, e = tx.Exec(`INSERT INTO learning_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,kind,seal,seal_bytes,seal_sha256,recorded_at) SELECT id,owner,kid,version,sha,publication,'started',seal,decode(bytes,'base64'),digest,at FROM `+selected, string(raw)); e != nil {
			tx.Rollback()
			t.Fatal(e)
		}
		if _, e = tx.Exec(`INSERT INTO learning_records(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,started_event_id,started_at) SELECT owner,kid,version,sha,id,at FROM `+selected, string(raw)); e != nil {
			tx.Rollback()
			t.Fatal(e)
		}
		if _, e = tx.Exec(`INSERT INTO learning_evidence_dependencies(evidence_kind,evidence_id,owner_user_id,kind,id,version,sha256) SELECT 'learning-event',x.id,x.owner,d.kind,d.id,d.version,d.sha256 FROM `+selected+` CROSS JOIN LATERAL learning_seal_dependencies(x.seal,true) d`, string(raw)); e != nil {
			tx.Rollback()
			t.Fatal(e)
		}
		if e = tx.Commit(); e != nil {
			t.Fatal(e)
		}
	}
	t.Log("sealed source100000", time.Since(seedStarted))
	if f.count("SELECT count(*) FROM learning_events") != 100000 {
		t.Fatal("source event count")
	}
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
	t.Log("migrated100000", time.Since(migrationStarted), "max50 batch", maxBatch)
	replay, e := f.repo.MigrateLegacyStudyBatch(ctx, 50, nil)
	if e != nil || replay.CreatedEvents != 0 || !replay.Done {
		t.Fatal("capacity replay", e)
	}
}
