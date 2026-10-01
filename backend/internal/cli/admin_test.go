package cli

import (
	"bytes"
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"strings"
	"testing"
)

const cliTestPassword = "a test passphrase with spaces "

type testInitializer struct {
	username, password string
	used               bool
	err                error
}

func (i *testInitializer) Initialize(ctx context.Context, username, password, request string) (auth.User, error) {
	if i.err != nil {
		return auth.User{}, i.err
	}
	if i.used {
		return auth.User{}, auth.ErrAlreadyInitialized
	}
	i.username = username
	i.password = password
	i.used = true
	return auth.User{ID: "10000000-0000-4000-8000-000000000001", Username: username, Roles: []auth.Role{auth.RoleLearner, auth.RoleAdmin}}, nil
}
func TestAdminCLISecretBoundary(t *testing.T) {
	for _, args := range [][]string{{"--username", "test_admin", "--password=" + cliTestPassword}, {"--username", "test_admin"}, {"--username", "test_admin", "--password-stdin", "unexpected"}, {"--username", "test_admin", "--password-stdin", "--unknown"}} {
		var out, errout bytes.Buffer
		i := &testInitializer{}
		code := RunAdminInit(context.Background(), args, strings.NewReader(cliTestPassword), &out, &errout, i)
		if code == 0 || i.used || strings.Contains(out.String()+errout.String(), cliTestPassword) {
			t.Fatal("unsafe argument accepted or secret exposed")
		}
	}
	for _, input := range []string{cliTestPassword + "\nextra\n", strings.Repeat("a", 515), "too short\n", cliTestPassword + "\rmore\n"} {
		var out, errout bytes.Buffer
		i := &testInitializer{}
		if RunAdminInit(context.Background(), []string{"--username", "test_admin", "--password-stdin"}, strings.NewReader(input), &out, &errout, i) == 0 || i.used {
			t.Fatal("unsafe stdin accepted")
		}
	}
	for _, ending := range []string{"", "\n", "\r\n"} {
		var out, errout bytes.Buffer
		i := &testInitializer{}
		if RunAdminInit(context.Background(), []string{"--username", "test_admin", "--password-stdin"}, strings.NewReader(cliTestPassword+ending), &out, &errout, i) != 0 || i.password != cliTestPassword {
			t.Fatal("stdin whitespace was changed")
		}
		if strings.Contains(out.String()+errout.String(), cliTestPassword) {
			t.Fatal("password echoed")
		}
		if RunAdminInit(context.Background(), []string{"--username", "test_admin", "--password-stdin"}, strings.NewReader(cliTestPassword), &out, &errout, i) == 0 {
			t.Fatal("second administrator initialized")
		}
	}
	i := &testInitializer{err: errors.New(cliTestPassword)}
	var out, errout bytes.Buffer
	RunAdminInit(context.Background(), []string{"--username", "test_admin", "--password-stdin"}, strings.NewReader(cliTestPassword), &out, &errout, i)
	if strings.Contains(out.String()+errout.String(), cliTestPassword) {
		t.Fatal("underlying error leaked secret")
	}
}
