package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestManagedKnowledgeFixtureDirectPublication(t *testing.T) {
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
	if e = resetManagedKnowledge(ctx, db, s, accounts, admin, root); e != nil {
		t.Fatal(e)
	}
	page, e := s.ListCurrentKnowledge(ctx, knowledgeadmin.Query{})
	if e != nil || page.Total != 2 {
		t.Fatal(page, e)
	}
	var approvals int
	if e = db.QueryRow("SELECT count(*) FROM content_review_decisions").Scan(&approvals); e != nil || approvals != 0 {
		t.Fatal("new fixture depends on old approvals", e)
	}
}
