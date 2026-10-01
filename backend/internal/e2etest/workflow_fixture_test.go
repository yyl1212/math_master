package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"testing"
)

func TestHarnessContentUsesRealAccountsAndPrivateService(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", s.ControlURL+"/scene/content", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	response, err := client.Do(req)
	if err != nil {
		t.Fatal("content control failed")
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatalf("content scene status: %d", response.StatusCode)
	}
	response, err = client.Get(s.APIURL + "/api/v1/content/drafts")
	if err != nil {
		t.Fatal("private service unavailable")
	}
	response.Body.Close()
	if response.StatusCode != 401 || len(response.Cookies()) != 0 {
		t.Fatal("content did not require real session")
	}
}

func TestContentFixtureRoles(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if err := store.Up(ctx, db, "../../../db/migrations"); err != nil {
		t.Fatal("isolated migration failed")
	}
	accounts, admin, err := fixtureAccounts(store.New(db))
	if err != nil {
		t.Fatal(err)
	}
	if err = resetWorkflow(ctx, db, accounts, admin); err != nil {
		t.Fatal(err)
	}
}
