package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestQueryPaginationFixture(t *testing.T) {
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
	if e = resetQueryPagination(ctx, db, s, accounts, admin, root); e != nil {
		t.Fatal(e)
	}
	page, e := s.ListCurrentKnowledge(ctx, knowledgeadmin.Query{Limit: 20})
	if e != nil || page.Total != 41 || len(page.Items) != 20 {
		t.Fatal(page, e)
	}
	page, e = s.ListCurrentKnowledge(ctx, knowledgeadmin.Query{Limit: 20, Offset: 40})
	if e != nil || page.Total != 41 || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
}
