package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
)

// Original technical fixtures only, in the harness's verified disposable database.
func resetStudyReview(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string) error {
	if e := resetManagedKnowledge(ctx, db, s, accounts, admin, root); e != nil {
		return e
	}
	editor, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		return e
	}
	a := knowledgeadmin.Access{TokenHash: editor.TokenHash, CSRF: editor.CSRF, RequestID: "study-review-fixture", IdempotencyKey: "review-proof-prompt"}
	proofID := knowledgeadmin.KnowledgeID("demo-equivalent-fractions")
	proof, e := s.ReadManagedKnowledge(ctx, a, proofID)
	if e != nil {
		return e
	}
	in := knowledgeadmin.CurrentInput{ExternalID: proof.ExternalID, Point: proof.Point, Sources: proof.Sources, TopicKeys: proof.TopicKeys}
	in.Point.Statement = "在题干条件下，下式成立。"
	in.Point.Equations = []string{`$a/b=(ac)/(bc)$`}
	if _, e = s.UpdateManagedKnowledge(ctx, a, proofID, proof.EditToken, in); e != nil {
		return e
	}
	learner, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		return e
	}
	owner := knowledgeadmin.Access{TokenHash: learner.TokenHash, CSRF: learner.CSRF, RequestID: "study-review-fixture"}
	for _, external := range []string{"demo-rational-fraction", "demo-equivalent-fractions"} {
		k, e := s.ReadCurrentKnowledge(ctx, knowledgeadmin.KnowledgeID(external))
		if e != nil {
			return e
		}
		owner.IdempotencyKey = "review-begin-" + external
		begun, e := s.ApplyManagedStudy(ctx, owner, k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
		if e != nil {
			return e
		}
		owner.IdempotencyKey = "review-complete-" + external
		if _, e = s.ApplyManagedStudy(ctx, owner, k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: begun.Record.Sequence}); e != nil {
			return e
		}
	}
	return nil
}
