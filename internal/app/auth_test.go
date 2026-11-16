package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/service"
)

func TestAuthSessionLifecycle(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	if status := app.AuthStatus(); status.Authenticated {
		t.Fatalf("new app started authenticated")
	}

	if _, err := app.Login(service.DefaultAdminUsername, "wrong"); err == nil {
		t.Fatalf("login with wrong password succeeded")
	} else if !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("login error = %v", err)
	}

	state, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword)
	if err != nil {
		t.Fatalf("login seeded operator: %v", err)
	}
	if !state.Authenticated || state.User == nil || state.User.Username != service.DefaultAdminUsername {
		t.Fatalf("auth state = %+v", state)
	}

	if status := app.AuthStatus(); !status.Authenticated || status.User == nil {
		t.Fatalf("auth status after login = %+v", status)
	}

	if state := app.Logout(); state.Authenticated {
		t.Fatalf("logout kept the session alive")
	}
	if status := app.AuthStatus(); status.Authenticated {
		t.Fatalf("auth status after logout = %+v", status)
	}
}

func TestDataOperationsRequireLogin(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}

	if _, err := app.ListVaccines(false); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("ListVaccines before login = %v, want unauthenticated", err)
	}
	result := app.CreatePatient(PatientInputDTO{FullName: "Blocked"})
	if result.OK || result.Code != string(domain.CodeUnauthenticated) {
		t.Fatalf("CreatePatient before login = %+v", result)
	}

	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login: %v", err)
	}
	mustOK(t, app.CreatePatient(PatientInputDTO{FullName: "Allowed"}))
}

func TestChangePasswordUpdatesSessionAndCredentials(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}

	state, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !state.UsingDefaultPassword {
		t.Fatal("seeded login did not warn about the default password")
	}

	if _, err := app.ChangePassword("wrong", "clinic-secret-2026"); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("wrong current password = %v, want forbidden", err)
	}

	updated, err := app.ChangePassword(service.DefaultAdminPassword, "clinic-secret-2026")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if updated.UsingDefaultPassword || !updated.Authenticated {
		t.Fatalf("updated auth state = %+v", updated)
	}
	if status := app.AuthStatus(); status.UsingDefaultPassword {
		t.Fatalf("stored session still warns about default password: %+v", status)
	}

	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("old password still logs in: %v", err)
	}
	if _, err := app.Login(service.DefaultAdminUsername, "clinic-secret-2026"); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
}

func TestRestoreBackupRequiresLoginWhenDatabaseIsReady(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	result := app.RestoreBackup()
	if result.OK || result.Code != "backup_error" {
		t.Fatalf("restore before login = %+v", result)
	}
	if result.Message == "" {
		t.Fatal("restore rejection has no message")
	}
}

func TestClearPublisherTokenRequiresLogin(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	result := app.ClearPublisherToken()
	if result.OK || result.Code != string(domain.CodeUnauthenticated) {
		t.Fatalf("clear token before login = %+v", result)
	}
}

func TestStaffCannotRunAdminOperations(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login admin: %v", err)
	}
	result := app.CreateUser(CreateUserInputDTO{
		Username: "nurse", DisplayName: "ممرضة", Password: "clinic-secret-2026", Role: "staff",
	})
	if !result.OK {
		t.Fatalf("create staff: %+v", result)
	}
	app.Logout()
	if _, err := app.Login("nurse", "clinic-secret-2026"); err != nil {
		t.Fatalf("login staff: %v", err)
	}

	if _, err := app.Users(); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("staff Users() = %v, want forbidden", err)
	}
	if r := app.CreateUser(CreateUserInputDTO{Username: "x", Password: "clinic-secret-2026", Role: "staff"}); r.Code != string(domain.CodeForbidden) {
		t.Fatalf("staff CreateUser = %+v", r)
	}
	if r := app.SetLowStockThreshold(5); r.Code != string(domain.CodeForbidden) {
		t.Fatalf("staff SetLowStockThreshold = %+v", r)
	}
	if r := app.DeleteVaccine(1); r.Code != string(domain.CodeForbidden) {
		t.Fatalf("staff DeleteVaccine = %+v", r)
	}
	if _, err := app.AuditEvents(AuditFilterDTO{}); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("staff AuditEvents = %v, want forbidden", err)
	}
	if r := app.backupTo(t.TempDir() + "/backup.db"); r.OK || r.Code != "backup_error" {
		t.Fatalf("staff backup = %+v", r)
	}

	mustOK(t, app.CreatePatient(PatientInputDTO{FullName: "Staff patient"}))
}

func TestDeactivatedAccountLosesSession(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	admin, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword)
	if err != nil {
		t.Fatalf("login admin: %v", err)
	}
	created := app.CreateUser(CreateUserInputDTO{
		Username: "nurse", Password: "clinic-secret-2026", Role: "staff",
	})
	if !created.OK {
		t.Fatalf("create staff: %+v", created)
	}
	app.Logout()
	if _, err := app.Login("nurse", "clinic-secret-2026"); err != nil {
		t.Fatalf("login staff: %v", err)
	}
	if _, err := app.services.Users.SetActive(context.Background(), admin.User.ID, created.EntityID, false); err != nil {
		t.Fatalf("deactivate staff: %v", err)
	}
	if status := app.AuthStatus(); status.Authenticated {
		t.Fatalf("deactivated session still authenticated: %+v", status)
	}
	if _, err := app.ListVaccines(false); !domain.IsCode(err, domain.CodeUnauthenticated) {
		t.Fatalf("data after deactivation = %v, want unauthenticated", err)
	}
}

func TestLoginThrottleBlocksAfterRepeatedFailures(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	now := time.Now().UTC()
	app.now = func() time.Time { return now }

	for i := 0; i < maxLoginFailures; i++ {
		if _, err := app.Login(service.DefaultAdminUsername, "wrong"); !domain.IsCode(err, domain.CodeUnauthenticated) {
			t.Fatalf("failure %d = %v, want unauthenticated", i+1, err)
		}
	}
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); !domain.IsCode(err, domain.CodeForbidden) {
		t.Fatalf("correct login during block = %v, want forbidden", err)
	}
	now = now.Add(loginBlockBase + time.Second)
	if _, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword); err != nil {
		t.Fatalf("login after block: %v", err)
	}
	if app.loginFailures != 0 || !app.loginBlockedUntil.IsZero() {
		t.Fatalf("throttle state not reset: failures=%d until=%v", app.loginFailures, app.loginBlockedUntil)
	}
}

func TestMutationsRecordActingUser(t *testing.T) {
	app := newAppAt(t.TempDir(), "test")
	t.Cleanup(func() { app.Shutdown(context.Background()) })
	if app.initErr != nil {
		t.Fatalf("startup: %v", app.initErr)
	}
	state, err := app.Login(service.DefaultAdminUsername, service.DefaultAdminPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	mustOK(t, app.CreatePatient(PatientInputDTO{FullName: "Audit patient"}))
	var metadata string
	if err := app.store.DB().QueryRow(
		`SELECT metadata FROM audit_events WHERE event_type = 'patient.created'`).Scan(&metadata); err != nil {
		t.Fatalf("read audit metadata: %v", err)
	}
	wantActor := `"actor_username":"` + state.User.Username + `"`
	if !strings.Contains(metadata, `"actor_id":`) || !strings.Contains(metadata, wantActor) {
		t.Fatalf("audit metadata missing actor: %s", metadata)
	}
}
