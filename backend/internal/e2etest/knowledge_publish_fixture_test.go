package e2etest

import (
	"context"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestKnowledgePublishFixture41(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	root, e := rootDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, root+"/db/migrations"); e != nil {
		t.Fatal(e)
	}
	s := store.New(db)
	accounts, admin, e := fixtureAccounts(s)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetKnowledgePublish(ctx, db, s, accounts, admin, root); e != nil {
		t.Fatal(e)
	}
	proof, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		t.Fatal(e)
	}
	a := k.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: "fixture-job", RequestID: "test"}
	j, e := s.CreatePublishJob(ctx, a, k.PublishCreateInput{Mode: k.PublishAll})
	if e != nil || j.Total != 41 {
		t.Fatal(j, e)
	}
	a.IdempotencyKey = "fixture-step"
	r, e := s.StepPublishJob(ctx, a, j.ID, k.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Published != 20 || r.Pending != 21 {
		t.Fatal(r, e)
	}
}
