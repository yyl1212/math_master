package e2etest

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"strings"
)

// 仅随机隔离harness模拟topics；正式切换由维护命令实施。
func resetTopicStudy(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string, normal content.ValidatedPackage, extras ...bool) error {
	if e := resetTopicCatalogue(ctx, db, s, accounts, admin, root, normal, extras...); e != nil {
		return e
	}
	_, e := db.ExecContext(ctx, "UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton")
	return e
}

func publishAwaitingTopic(ctx context.Context, s *store.Store, accounts *auth.Service, root string) error {
	in, e := learningFixtureInput(root)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	in.Package.ID = "legacy-awaiting-topic"
	k := in.Package.Knowledge[0]
	k.ID = "study-awaiting-topic"
	k.Title = "Awaiting topic original fixture"
	k.TitleZh = "待归类原创夹具"
	k.Relations = []content.Relation{}
	in.Package.Knowledge = []content.Knowledge{k}
	unit := in.Package.Units[0]
	unit.ID = "study-awaiting-topic-unit"
	unit.Knowledge.ID = k.ID
	in.Package.Units = []content.Unit{unit}
	in.Package.Paths = []content.Path{}
	for i := range in.Package.Assets {
		old := in.Package.Assets[i].ID
		in.Package.Assets[i].ID = "awaiting-" + old
		in.Package.Assets[i].Knowledge.ID = k.ID
		in.AssetBytes[i].ID = in.Package.Assets[i].ID
		for j := range in.Package.Units[0].AssetIDs {
			if in.Package.Units[0].AssetIDs[j] == old {
				in.Package.Units[0].AssetIDs[j] = in.Package.Assets[i].ID
			}
		}
		for j := range in.Package.Units[0].Angles {
			in.Package.Units[0].Angles[j].Body = strings.ReplaceAll(in.Package.Units[0].Angles[j].Body, "asset:"+old, "asset:"+in.Package.Assets[i].ID)
		}
	}
	in.SourceMap = []publication.SourceLink{}
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	d, e := s.CreateDraft(ctx, author, in)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	author, e = nextFixtureAccess(author)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	sub, e := s.SubmitDraft(ctx, author, d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	if _, e = s.DecideReview(ctx, reviewer, sub.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Separate original fixture accounts.", Note: "Original isolated unclassified legacy fixture only."}); e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	p, e := s.PrepareRelease(ctx, manager, publication.PrepareInput{SubmissionIDs: []string{sub.ID}, Reason: "Prepare an original legacy unclassified fixture."})
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	manager, e = nextFixtureAccess(manager)
	if e != nil {
		return fmt.Errorf("awaiting legacy setup: %w", e)
	}
	_, e = s.ActivateRelease(ctx, manager, p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Publish original pre-taxonomy unclassified fixture."})
	return e
}
