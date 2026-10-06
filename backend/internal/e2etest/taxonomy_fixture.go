package e2etest

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
)

// Explicitly reset only the additional tables owned by this isolated harness.
// Production migration and publication commands never use this list.
var taxonomyResetTableNames = []string{
	"taxonomy_heads", "taxonomy_release_assignments", "taxonomy_releases",
	"taxonomy_review_bindings", "taxonomy_submission_assignments", "taxonomy_assignment_drafts",
	"taxonomy_idempotency", "taxonomy_nodes", "taxonomy_versions", "taxonomy_source_batches",
}

// 分类与数学内容全为隔离测试数据，使用真实送审/审核/激活，不作真实数学批准。
func resetTopicCatalogue(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string, normal content.ValidatedPackage) error {
	if e := resetWorkflow(ctx, db, accounts, admin); e != nil {
		return e
	}
	if _, e := s.ImportDraft(ctx, normal); e != nil {
		return e
	}
	in, e := learningFixtureInput(root)
	if e != nil {
		return e
	}
	batch := testutil.TaxonomyBatch()
	refSHA := strings.Repeat("c", 64)
	path := "Original/browser-fixture.json"
	batch.Manifest.SourceFiles = append(batch.Manifest.SourceFiles, taxonomy.SourceFile{Path: "Knowledge_JSON/" + path, SizeBytes: 2, SHA256: refSHA})
	files, _ := json.Marshal(batch.Manifest.SourceFiles)
	sha := sha256.Sum256(files)
	batch.Manifest.SnapshotID = hex.EncodeToString(sha[:])
	members := []taxonomy.AssignmentInput{}
	in.SourceMap = []publication.SourceLink{}
	for i, k := range in.Package.Knowledge {
		ref := taxonomy.SourceRecordRef{SourceID: "original-browser-source", WorkFamilyID: "original-browser-work", RecordID: k.ID, Path: path, SHA256: refSHA}
		batch.SourceRecordIndex = append(batch.SourceRecordIndex, ref)
		in.SourceMap = append(in.SourceMap, publication.SourceLink{Knowledge: content.VersionRef{ID: k.ID, Version: k.Version}, BatchSHA256: batch.Manifest.SnapshotID, RelativePath: path, SHA256: refSHA, LegacyID: k.ID, Note: "Original fixture evidence; no real mathematical approval."})
		topics := []string{fmt.Sprintf("msc-00a%02d", i)}
		if i == 0 {
			topics = append(topics, "msc-00a01")
		}
		members = append(members, taxonomy.AssignmentInput{Knowledge: content.VersionRef{ID: k.ID, Version: k.Version}, TopicIDs: topics, SourceRefs: []taxonomy.SourceRecordRef{ref}, SourceBatchSHA: batch.Manifest.SnapshotID})
	}
	version, e := s.InstallTaxonomyBatch(ctx, batch)
	if e != nil {
		return e
	}
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	draft, e := s.CreateDraft(ctx, author, in)
	if e != nil {
		return e
	}
	revision := int64(0)
	for _, member := range members {
		author, e = nextFixtureAccess(author)
		if e != nil {
			return e
		}
		saved, e := s.SaveDraftTopics(ctx, author, draft.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: draft.Revision, ExpectedAssignmentRevision: revision, TaxonomyVersionID: version.ID, Member: member})
		if e != nil {
			return e
		}
		revision = saved.AssignmentRevision
	}
	author, e = nextFixtureAccess(author)
	if e != nil {
		return e
	}
	sub, e := s.SubmitDraft(ctx, author, draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
	if e != nil {
		return e
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	if _, e = s.DecideReview(ctx, reviewer, sub.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Separate technical fixture author and reviewer accounts.", Note: "Original isolated fixture only; no real independent mathematical approval."}); e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	pair := taxonomy.PairRef{TaxonomyVersionID: version.ID}
	release, e := s.PrepareTopicRelease(ctx, manager, taxonomy.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedPair: pair, Reason: "Prepare original isolated catalogue and knowledge together."})
	if e != nil {
		return e
	}
	manager, e = nextFixtureAccess(manager)
	if e != nil {
		return e
	}
	_, e = s.ActivateTopicRelease(ctx, manager, release.ID, taxonomy.ActivateInput{ExpectedPair: pair, ManifestSHA: release.ManifestSHA, Reason: "Activate only this original isolated browser fixture."})
	return e
}
