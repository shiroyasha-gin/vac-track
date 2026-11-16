package app

import (
	"fmt"
	"time"

	"vaccineshit/internal/domain"
	"vaccineshit/internal/service"
)

// AuthUserDTO is the safe authenticated-user view returned to the private UI.
// Password hashes never cross the binding boundary.
type AuthUserDTO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
}

// AuthStateDTO describes the current local session.
type AuthStateDTO struct {
	Authenticated        bool         `json:"authenticated"`
	UsingDefaultPassword bool         `json:"usingDefaultPassword"`
	User                 *AuthUserDTO `json:"user,omitempty"`
}

func authState(user *service.SessionUser) AuthStateDTO {
	if user == nil {
		return AuthStateDTO{}
	}
	return AuthStateDTO{
		Authenticated:        true,
		UsingDefaultPassword: user.UsingDefaultPassword,
		User: &AuthUserDTO{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
		},
	}
}

// LoginThrottle limits repeated failed logins. The app is local, but offline
// password guessing still needs a cost.
const (
	maxLoginFailures = 5
	loginBlockBase   = 30 * time.Second
	loginBlockMax    = 5 * time.Minute
)

// Login verifies local credentials and starts an in-memory session. The seeded
// default account exists only until the operator creates or replaces accounts.
func (a *App) Login(username, password string) (AuthStateDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return AuthStateDTO{}, a.userError(err)
	}
	now := a.now()
	if now.Before(a.loginBlockedUntil) {
		remaining := a.loginBlockedUntil.Sub(now).Round(time.Second)
		return AuthStateDTO{}, domain.NewError(domain.CodeForbidden, "",
			fmt.Sprintf("تم إيقاف تسجيل الدخول مؤقتاً بعد محاولات فاشلة. حاول بعد %s.", humanDuration(remaining)))
	}
	ctx, cancel := a.opContext()
	defer cancel()
	user, err := a.services.Users.Authenticate(ctx, username, password)
	if err != nil {
		a.recordLoginFailureLocked(now)
		return AuthStateDTO{}, a.userError(err)
	}
	a.loginFailures = 0
	a.loginBlockedUntil = time.Time{}
	a.session = &user
	return authState(a.session), nil
}

func (a *App) recordLoginFailureLocked(now time.Time) {
	a.loginFailures++
	if a.loginFailures < maxLoginFailures {
		return
	}
	block := loginBlockBase
	for i := maxLoginFailures; i < a.loginFailures && block < loginBlockMax; i++ {
		block *= 2
	}
	if block > loginBlockMax {
		block = loginBlockMax
	}
	a.loginBlockedUntil = now.Add(block)
	a.logf("login temporarily blocked", "failures", a.loginFailures, "until", a.loginBlockedUntil)
}

func humanDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d ثانية", int(d.Seconds()))
	}
	return fmt.Sprintf("%d دقيقة", int(d.Minutes()))
}

// Logout clears the in-memory session. All data operations then require login
// again.
func (a *App) Logout() AuthStateDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.session = nil
	return AuthStateDTO{}
}

// AuthStatus reports whether the current process holds an authenticated
// session. Sessions are intentionally memory-only and end with the app.
func (a *App) AuthStatus() AuthStateDTO {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session != nil {
		if err := a.authReadyLocked(); err != nil {
			a.session = nil
		}
	}
	return authState(a.session)
}

// ChangePassword verifies the current password and replaces it. The returned
// state reflects that the account no longer uses the seeded default password.
func (a *App) ChangePassword(current, next string) (AuthStateDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return AuthStateDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	updated, err := a.services.Users.ChangePassword(ctx, a.session.ID, current, next)
	if err != nil {
		return AuthStateDTO{}, a.userError(err)
	}
	a.session = &updated
	return authState(a.session), nil
}

// authReadyLocked is readyLocked plus the login gate used by every data
// operation. The private UI hides the app before login; this check also blocks a
// caller that invokes bindings directly. It re-reads the account so a
// deactivation or role change takes effect without restarting the app.
func (a *App) authReadyLocked() error {
	if err := a.readyLocked(); err != nil {
		return err
	}
	if a.session == nil {
		return domain.NewError(domain.CodeUnauthenticated, "", "يجب تسجيل الدخول للمتابعة.")
	}
	ctx, cancel := a.opContext()
	defer cancel()
	fresh, err := a.services.Users.GetSessionUser(ctx, a.session.ID)
	if err != nil {
		if domain.IsCode(err, domain.CodeUnauthenticated) {
			a.session = nil
			return domain.NewError(domain.CodeUnauthenticated, "", "انتهت الجلسة أو تم تعطيل الحساب. سجّل الدخول من جديد.")
		}
		return err
	}
	fresh.UsingDefaultPassword = a.session.UsingDefaultPassword
	a.session = &fresh
	return nil
}

// adminReadyLocked requires a signed-in admin for account management and other
// station-wide operations.
func (a *App) adminReadyLocked() error {
	if err := a.authReadyLocked(); err != nil {
		return err
	}
	if a.session == nil || a.session.Role != "admin" {
		return domain.NewError(domain.CodeForbidden, "", "هذه العملية متاحة لمسؤول النظام فقط.")
	}
	return nil
}
