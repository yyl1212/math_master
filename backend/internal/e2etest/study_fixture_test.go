package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestStudyUnclassifiedRealFixture(t *testing.T) {
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
	normal, e := loadFixture(root, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetTopicStudy(ctx, db, s, accounts, admin, root, normal, true); e != nil {
		t.Fatal("unclassified fixture", e)
	}
}
func TestStudyControlPublicationRealWorkflow(t *testing.T) {
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
	normal, e := loadFixture(root, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetTopicStudy(ctx, db, s, accounts, admin, root, normal); e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"move", "upgrade", "withdraw"} {
		if e = changeStudyPublication(ctx, db, s, accounts, op); e != nil {
			t.Fatal("study publication", op, e)
		}
	}
}
