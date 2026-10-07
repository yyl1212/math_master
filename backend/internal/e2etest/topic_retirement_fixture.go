package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"
)

// Only the token/loopback harness calls this simulated mode change. C7 uses the real maintenance cutover.
func enableTopicRetirementFixture(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service) error {
	version, e := s.InstallTaxonomyBatch(ctx, testutil.TaxonomyBatch())
	if e != nil {
		return e
	}
	var head sql.NullString
	if e = db.QueryRowContext(ctx, "SELECT snapshot_id FROM publication_heads WHERE singleton").Scan(&head); e != nil && e != sql.ErrNoRows {
		return e
	}
	pair := taxonomy.PairRef{TaxonomyVersionID: version.ID}
	if head.Valid {
		pair.KnowledgeHead = &head.String
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	p, e := s.PrepareTopicRelease(ctx, manager, taxonomy.PrepareInput{ExpectedPair: pair, SubmissionIDs: []string{}, Reason: "Classification-only isolated retirement fixture."})
	if e != nil {
		return e
	}
	manager, e = nextFixtureAccess(manager)
	if e != nil {
		return e
	}
	if _, e = s.ActivateTopicRelease(ctx, manager, p.ID, taxonomy.ActivateInput{ExpectedPair: p.Pair, ManifestSHA: p.ManifestSHA, Reason: "Keep original legacy knowledge and attempt facts."}); e != nil {
		return e
	}
	_, e = db.ExecContext(ctx, "UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton")
	return e
}
