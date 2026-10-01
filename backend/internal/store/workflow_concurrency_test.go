package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"testing"
)

func TestConcurrentReviewFinalState(t *testing.T) {
	f := newWorkflowFixture(t)
	sub := f.Submitted("author_a")
	gate := holdAdminLock(t, f.db)
	results := make(chan error, 2)
	for _, name := range []string{"reviewer_a", "reviewer_b"} {
		a := f.Access(name, false)
		go func() { _, err := f.repo.DecideReview(f.ctx, a, sub.ID, approvedReviewInput()); results <- err }()
	}
	waitForAdminLockWaiters(t, f.db, 2)
	if err := gate.Commit(); err != nil {
		t.Fatal(err)
	}
	success, conflict := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, publication.ErrReviewConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 || f.count(`SELECT count(*) FROM content_review_decisions`) != 1 || f.count(`SELECT count(*) FROM content_workflow_events WHERE action='decideReview'`) != 1 {
		t.Fatal("concurrent review left multiple terminal records")
	}
}
func TestReviewRoleRevocationRacesDecision(t *testing.T) {
	for _, first := range []string{"revocation", "decision"} {
		t.Run(first, func(t *testing.T) {
			f := newWorkflowFixture(t)
			sub := f.Submitted("author_a")
			gate := holdAdminLock(t, f.db)
			decisionResult := make(chan error, 1)
			revokeResult := make(chan error, 1)
			actor := f.Access("reviewer_a", false)
			admin := f.Access("admin_a", true)
			revoke := func() {
				_, err := f.repo.ReplaceAccountRoles(f.ctx, auth.SessionProof{TokenHash: admin.TokenHash, CSRF: admin.CSRF}, f.ids["reviewer_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "review-role-race")
				revokeResult <- err
			}
			decide := func() {
				_, err := f.repo.DecideReview(f.ctx, actor, sub.ID, approvedReviewInput())
				decisionResult <- err
			}
			if first == "revocation" {
				go revoke()
				waitForAdminLockWaiters(t, f.db, 1)
				go decide()
			} else {
				go decide()
				waitForAdminLockWaiters(t, f.db, 1)
				go revoke()
			}
			waitForAdminLockWaiters(t, f.db, 2)
			if err := gate.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-revokeResult; err != nil {
				t.Fatal(err)
			}
			err := <-decisionResult
			if first == "revocation" {
				if !errors.Is(err, auth.ErrAuthenticationRequired) && !errors.Is(err, auth.ErrForbidden) {
					t.Fatal("decision ignored earlier revocation", err)
				}
				if f.count(`SELECT count(*) FROM content_review_decisions`) != 0 {
					t.Fatal("revoked decision persisted")
				}
			} else {
				if err != nil {
					t.Fatal("first decision failed", err)
				}
				if f.count(`SELECT count(*) FROM content_review_decisions`) != 1 {
					t.Fatal("historical decision erased")
				}
			}
			if _, err = f.repo.ReadSession(f.ctx, actor.TokenHash, false); !errors.Is(err, auth.ErrAuthenticationRequired) {
				t.Fatal("role mutation left old reviewer session valid")
			}
		})
	}
}
