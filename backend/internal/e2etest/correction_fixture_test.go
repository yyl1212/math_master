package e2etest

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"testing"
	"time"
)

func TestCorrectionRealFixtureAndSharedReset(t *testing.T) {
	db := testutil.Database(t)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()
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
	if e = resetLearning(ctx, db, s, accounts, admin, root, normal, LearningBasic); e != nil {
		t.Fatal(e)
	}
	if e = setupCorrection(ctx, db, s, accounts); e != nil {
		t.Fatal(e)
	}
	before, e := correctionState(ctx, db)
	if e != nil || before.AttemptID == nil || before.Cases != 0 || before.OriginalSHA == "" {
		t.Fatal(before, e)
	}
	for _, scene := range []string{"correction-withdraw-instance", "correction-publish-equivalent", "correction-approve-equivalent", "correction-run"} {
		handled, e := correctionChange(ctx, db, s, accounts, root, scene)
		if !handled || e != nil {
			t.Fatal(scene, handled, e)
		}
	}
	after, e := correctionState(ctx, db)
	if e != nil || after.ResultID == nil || after.Results < 1 || after.Grants != 1 || before.OriginalSHA != after.OriginalSHA {
		t.Fatal(after, e)
	}
	if e = resetLearning(ctx, db, s, accounts, admin, root, normal, LearningBasic); e != nil {
		t.Fatal(e)
	}
	for _, table := range correctionResetTableNames {
		var n int
		if e = db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != 0 {
			t.Fatal(table, n, e)
		}
	}
	if e = db.QueryRowContext(ctx, "SELECT count(*) FROM goose_db_version WHERE id=0 AND correction_enabled").Scan(new(int)); e != nil {
		t.Fatal("permanent marker", e)
	}
}
func TestCorrectionHarnessScenesArePrivate(t *testing.T) {
	state, _, _, _ := startHarness(t)
	client := &http.Client{Timeout: 45 * time.Second}
	for _, path := range []string{"/scene/correction", "/correction/state", "/scene/correction-run"} {
		method := "POST"
		if path == "/correction/state" {
			method = "GET"
		}
		r, e := http.NewRequest(method, state.ControlURL+path, nil)
		if e != nil {
			t.Fatal(e)
		}
		response, e := client.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		response.Body.Close()
		if response.StatusCode != 401 {
			t.Fatal("missing token", path, response.StatusCode)
		}
	}
	req, _ := http.NewRequest("POST", state.ControlURL+"/scene/correction", nil)
	req.Header.Set("Authorization", "Bearer "+state.Token)
	response, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 204 {
		t.Fatal("correction scene", response.StatusCode)
	}
	req, _ = http.NewRequest("GET", state.ControlURL+"/correction/state", nil)
	req.Header.Set("Authorization", "Bearer "+state.Token)
	response, e = client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer response.Body.Close()
	var v correctionDatabaseState
	if json.NewDecoder(response.Body).Decode(&v) != nil || response.StatusCode != 200 || v.AttemptID == nil {
		t.Fatal("correction state")
	}
}
