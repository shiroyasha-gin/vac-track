package service

import (
	"strings"
	"testing"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
)

func TestEnsureDefaultAdminSeedsOnce(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin twice: %v", err)
	}

	var count int
	var hash string
	if err := f.db.DB().QueryRowContext(f.ctx, "SELECT COUNT(*), MAX(password_hash) FROM users").Scan(&count, &hash); err != nil {
		t.Fatalf("read users: %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1", count)
	}
	if hash == DefaultAdminPassword || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("password was not stored as a bcrypt hash")
	}
}

func TestAuthenticateDefaultAdmin(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}

	session, err := f.svc.Users.Authenticate(f.ctx, " Admin ", DefaultAdminPassword)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if session.Username != DefaultAdminUsername || session.DisplayName == "" || session.Role != "admin" {
		t.Fatalf("session = %+v", session)
	}
	if f.countAudit("auth.login_succeeded") != 1 {
		t.Fatalf("login audit event was not recorded")
	}

	for _, attempt := range []struct{ username, password string }{
		{DefaultAdminUsername, "wrong-password"},
		{"missing", DefaultAdminPassword},
		{"", DefaultAdminPassword},
		{DefaultAdminUsername, ""},
	} {
		if _, err := f.svc.Users.Authenticate(f.ctx, attempt.username, attempt.password); err == nil {
			t.Fatalf("authenticate(%q) succeeded, want failure", attempt.username)
		} else if !domain.IsCode(err, domain.CodeUnauthenticated) {
			t.Fatalf("authenticate(%q) error = %v, want unauthenticated", attempt.username, err)
		}
	}
}

func TestAuthenticateExistingUserIsNotReseeded(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("authenticate before seeding = %v, want unauthenticated", err)
	}
	if _, err := f.db.DB().ExecContext(f.ctx,
		`INSERT INTO users (username, password_hash, display_name, role, created_at, updated_at)
		 VALUES ('operator', '$2a$10$J0tLnQ7n07X9gVEL2miHMuSrzzrcf4dluBjimd08J2MGDC4q3PDE.', 'Operator', 'staff', '2026-11-15T12:00:00Z', '2026-11-15T12:00:00Z')`,
	); err != nil {
		t.Fatalf("insert operator: %v", err)
	}
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("ensure default admin: %v", err)
	}
	var count int
	if err := f.db.DB().QueryRowContext(f.ctx, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1 (no reseed)", count)
	}
}

func TestChangePassword(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	session, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if !session.UsingDefaultPassword {
		t.Fatal("seeded account was not flagged as using the default password")
	}

	updated, err := f.svc.Users.ChangePassword(f.ctx, session.ID, DefaultAdminPassword, "clinic-secret-2026")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if updated.UsingDefaultPassword {
		t.Fatal("changed account is still flagged as using the default password")
	}
	if f.countAudit("auth.password_changed") != 1 {
		t.Fatal("password change audit event was not recorded")
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("old password still works: %v", err)
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, "clinic-secret-2026"); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
}

func TestChangePasswordValidation(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	session, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	cases := []struct {
		name    string
		current string
		next    string
		code    domain.ErrorCode
	}{
		{"wrong current", "not-the-password", "clinic-secret-2026", domain.CodeForbidden},
		{"short next", DefaultAdminPassword, "short", domain.CodeInvalidInput},
		{"short next runes", DefaultAdminPassword, "كلمةسري", domain.CodeInvalidInput},
		{"same password", DefaultAdminPassword, DefaultAdminPassword, domain.CodeInvalidInput},
		{"empty current", "", "clinic-secret-2026", domain.CodeInvalidInput},
	}
	for _, tc := range cases {
		if _, err := f.svc.Users.ChangePassword(f.ctx, session.ID, tc.current, tc.next); !domain.IsCode(err, tc.code) {
			t.Fatalf("%s: error = %v, want %s", tc.name, err, tc.code)
		}
	}
	if f.countAudit("auth.password_changed") != 0 {
		t.Fatal("rejected password changes wrote an audit event")
	}
}

func TestCreateAndListUsers(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	created, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "nurse", DisplayName: "ممرضة", Password: "clinic-secret-2026", Role: "staff",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Role != "staff" || !created.Active {
		t.Fatalf("created account = %+v", created)
	}
	if f.countAudit("user.created") != 1 {
		t.Fatal("user creation audit event missing")
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, "nurse", "clinic-secret-2026"); err != nil {
		t.Fatalf("authenticate created user: %v", err)
	}
	accounts, err := f.svc.Users.List(f.ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("user count = %d, want 2", len(accounts))
	}
	if _, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "NURSE", Password: "clinic-secret-2026", Role: "staff",
	}); !domain.IsCode(err, domain.CodeConflict) {
		t.Fatalf("duplicate username = %v, want conflict", err)
	}
	if _, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "nurse2", Password: "short", Role: "staff",
	}); !domain.IsCode(err, domain.CodeInvalidInput) {
		t.Fatalf("short password = %v, want invalid input", err)
	}
	if _, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "nurse3", Password: "clinic-secret-2026", Role: "owner",
	}); !domain.IsCode(err, domain.CodeInvalidInput) {
		t.Fatalf("invalid role = %v, want invalid input", err)
	}
}

func TestLastAdminCannotBeRemoved(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	admin, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword)
	if err != nil {
		t.Fatalf("authenticate admin: %v", err)
	}
	if _, err := f.svc.Users.Update(f.ctx, admin.ID, admin.ID, AccountUpdate{
		DisplayName: admin.DisplayName, Role: "staff", Active: true,
	}); !domain.IsCode(err, domain.CodeConflict) {
		t.Fatalf("demote last admin = %v, want conflict", err)
	}
	if _, err := f.svc.Users.SetActive(f.ctx, admin.ID, admin.ID, false); !domain.IsCode(err, domain.CodeInvalidInput) {
		t.Fatalf("deactivate self = %v, want invalid input", err)
	}

	other, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "second", Password: "clinic-secret-2026", Role: "admin",
	})
	if err != nil {
		t.Fatalf("create second admin: %v", err)
	}
	if _, err := f.svc.Users.Update(f.ctx, admin.ID, admin.ID, AccountUpdate{
		DisplayName: admin.DisplayName, Role: "staff", Active: true,
	}); err != nil {
		t.Fatalf("demote with another admin present: %v", err)
	}
	if _, err := f.svc.Users.SetActive(f.ctx, other.ID, other.ID, false); !domain.IsCode(err, domain.CodeInvalidInput) {
		t.Fatalf("deactivate self after demotion = %v, want invalid input", err)
	}
}

func TestResetPasswordAndDeactivate(t *testing.T) {
	f := newFixture(t)
	if err := f.svc.Users.EnsureDefaultAdmin(f.ctx); err != nil {
		t.Fatalf("seed default admin: %v", err)
	}
	admin, err := f.svc.Users.Authenticate(f.ctx, DefaultAdminUsername, DefaultAdminPassword)
	if err != nil {
		t.Fatalf("authenticate admin: %v", err)
	}
	user, err := f.svc.Users.Create(f.ctx, AccountInput{
		Username: "nurse", Password: "clinic-secret-2026", Role: "staff",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := f.svc.Users.ResetPassword(f.ctx, user.ID, "another-secret-2026"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if f.countAudit("user.password_reset") != 1 {
		t.Fatal("password reset audit event missing")
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, "nurse", "another-secret-2026"); err != nil {
		t.Fatalf("authenticate reset password: %v", err)
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, "nurse", "clinic-secret-2026"); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("old password still works: %v", err)
	}
	if _, err := f.svc.Users.SetActive(f.ctx, admin.ID, user.ID, false); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}
	if f.countAudit("user.deactivated") != 1 {
		t.Fatal("deactivation audit event missing")
	}
	if _, err := f.svc.Users.Authenticate(f.ctx, "nurse", "another-secret-2026"); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("deactivated user can still log in: %v", err)
	}
	if _, err := f.svc.Users.GetSessionUser(f.ctx, user.ID); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("deactivated session = %v, want unauthenticated", err)
	}
}

func TestAuditRecordsActingUser(t *testing.T) {
	f := newFixture(t)
	ctx := audit.WithActor(f.ctx, audit.Actor{ID: 42, Username: "operator"})
	if _, err := f.svc.Vaccines.Create(ctx, VaccineInput{Name: "MMR", Unit: "dose"}); err != nil {
		t.Fatalf("create vaccine: %v", err)
	}
	var metadata string
	if err := f.db.DB().QueryRowContext(f.ctx,
		`SELECT metadata FROM audit_events WHERE event_type = 'vaccine.created'`).Scan(&metadata); err != nil {
		t.Fatalf("read audit metadata: %v", err)
	}
	if !strings.Contains(metadata, `"actor_id":42`) || !strings.Contains(metadata, `"actor_username":"operator"`) {
		t.Fatalf("audit metadata missing actor: %s", metadata)
	}
}
