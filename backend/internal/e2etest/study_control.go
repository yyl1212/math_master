package e2etest

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

func changeStudyPublication(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, operation string) error {
	state, e := readTaxonomyControl(ctx, db)
	if e != nil {
		return e
	}
	if operation == "withdraw" {
		var version int
		if e = db.QueryRowContext(ctx, "SELECT k.version FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.id='learning-root' AND m.availability='active'", *state.Pair.KnowledgeHead).Scan(&version); e != nil {
			return e
		}
		manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
		if e != nil {
			return e
		}
		_, e = s.WithdrawVersion(ctx, manager, publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: "learning-root", Version: version}, ExpectedHead: state.Pair.KnowledgeHead, Reason: "Withdraw only original isolated study fixture content."})
		return e
	}
	if operation != "move" && operation != "upgrade" {
		return fmt.Errorf("unknown study fixture operation")
	}
	var submissionID string
	if e = db.QueryRowContext(ctx, "SELECT evidence_submission_id::text FROM taxonomy_release_assignments WHERE release_id=$1 AND knowledge_id='learning-root' LIMIT 1", *state.Pair.TaxonomyHead).Scan(&submissionID); e != nil {
		return e
	}
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	sub, e := s.ReadSubmission(ctx, author, submissionID)
	if e != nil {
		return e
	}
	topics, e := s.ReadSubmissionTopics(ctx, author, submissionID)
	if e != nil {
		return e
	}
	in := publication.DraftInput{CatalogueVersion: sub.Frozen.CatalogueVersion, Package: sub.Frozen.Package, SourceMap: sub.Frozen.SourceMap, AssetBytes: []publication.AssetInput{}}
	for _, asset := range sub.Frozen.Assets {
		data, e := s.ReadSubmissionAsset(ctx, author, submissionID, asset.SHA256)
		if e != nil {
			return e
		}
		in.AssetBytes = append(in.AssetBytes, publication.AssetInput{ID: asset.ID, Base64: base64.StdEncoding.EncodeToString(data)})
	}
	if operation == "move" {
		for i := range topics.Members {
			if topics.Members[i].Knowledge.ID == "learning-root" {
				topics.Members[i].TopicIDs = []string{"msc-13c60"}
			}
		}
	} else {
		in.Package.Version++
		for i := range in.Package.Knowledge {
			in.Package.Knowledge[i].Version++
			for j := range in.Package.Knowledge[i].Relations {
				in.Package.Knowledge[i].Relations[j].Target.Version++
			}
			in.Package.Knowledge[i].Scope += " Updated original fixture reading scope."
		}
		for i := range in.Package.Units {
			in.Package.Units[i].Version++
			in.Package.Units[i].Knowledge.Version++
		}
		for i := range in.Package.Assets {
			in.Package.Assets[i].Knowledge.Version++
		}
		for i := range in.Package.Paths {
			in.Package.Paths[i].Version++
			for j := range in.Package.Paths[i].Nodes {
				in.Package.Paths[i].Nodes[j].Version++
			}
		}
		for i := range in.SourceMap {
			in.SourceMap[i].Knowledge.Version++
		}
		for i := range topics.Members {
			topics.Members[i].Knowledge.Version++
		}
	}
	author, e = nextFixtureAccess(author)
	if e != nil {
		return e
	}
	draft, e := s.CreateDraft(ctx, author, in)
	if e != nil {
		return e
	}
	revision := int64(0)
	for _, member := range topics.Members {
		author, e = nextFixtureAccess(author)
		if e != nil {
			return e
		}
		view, e := s.SaveDraftTopics(ctx, author, draft.ID, taxonomy.DraftTopicInput{ExpectedDraftRevision: draft.Revision, ExpectedAssignmentRevision: revision, TaxonomyVersionID: state.Pair.TaxonomyVersionID, Member: member})
		if e != nil {
			return e
		}
		revision = view.AssignmentRevision
	}
	author, e = nextFixtureAccess(author)
	if e != nil {
		return e
	}
	frozen, e := s.SubmitDraft(ctx, author, draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
	if e != nil {
		return e
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	if _, e = s.DecideReview(ctx, reviewer, frozen.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Separate original fixture accounts.", Note: "Review only isolated study fixture update."}); e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	p, e := s.PrepareTopicRelease(ctx, manager, taxonomy.PrepareInput{SubmissionIDs: []string{frozen.ID}, ExpectedPair: state.Pair, Reason: "Prepare an original isolated study publication update."})
	if e != nil {
		return e
	}
	manager, e = nextFixtureAccess(manager)
	if e != nil {
		return e
	}
	_, e = s.ActivateTopicRelease(ctx, manager, p.ID, taxonomy.ActivateInput{ExpectedPair: state.Pair, ManifestSHA: p.ManifestSHA, Reason: "Activate an original isolated study publication update."})
	return e
}
