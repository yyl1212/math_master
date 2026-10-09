package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func managedKnowledgeChange(ctx context.Context, s *store.Store, accounts *auth.Service, operation string) error {
	proof, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		return e
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: proof.IdempotencyKey, RequestID: "managed-browser-change"}
	id := knowledgeadmin.KnowledgeID("demo-rational-fraction")
	k, e := s.ReadManagedKnowledge(ctx, a, id)
	if e != nil {
		return e
	}
	if operation == "unpublish" {
		_, e = s.SetManagedKnowledgeState(ctx, a, id, k.EditToken, "unpublish")
		return e
	}
	in := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}
	switch operation {
	case "update":
		in.Point.Statement += " Browser correction preserves learning notes."
	case "move":
		in.Point.MSCCodes = []string{"97F50"}
		in.Point.ClassificationEvidence[0].MSCCode = "97F50"
	default:
		return knowledgeadmin.ErrInvalid
	}
	_, e = s.UpdateManagedKnowledge(ctx, a, id, k.EditToken, in)
	return e
}
