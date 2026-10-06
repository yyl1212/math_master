package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestTaxonomyCapacity6603With1000Knowledge(t *testing.T) {
	capacity, e := testutil.CapacityContent("../content/testdata", "../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	f := newWorkflowFixture(t)
	whole, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
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
	start := time.Now()
	root, e := f.repo.ReadTopic(whole, "msc-00")
	elapsed := time.Since(start)
	if e != nil || root.Summary.PublishedKnowledgeCount != 1000 || elapsed > 8*time.Second {
		t.Fatal("root capacity", elapsed, e)
	}
	t.Log("deduplicated 1000 knowledge", elapsed)
	start = time.Now()
	page, e := f.repo.ListTopicKnowledge(whole, "msc-00", taxonomy.Query{Limit: 100})
	elapsed = time.Since(start)
	raw, _ := json.Marshal(page)
	if e != nil || page.Total != 1000 || len(page.Items) != 100 || elapsed > 8*time.Second || len(raw) > 2<<20 {
		t.Fatal("paged capacity", elapsed, len(raw), e)
	}
	t.Log("100 item page", elapsed, "bytes", len(raw))
	nodes, e := f.repo.ListTopics(whole, taxonomy.Query{Level: 3, Limit: 100})
	if e != nil || nodes.Total != 4969 || len(nodes.Items) != 100 {
		t.Fatal("classification page", e)
	}
	// 正常满容量读之后也必须拒绝缺失的不可变性保护，不能降级为旧流程。
	f.exec("ALTER TABLE taxonomy_release_assignments DISABLE TRIGGER taxonomy_member_guard")
	if _, e = f.repo.ReadTopic(whole, "msc-00"); !errors.Is(e, taxonomy.ErrNotConfigured) {
		t.Fatal("disabled membership guard accepted", e)
	}
}
