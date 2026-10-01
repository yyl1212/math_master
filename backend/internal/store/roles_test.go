package store_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

const adminReason = "Confirmed identity and independent staff role."

func (f *authFixture) admin() *auth.AdminService {
	return auth.NewAdminService(f.repo, auth.NewArgon2Hasher(rand.Reader), rand.Reader)
}
func (f *authFixture) reauth(cookies auth.Cookies, csrf string) {
	f.t.Helper()
	if _, err := f.service.Reauthenticate(f.ctx, cookies, csrf, auth.ReauthInput{Password: accountTestPassword}, "admin-proof"); err != nil {
		f.t.Fatal("admin password proof failed")
	}
}
func twoAdmins(t *testing.T) (*authFixture, *auth.AdminService, auth.User, auth.Cookies, string, auth.User, auth.Cookies, string) {
	t.Helper()
	f := newAuthFixture(t)
	admin := f.admin()
	a, err := admin.Initialize(f.ctx, "admin_a", accountTestPassword, "init")
	if err != nil {
		t.Fatal("admin init failed")
	}
	ca, sa := f.login(a.Username, accountTestPassword)
	f.reauth(ca, sa)
	b := f.register("admin_b")
	if _, err = admin.ReplaceRoles(f.ctx, ca, sa, b.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleAdmin}, Reason: adminReason}, "grant"); err != nil {
		t.Fatal("second admin grant failed")
	}
	cb, sb := f.login(b.Username, accountTestPassword)
	f.reauth(cb, sb)
	return f, admin, a, ca, sa, b, cb, sb
}
func holdAdminLock(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal("lock transaction failed")
	}
	t.Cleanup(func() { tx.Rollback() })
	if _, err = tx.Exec("SELECT pg_advisory_xact_lock(1296127049)"); err != nil {
		t.Fatal("test admin lock failed")
	}
	return tx
}
func waitForAdminLockWaiters(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT count(*) FROM pg_locks WHERE locktype='advisory' AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND classid=0 AND objid=1296127049 AND objsubid=1 AND NOT granted").Scan(&count); err != nil {
			t.Fatal("waiter observation failed")
		}
		if count >= n {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("expected admin lock waiters not observed")
		case <-ticker.C:
		}
	}
}
func TestAdminInitializeOnce(t *testing.T) {
	f := newAuthFixture(t)
	admin := f.admin()
	lock := holdAdminLock(t, f.db)
	results := make(chan error, 2)
	for _, name := range []string{"init_one", "init_two"} {
		go func(name string) {
			_, err := admin.Initialize(f.ctx, name, accountTestPassword, "init-race")
			results <- err
		}(name)
	}
	waitForAdminLockWaiters(t, f.db, 2)
	if lock.Commit() != nil {
		t.Fatal("lock release failed")
	}
	success, already := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrAlreadyInitialized) {
			already++
		} else {
			t.Fatal("unexpected initializer failure")
		}
	}
	if success != 1 || already != 1 || f.count("SELECT count(*) FROM auth_bootstrap") != 1 || f.count("SELECT count(*) FROM auth_users") != 1 || f.count("SELECT count(*) FROM auth_user_roles WHERE role IN ('learner','admin')") != 2 {
		t.Fatal("concurrent init left wrong state")
	}
	t.Run("auditRollback", func(t *testing.T) {
		f := newAuthFixture(t)
		f.exec("CREATE FUNCTION fail_init_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$")
		f.exec("CREATE TRIGGER fail_init_audit BEFORE INSERT ON auth_audit_events FOR EACH ROW EXECUTE FUNCTION fail_init_audit()")
		if _, err := f.admin().Initialize(f.ctx, "no_partial", accountTestPassword, "init"); !errors.Is(err, auth.ErrUnavailable) {
			t.Fatal("init audit failure hidden")
		}
		if f.count("SELECT count(*) FROM auth_bootstrap") != 0 || f.count("SELECT count(*) FROM auth_users") != 0 {
			t.Fatal("partial initialized account")
		}
	})
}
func TestLastAdminConcurrentDemotion(t *testing.T) {
	f, admin, a, ca, sa, b, cb, sb := twoAdmins(t)
	lock := holdAdminLock(t, f.db)
	results := make(chan error, 2)
	for _, entry := range []struct {
		user    auth.User
		cookies auth.Cookies
		csrf    string
	}{{a, ca, sa}, {b, cb, sb}} {
		go func(entry struct {
			user    auth.User
			cookies auth.Cookies
			csrf    string
		}) { delta, err := admin.ReplaceRoles(f.ctx, entry.cookies, entry.csrf, entry.user.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "self-demote"); if err == nil && !delta.ClearSession {
			results <- errors.New("self cookie not cleared")
			return
		}; results <- err }(entry)
	}
	waitForAdminLockWaiters(t, f.db, 2)
	if lock.Commit() != nil {
		t.Fatal("lock release failed")
	}
	success, last := 0, 0
	for i := 0; i < 2; i++ {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, auth.ErrLastAdminRequired) {
			last++
		} else {
			t.Fatal("unexpected concurrent self demotion result")
		}
	}
	if success != 1 || last != 1 || f.count("SELECT count(*) FROM auth_user_roles WHERE role='admin'") != 1 {
		t.Fatal("last administrator protection failed")
	}
}

type adminWriteGate struct {
	auth.AdminRepository
	entered chan struct{}
	release chan struct{}
}

func (r adminWriteGate) ReplaceAccountRoles(ctx context.Context, proof auth.SessionProof, target string, input auth.RolesInput, request string) (auth.RoleMutation, error) {
	r.entered <- struct{}{}
	<-r.release
	return r.AdminRepository.ReplaceAccountRoles(ctx, proof, target, input, request)
}
func TestAdminRevocationRacesWrite(t *testing.T) {
	for _, revokeFirst := range []bool{true, false} {
		name := "writeCommitsFirst"
		if revokeFirst {
			name = "revokeCommitsFirst"
		}
		t.Run(name, func(t *testing.T) {
			f, admin, _, ca, sa, b, cb, sb := twoAdmins(t)
			target := f.register("race_target")
			gate := adminWriteGate{f.repo, make(chan struct{}, 1), make(chan struct{})}
			gated := auth.NewAdminService(gate, auth.NewArgon2Hasher(rand.Reader), rand.Reader)
			result := make(chan error, 1)
			if revokeFirst {
				go func() {
					_, err := gated.ReplaceRoles(f.ctx, cb, sb, target.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleEditor}, Reason: adminReason}, "racing-write")
					result <- err
				}()
			} else {
				go func() {
					_, err := gated.ReplaceRoles(f.ctx, ca, sa, b.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "racing-revoke")
					result <- err
				}()
			}
			select {
			case <-gate.entered:
			case <-time.After(2 * time.Second):
				t.Fatal("admin precheck gate not entered")
			}
			if revokeFirst {
				if _, err := admin.ReplaceRoles(f.ctx, ca, sa, b.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "racing-revoke"); err != nil {
					t.Fatal("revocation failed")
				}
			} else {
				if _, err := admin.ReplaceRoles(f.ctx, cb, sb, target.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleEditor}, Reason: adminReason}, "racing-write"); err != nil {
					t.Fatal("authorized write failed")
				}
			}
			close(gate.release)
			err := <-result
			if revokeFirst {
				if !errors.Is(err, auth.ErrAuthenticationRequired) || f.count("SELECT count(*) FROM auth_user_roles WHERE user_id=$1 AND role='editor'", target.ID) != 0 {
					t.Fatal("revoked precheck authorized write")
				}
			} else {
				if err != nil || f.count("SELECT count(*) FROM auth_audit_events w JOIN auth_audit_events r ON w.request_id='racing-write' AND r.request_id='racing-revoke' AND w.id<r.id") != 1 {
					t.Fatal("write before revocation missing audit order")
				}
			}
			if _, err = admin.ListUsers(f.ctx, cb, auth.UserQuery{}); !errors.Is(err, auth.ErrAuthenticationRequired) {
				t.Fatal("revoked administrator read allowed")
			}
		})
	}
	t.Run("contendedTransactions", func(t *testing.T) {
		f, admin, _, ca, sa, b, cb, sb := twoAdmins(t)
		target := f.register("contended_target")
		lock := holdAdminLock(t, f.db)
		writeResult := make(chan error, 1)
		revokeResult := make(chan error, 1)
		go func() {
			_, err := admin.ReplaceRoles(f.ctx, ca, sa, b.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: adminReason}, "contended-revoke")
			revokeResult <- err
		}()
		go func() {
			_, err := admin.ReplaceRoles(f.ctx, cb, sb, target.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleEditor}, Reason: adminReason}, "contended-write")
			writeResult <- err
		}()
		waitForAdminLockWaiters(t, f.db, 2)
		if lock.Commit() != nil {
			t.Fatal("lock release failed")
		}
		if <-revokeResult != nil {
			t.Fatal("valid revocation failed")
		}
		err := <-writeResult
		if err == nil {
			if f.count("SELECT count(*) FROM auth_audit_events w JOIN auth_audit_events r ON w.request_id='contended-write' AND r.request_id='contended-revoke' AND w.id<r.id") != 1 {
				t.Fatal("post-revocation write committed")
			}
		} else if !errors.Is(err, auth.ErrAuthenticationRequired) {
			t.Fatal("unexpected contended write rejection")
		}
	})
}
func TestResetRequiresOwnershipAndReauth(t *testing.T) {
	f := newAuthFixture(t)
	admin := f.admin()
	actor, err := admin.Initialize(f.ctx, "reset_admin", accountTestPassword, "init")
	if err != nil {
		t.Fatal("init failed")
	}
	cookies, csrf := f.login(actor.Username, accountTestPassword)
	target, old, _ := f.signup("reset_target")
	input := auth.ResetInput{TemporaryPassword: changedTestPassword, Reason: adminReason, OwnershipNote: "Identity verified in person using trusted records."}
	if _, err = admin.ResetPassword(f.ctx, cookies, csrf, target.ID, input, "reset"); !errors.Is(err, auth.ErrReauthRequired) {
		t.Fatal("login implicitly allowed reset")
	}
	f.reauth(cookies, csrf)
	h := tokenHash(t, cookies.Session)
	f.exec("UPDATE auth_sessions SET reauthenticated_at=clock_timestamp()-interval '5 minutes' WHERE token_hash=$1", h[:])
	if _, err = admin.ResetPassword(f.ctx, cookies, csrf, target.ID, input, "expired"); !errors.Is(err, auth.ErrReauthRequired) {
		t.Fatal("expired proof accepted")
	}
	f.reauth(cookies, csrf)
	missing := input
	missing.OwnershipNote = ""
	if _, err = admin.ResetPassword(f.ctx, cookies, csrf, target.ID, missing, "missing"); !errors.Is(err, auth.ErrInvalidInput) {
		t.Fatal("missing ownership accepted")
	}
	if _, err = admin.ResetPassword(f.ctx, cookies, csrf, target.ID, input, "reset"); err != nil {
		t.Fatal("reset failed")
	}
	if f.count("SELECT count(*) FROM auth_users WHERE id=$1 AND credential_version=2 AND must_change_password", target.ID) != 1 {
		t.Fatal("reset state failed")
	}
	if user, err := f.service.Session(f.ctx, old); err != nil || user != nil {
		t.Fatal("old reset session valid")
	}
	before := f.count("SELECT credential_version FROM auth_users WHERE id=$1", actor.ID)
	delta, err := admin.ReplaceRoles(f.ctx, cookies, csrf, actor.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleAdmin, auth.RoleLearner, auth.RoleAdmin}, Reason: adminReason}, "same-roles")
	if err != nil || delta.ClearSession || f.count("SELECT credential_version FROM auth_users WHERE id=$1", actor.ID) != before {
		t.Fatal("same roles revoked identity")
	}
	if f.count("SELECT count(*) FROM auth_audit_events WHERE request_id='same-roles'") != 1 {
		t.Fatal("same role set not audited")
	}
	delta, err = admin.ResetPassword(f.ctx, cookies, csrf, actor.ID, input, "self-reset")
	if err != nil || !delta.ClearSession {
		t.Fatal("self reset cookie not cleared")
	}
}
func TestRestrictedResetSession(t *testing.T) {
	f, admin, a, ca, sa, _, _, _ := twoAdmins(t)
	input := auth.ResetInput{TemporaryPassword: changedTestPassword, Reason: adminReason, OwnershipNote: "Identity verified with trusted records."}
	if _, err := admin.ResetPassword(f.ctx, ca, sa, a.ID, input, "reset-self"); err != nil {
		t.Fatal("self reset failed")
	}
	cookies, csrf := f.login(a.Username, changedTestPassword)
	if view, _, err := f.service.Context(f.ctx, cookies); err != nil || view.User == nil || !view.User.MustChangePassword {
		t.Fatal("restricted context unavailable")
	}
	if _, err := f.service.Reauthenticate(f.ctx, cookies, csrf, auth.ReauthInput{Password: changedTestPassword}, "restricted"); !errors.Is(err, auth.ErrPasswordChangeRequired) {
		t.Fatal("restricted reauth allowed")
	}
	if _, err := admin.ListUsers(f.ctx, cookies, auth.UserQuery{}); !errors.Is(err, auth.ErrPasswordChangeRequired) {
		t.Fatal("restricted admin allowed")
	}
	if _, err := f.service.ChangePassword(f.ctx, cookies, csrf, auth.PasswordInput{CurrentPassword: changedTestPassword, NewPassword: accountTestPassword}, "required-change"); err != nil {
		t.Fatal("required change denied")
	}
	if user, err := f.service.Session(f.ctx, cookies); err != nil || user != nil {
		t.Fatal("changed session did not require login")
	}
	f.login(a.Username, accountTestPassword)
}
func TestLiteralUserSearch(t *testing.T) {
	f := newAuthFixture(t)
	admin := f.admin()
	a, err := admin.Initialize(f.ctx, "search_admin", accountTestPassword, "init")
	if err != nil {
		t.Fatal("init failed")
	}
	ca, sa := f.login(a.Username, accountTestPassword)
	f.reauth(ca, sa)
	for _, name := range []string{"alpha_one", "alpha_two", "beta_user"} {
		f.register(name)
	}
	for _, tt := range []struct {
		q    string
		want int
	}{{"%", 0}, {"_", 4}, {"'", 0}, {"ALPHA", 2}} {
		page, err := admin.ListUsers(f.ctx, ca, auth.UserQuery{Q: tt.q})
		if err != nil || page.Total != tt.want || len(page.Items) != tt.want {
			t.Fatal("literal search failed")
		}
	}
	page, err := admin.ListUsers(f.ctx, ca, auth.UserQuery{Limit: 1, Offset: 1})
	if err != nil || page.Total != 4 || len(page.Items) != 1 || page.Items[0].Username != "alpha_two" {
		t.Fatal("stable pagination failed")
	}
	for _, roles := range [][]auth.Role{{auth.RoleLearner}, {auth.RoleLearner, auth.RoleEditor}, {auth.RoleLearner, auth.RoleReviewer}} {
		u := f.register("role_" + string(roles[len(roles)-1]))
		if _, err = admin.ReplaceRoles(f.ctx, ca, sa, u.ID, auth.RolesInput{Roles: roles, Reason: adminReason}, "grant"); err != nil {
			t.Fatal("role grant failed")
		}
		c, _ := f.login(u.Username, accountTestPassword)
		if _, err = admin.ListUsers(f.ctx, c, auth.UserQuery{}); !errors.Is(err, auth.ErrForbidden) {
			t.Fatal("nonadmin read authorized")
		}
	}
	if _, err = admin.ListUsers(f.ctx, auth.Cookies{}, auth.UserQuery{}); !errors.Is(err, auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous read authorized")
	}
}
