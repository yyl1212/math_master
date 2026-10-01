package e2etest

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yyl1212/math_master/backend/internal/testutil"
)

type testState struct {
	APIURL     string `json:"apiURL"`
	ControlURL string `json:"controlURL"`
	Token      string `json:"token"`
	Database   string `json:"database"`
}

func TestHarnessRejectsUnsafeDatabaseAndPublicBind(t *testing.T) {
	safe := "postgres://local:local@127.0.0.1/math_master_test_ci"
	for _, c := range []Config{
		{TestDatabaseURL: "postgres://local:local@127.0.0.1/math_master", APIAddr: "127.0.0.1:0", ControlAddr: "127.0.0.1:0", StateFile: filepath.Join(t.TempDir(), "runtime.local.json")},
		{TestDatabaseURL: safe, APIAddr: "0.0.0.0:8080", ControlAddr: "127.0.0.1:0", StateFile: filepath.Join(t.TempDir(), "runtime.local.json")},
		{TestDatabaseURL: safe, APIAddr: "127.0.0.1:0", ControlAddr: "0.0.0.0:8081", StateFile: filepath.Join(t.TempDir(), "runtime.local.json")},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := Run(ctx, c)
		cancel()
		if err == nil {
			t.Fatal("unsafe configuration accepted")
		}
	}
}
func startHarness(t *testing.T) (testState, context.CancelFunc, <-chan error, string) {
	t.Helper()
	c := Config{TestDatabaseURL: os.Getenv("TEST_DATABASE_URL"), APIAddr: "127.0.0.1:0", ControlAddr: "127.0.0.1:0", StateFile: filepath.Join(t.TempDir(), "runtime.local.json")}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, c) }()
	t.Cleanup(cancel)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, e := os.ReadFile(c.StateFile); e == nil {
			var s testState
			if json.Unmarshal(b, &s) == nil && s.Token != "" {
				return s, cancel, done, c.StateFile
			}
		}
		select {
		case <-done:
			t.Fatal("harness exited before readiness")
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("harness readiness timed out")
	return testState{}, cancel, done, c.StateFile
}
func TestHarnessRequiresControlToken(t *testing.T) {
	s, cancel, done, path := startHarness(t)
	info, e := os.Stat(path)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("state permissions")
	}
	client := &http.Client{Timeout: time.Second * 3}
	for _, token := range []string{"", "incorrect"} {
		req, _ := http.NewRequest("POST", s.ControlURL+"/scene/published", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		r, e := client.Do(req)
		if e != nil {
			t.Fatal("control request failed")
		}
		r.Body.Close()
		if r.StatusCode != 401 {
			t.Fatal("control accepted unknown token")
		}
	}
	r, e := client.Get(s.APIURL + "/api/v1/knowledge/equivalent-fractions")
	if e != nil {
		t.Fatal("public request failed")
	}
	r.Body.Close()
	if r.StatusCode != 404 {
		t.Fatal("unauthorized publication")
	}
	req, _ := http.NewRequest("POST", s.ControlURL+"/scene/published", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	r, e = client.Do(req)
	if e != nil {
		t.Fatal("authorized control failed")
	}
	r.Body.Close()
	if r.StatusCode != 204 {
		t.Fatal("publication scene failed")
	}
	r, e = client.Get(s.APIURL + "/api/v1/knowledge/equivalent-fractions")
	if e != nil {
		t.Fatal("published request failed")
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatal("real handler not published")
	}
	r, e = client.Get(s.APIURL + "/scene/published")
	if e != nil {
		t.Fatal("public route failed")
	}
	r.Body.Close()
	if r.StatusCode != 404 {
		t.Fatal("control leaked into public handler")
	}
	cancel()
	if e := <-done; e != nil {
		t.Fatal("cleanup failed")
	}
}
func TestHarnessCleansIsolatedDatabase(t *testing.T) {
	parent := testutil.Database(t)
	if _, e := parent.Exec("CREATE TABLE sentinel (n int); INSERT INTO sentinel VALUES(42)"); e != nil {
		t.Fatal("parent seed failed")
	}
	s, cancel, done, path := startHarness(t)
	if !strings.HasPrefix(s.Database, "math_master_test_") {
		t.Fatal("unsafe runtime name")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), "postgres://") || strings.Contains(string(b), "postgresql://") {
		t.Fatal("credentials persisted")
	}
	var exists bool
	if e := parent.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", s.Database).Scan(&exists); e != nil || !exists {
		t.Fatal("random database not created")
	}
	cancel()
	if e := <-done; e != nil {
		t.Fatal("cleanup failed")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("state not removed")
	}
	if e := parent.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)", s.Database).Scan(&exists); e != nil || exists {
		t.Fatal("random database not removed")
	}
	var n int
	if e := parent.QueryRow("SELECT n FROM sentinel").Scan(&n); e != nil || n != 42 {
		t.Fatal("unrelated database changed")
	}
}
