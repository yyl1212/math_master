package e2etest

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func resetNoteSharing(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string) error {
	if e := resetManagedKnowledge(ctx, db, s, accounts, admin, root); e != nil {
		return e
	}
	adminProof, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		return e
	}
	learnerProof, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		return e
	}
	a := knowledgeadmin.Access{TokenHash: adminProof.TokenHash, CSRF: adminProof.CSRF, RequestID: "note-sharing-fixture"}
	owner := knowledgeadmin.Access{TokenHash: learnerProof.TokenHash, CSRF: learnerProof.CSRF, RequestID: "note-sharing-fixture"}
	a.IdempotencyKey = "read-base"
	base, e := s.ReadManagedKnowledge(ctx, a, knowledgeadmin.KnowledgeID("demo-rational-fraction"))
	if e != nil {
		return e
	}
	for i := 1; i <= 13; i++ {
		id := fmt.Sprintf("note-sharing-original-fixture-%02d", i)
		point := base.Point
		point.ID = id
		point.Title = fmt.Sprintf("Original technical knowledge %02d", i)
		point.TitleZH = fmt.Sprintf("原创技术知识 %02d", i)
		a.IdempotencyKey = fmt.Sprintf("create-note-knowledge-%d", i)
		k, e := s.CreateManagedKnowledge(ctx, a, knowledgeadmin.CurrentInput{ExternalID: id, Point: point, Sources: base.Sources, TopicKeys: base.TopicKeys})
		if e != nil {
			return e
		}
		a.IdempotencyKey = fmt.Sprintf("publish-note-knowledge-%d", i)
		k, e = s.SetManagedKnowledgeState(ctx, a, k.ID, k.EditToken, "publish")
		if e != nil {
			return e
		}
		owner.IdempotencyKey = fmt.Sprintf("save-fixture-note-%d", i)
		n, e := s.SaveManagedNote(ctx, owner, k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: fmt.Sprintf("原创学习技术夹具 %02d：$x^2$。", i)})
		if e != nil {
			return e
		}
		owner.IdempotencyKey = fmt.Sprintf("submit-fixture-note-%d", i)
		r, e := s.Submit(ctx, owner, k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: fmt.Sprintf("Shared technical note %02d", i)})
		if e != nil {
			return e
		}
		if i <= 12 {
			a.IdempotencyKey = fmt.Sprintf("approve-fixture-note-%d", i)
			if _, e = s.Decide(ctx, a, r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence}); e != nil {
				return e
			}
		}
	}
	return nil
}
