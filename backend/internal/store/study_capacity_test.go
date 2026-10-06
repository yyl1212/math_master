package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

func studyCapacityPublished(t *testing.T, ctx context.Context) (*workflowFixture, taxonomy.PairRef) {
	capacity, e := testutil.CapacityContent("../content/testdata", "../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	f := newWorkflowFixture(t)
	whole := ctx
	f.ctx = whole
	batch := testutil.TaxonomyBatch()
	path := "capacity/original.json"
	sourceSHA := strings.Repeat("c", 64)
	batch.Manifest.SourceFiles = append(batch.Manifest.SourceFiles, taxonomy.SourceFile{Path: "Knowledge_JSON/" + path, SizeBytes: 2, SHA256: sourceSHA})
	files, _ := json.Marshal(batch.Manifest.SourceFiles)
	hash := sha256.Sum256(files)
	batch.Manifest.SnapshotID = hex.EncodeToString(hash[:])
	for _, k := range capacity.Snapshot.Knowledge {
		batch.SourceRecordIndex = append(batch.SourceRecordIndex, taxonomy.SourceRecordRef{SourceID: "capacity-original", WorkFamilyID: "capacity-original-work", RecordID: k.ID, Path: path, SHA256: sourceSHA})
	}
	version, e := f.repo.InstallTaxonomyBatch(whole, batch)
	if e != nil {
		t.Fatal(e)
	}
	submissions := []string{}
	for _, in := range capacity.Inputs {
		in.SourceMap = []publication.SourceLink{}
		for _, k := range in.Package.Knowledge {
			in.SourceMap = append(in.SourceMap, publication.SourceLink{Knowledge: content.VersionRef{ID: k.ID, Version: k.Version}, BatchSHA256: batch.Manifest.SnapshotID, RelativePath: path, SHA256: sourceSHA, LegacyID: k.ID, Note: "Original capacity technical fixture only."})
		}
		d, e := f.repo.CreateDraft(whole, f.Access("author_a", false), in)
		if e != nil {
			t.Fatal("create capacity draft", e)
		}
		revision := int64(0)
		for _, k := range in.Package.Knowledge {
			m := taxonomy.AssignmentInput{Knowledge: content.VersionRef{ID: k.ID, Version: k.Version}, TopicIDs: []string{"msc-00a00", "msc-00a01"}, SourceRefs: []taxonomy.SourceRecordRef{{SourceID: "capacity-original", WorkFamilyID: "capacity-original-work", RecordID: k.ID, Path: path, SHA256: sourceSHA}}, SourceBatchSHA: batch.Manifest.SnapshotID}
			v, e := f.repo.SaveDraftTopics(whole, f.Access("author_a", false), d.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: d.Revision, ExpectedAssignmentRevision: revision, TaxonomyVersionID: version.ID, Member: m})
			if e != nil {
				t.Fatal("save capacity assignment", e)
			}
			revision = v.AssignmentRevision
		}
		sub, e := f.repo.SubmitDraft(whole, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
		if e != nil {
			t.Fatal("submit capacity draft", e)
		}
		if _, e = f.repo.DecideReview(whole, f.Access("reviewer_a", false), sub.ID, approvedReviewInput()); e != nil {
			t.Fatal("review capacity draft", e)
		}
		submissions = append(submissions, sub.ID)
	}
	pair := taxonomy.PairRef{TaxonomyVersionID: version.ID}
	for at := 0; at < len(submissions); at += 20 {
		end := at + 20
		if end > len(submissions) {
			end = len(submissions)
		}
		p, e := f.repo.PrepareTopicRelease(whole, f.Access("admin_a", false), taxonomy.PrepareInput{SubmissionIDs: submissions[at:end], ExpectedPair: pair, Reason: "Prepare maximum original isolated taxonomy fixture."})
		if e != nil {
			t.Fatal("capacity prepare", e)
		}
		if _, e = f.repo.ActivateTopicRelease(whole, f.Access("admin_a", true), p.ID, taxonomy.ActivateInput{ExpectedPair: pair, ManifestSHA: p.ManifestSHA, Reason: "Activate maximum original isolated taxonomy fixture."}); e != nil {
			t.Fatal("capacity activate", e)
		}
		pair.KnowledgeHead = p.KnowledgePublicationID
		pair.TaxonomyHead = &p.ID
	}

	return f, pair
}

func TestStudyCapacityReadPages(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	f, pair := studyCapacityPublished(t, ctx)
	f.exec(`WITH created AS(INSERT INTO auth_users(id,username,password_phc) SELECT gen_random_uuid(),'study_capacity_'||lpad(n::text,4,'0'),u.password_phc FROM generate_series(1,499) n CROSS JOIN auth_users u WHERE u.id=$1 RETURNING id) INSERT INTO auth_user_roles(user_id,role) SELECT id,'learner' FROM created`, f.ids["author_a"])
	f.exec(`WITH owners AS(SELECT id FROM auth_users WHERE username LIKE 'study_capacity_%' OR id=$1),points AS(SELECT k.id,k.version,k.sha256 FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$2 AND m.kind='knowledge' AND m.availability='active' ORDER BY k.id LIMIT 100),clock AS(SELECT clock_timestamp()-interval '1 day' AS at)
 INSERT INTO study_records(owner_user_id,knowledge_id,state,sequence,body,last_known_ref,updated_at)
 SELECT o.id,k.id,'completed',2,jsonb_build_object('knowledgeId',k.id,'state','completed','sequence',2,'firstStartedAt',c.at,'firstCompletedAt',c.at,'lastCompletedAt',c.at,'lastReadAt',c.at,'lastReviewedAt',NULL,'completedRef',jsonb_build_object('id',k.id,'version',k.version,'sha256',k.sha256),'lastReviewRef',NULL,'lastReviewId',NULL,'activeReviewId',NULL),jsonb_build_object('id',k.id,'version',k.version,'sha256',k.sha256),c.at FROM owners o CROSS JOIN points k CROSS JOIN clock c`, f.ids["author_a"], *pair.KnowledgeHead)
	f.exec(`INSERT INTO study_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,taxonomy_version_id,taxonomy_head,kind,recorded_at,source_kind,action,idempotency_key) SELECT gen_random_uuid(),r.owner_user_id,r.knowledge_id,(r.last_known_ref->>'version')::integer,r.last_known_ref->>'sha256',$1,$2,v.kind,(r.body->>'firstStartedAt')::timestamptz,'native',v.kind,gen_random_uuid() FROM study_records r CROSS JOIN(VALUES('started'),('completed'))v(kind)`, pair.TaxonomyVersionID, *pair.TaxonomyHead)
	if f.count("SELECT count(*) FROM study_events") != 100000 || f.count("SELECT count(DISTINCT owner_user_id) FROM study_records") != 500 {
		t.Fatal("capacity fixture incomplete")
	}
	a0 := f.Access("author_a", false)
	a := study.Access{TokenHash: a0.TokenHash, CSRF: a0.CSRF, IdempotencyKey: a0.IdempotencyKey, RequestID: a0.RequestID}
	check := func(name string, fn func() (any, error)) {
		t.Helper()
		start := time.Now()
		value, e := fn()
		elapsed := time.Since(start)
		raw, _ := json.Marshal(value)
		if e != nil || elapsed > 8*time.Second || len(raw) > 2<<20 {
			t.Fatal(name, elapsed, len(raw), e)
		}
		t.Log(name, elapsed, len(raw))
	}
	before := f.count("SELECT count(*) FROM study_content_changes")
	check("overview", func() (any, error) { return f.repo.ReadStudyOverview(ctx, a) })
	check("63 theme progress", func() (any, error) { return f.repo.ListStudyTopics(ctx, a, study.ListQuery{Limit: 100}) })
	check("100 knowledge page", func() (any, error) { return f.repo.ListStudyKnowledge(ctx, a, study.ListQuery{Limit: 100}) })
	check("50 of 100000 history", func() (any, error) { return f.repo.ListStudyHistory(ctx, a, study.HistoryQuery{Limit: 50}) })
	if f.count("SELECT count(*) FROM study_content_changes") != before {
		t.Fatal("reads changed publication facts")
	}
}
