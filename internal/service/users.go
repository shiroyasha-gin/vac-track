package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"vaccineshit/internal/audit"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/store/generated"
)

// DefaultAdminUsername and DefaultAdminPassword seed the very first local
// account. The password is hashed with bcrypt and never stored in clear.
const (
	DefaultAdminUsername    = "admin"
	DefaultAdminPassword    = "admin123"
	DefaultAdminDisplayName = "مسؤول النظام"
)

// dummyPasswordHash keeps unknown-user checks as slow as a real bcrypt check,
// so response time does not reveal whether a username exists.
const dummyPasswordHash = "$2a$10$J0tLnQ7n07X9gVEL2miHMuSrzzrcf4dluBjimd08J2MGDC4q3PDE."

// UserService authenticates local operators and seeds the first account.
type UserService struct {
	base
}

// SessionUser is the safe, password-free view of an authenticated user.
type SessionUser struct {
	ID                   int64
	Username             string
	DisplayName          string
	Role                 string
	Active               bool
	UsingDefaultPassword bool
}

// Account is the admin-facing user row. It never includes a password hash.
type Account struct {
	ID          int64
	Username    string
	DisplayName string
	Role        string
	Active      bool
	CreatedAt   string
	UpdatedAt   string
}

// AccountInput creates a new local account.
type AccountInput struct {
	Username    string
	DisplayName string
	Password    string
	Role        string
}

// AccountUpdate changes an existing account's profile and access state.
type AccountUpdate struct {
	DisplayName string
	Role        string
	Active      bool
}

func account(user generated.User) Account {
	display := strings.TrimSpace(user.DisplayName)
	if display == "" {
		display = user.Username
	}
	return Account{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: display,
		Role:        user.Role,
		Active:      user.IsActive == 1,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

func sessionUser(user generated.User) SessionUser {
	display := strings.TrimSpace(user.DisplayName)
	if display == "" {
		display = user.Username
	}
	return SessionUser{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: display,
		Role:        user.Role,
		Active:      user.IsActive == 1,
		UsingDefaultPassword: bcrypt.CompareHashAndPassword(
			[]byte(user.PasswordHash), []byte(DefaultAdminPassword),
		) == nil,
	}
}

// EnsureDefaultAdmin creates the seeded operator only when the users table is
// empty. Existing installations keep their accounts untouched.
func (s *UserService) EnsureDefaultAdmin(ctx context.Context) error {
	count, err := s.store.Queries().CountUsers(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DefaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := s.timestamp()
	if _, err := s.store.Queries().CreateUser(ctx, generated.CreateUserParams{
		Username:     DefaultAdminUsername,
		PasswordHash: string(hash),
		DisplayName:  DefaultAdminDisplayName,
		Role:         "admin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		// Another instance may have seeded the same empty database first.
		if count, countErr := s.store.Queries().CountUsers(ctx); countErr == nil && count > 0 {
			return nil
		}
		return err
	}
	return nil
}

// Authenticate verifies a username and password. Every failure returns the same
// message so the UI cannot be used to enumerate accounts.
func (s *UserService) Authenticate(ctx context.Context, username, password string) (SessionUser, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return SessionUser{}, invalidCredentials()
	}
	user, err := s.store.Queries().GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyPasswordHash), []byte(password))
			return SessionUser{}, invalidCredentials()
		}
		return SessionUser{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return SessionUser{}, invalidCredentials()
	}
	if user.IsActive != 1 {
		return SessionUser{}, invalidCredentials()
	}
	session := sessionUser(user)
	if err := s.store.WithTx(ctx, func(q generated.Querier) error {
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventLoginSucceeded,
			EntityType: "user",
			EntityID:   idString(user.ID),
			Metadata:   map[string]any{"username": user.Username},
		})
	}); err != nil {
		return SessionUser{}, err
	}
	return session, nil
}

func invalidCredentials() error {
	return domain.NewError(domain.CodeUnauthenticated, "", "اسم المستخدم أو كلمة المرور غير صحيحة.")
}

// ChangePassword replaces the signed-in operator's password after verifying
// the current one. The new password is hashed before it touches the database.
func (s *UserService) ChangePassword(ctx context.Context, userID int64, current, next string) (SessionUser, error) {
	if userID <= 0 {
		return SessionUser{}, domain.NewError(domain.CodeUnauthenticated, "", "يجب تسجيل الدخول للمتابعة.")
	}
	if current == "" {
		return SessionUser{}, domain.NewError(domain.CodeInvalidInput, "currentPassword", "أدخل كلمة المرور الحالية.")
	}
	if utf8.RuneCountInString(next) < 8 {
		return SessionUser{}, domain.NewError(domain.CodeInvalidInput, "newPassword", "يجب أن تتكون كلمة المرور الجديدة من 8 أحرف على الأقل.")
	}
	if len([]byte(next)) > 72 {
		return SessionUser{}, domain.NewError(domain.CodeInvalidInput, "newPassword", "كلمة المرور طويلة جداً. استخدم 72 بايت أو أقل.")
	}
	if next == current {
		return SessionUser{}, domain.NewError(domain.CodeInvalidInput, "newPassword", "اختر كلمة مرور مختلفة عن الحالية.")
	}
	user, err := s.store.Queries().GetUser(ctx, userID)
	if err != nil {
		return SessionUser{}, requireFound(err, "User")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)) != nil {
		return SessionUser{}, domain.NewError(domain.CodeForbidden, "currentPassword", "كلمة المرور الحالية غير صحيحة.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return SessionUser{}, err
	}
	var updated generated.User
	if err := s.store.WithTx(ctx, func(q generated.Querier) error {
		updated, err = q.UpdateUserPassword(ctx, generated.UpdateUserPasswordParams{
			PasswordHash: string(hash),
			UpdatedAt:    s.timestamp(),
			ID:           userID,
		})
		if err != nil {
			return err
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventPasswordChanged,
			EntityType: "user",
			EntityID:   idString(userID),
			Metadata:   map[string]any{"username": user.Username},
		})
	}); err != nil {
		return SessionUser{}, err
	}
	return sessionUser(updated), nil
}

// GetSessionUser reloads an account for the active session. Deactivation or
// deletion makes the session invalid without restarting the app.
func (s *UserService) GetSessionUser(ctx context.Context, id int64) (SessionUser, error) {
	if id <= 0 {
		return SessionUser{}, invalidCredentials()
	}
	user, err := s.store.Queries().GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionUser{}, invalidCredentials()
		}
		return SessionUser{}, err
	}
	if user.IsActive != 1 {
		return SessionUser{}, invalidCredentials()
	}
	display := strings.TrimSpace(user.DisplayName)
	if display == "" {
		display = user.Username
	}
	// This path runs before every data operation, so it avoids the bcrypt
	// comparison used for the login-time default-password warning. The caller
	// keeps the warning flag from the authenticated session.
	return SessionUser{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: display,
		Role:        user.Role,
		Active:      true,
	}, nil
}

// List returns every local account, newest active accounts first.
func (s *UserService) List(ctx context.Context) ([]Account, error) {
	rows, err := s.store.Queries().ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]Account, 0, len(rows))
	for _, row := range rows {
		accounts = append(accounts, account(row))
	}
	return accounts, nil
}

// Create adds a local account. The caller must be an authenticated admin.
func (s *UserService) Create(ctx context.Context, input AccountInput) (Account, error) {
	username, err := validateUsername(input.Username)
	if err != nil {
		return Account{}, err
	}
	display, err := validateDisplayName(input.DisplayName)
	if err != nil {
		return Account{}, err
	}
	role, err := validateRole(input.Role)
	if err != nil {
		return Account{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return Account{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}
	now := s.timestamp()
	var created generated.User
	if err := s.store.WithTx(ctx, func(q generated.Querier) error {
		created, err = q.CreateUser(ctx, generated.CreateUserParams{
			Username:     username,
			PasswordHash: string(hash),
			DisplayName:  display,
			Role:         role,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
		if err != nil {
			return mapPersistenceError(err, "اسم المستخدم مستخدم بالفعل.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventUserCreated,
			EntityType: "user",
			EntityID:   idString(created.ID),
			Metadata:   map[string]any{"username": created.Username, "role": created.Role},
		})
	}); err != nil {
		return Account{}, err
	}
	return account(created), nil
}

// Update changes an account's display name, role and access state. It refuses
// to remove or deactivate the last active admin and to deactivate the caller.
func (s *UserService) Update(ctx context.Context, actorID, id int64, input AccountUpdate) (Account, error) {
	display, err := validateDisplayName(input.DisplayName)
	if err != nil {
		return Account{}, err
	}
	role, err := validateRole(input.Role)
	if err != nil {
		return Account{}, err
	}
	if actorID == id && !input.Active {
		return Account{}, domainInvalid("لا يمكنك تعطيل الحساب الذي تستخدمه الآن.")
	}
	existing, err := s.store.Queries().GetUser(ctx, id)
	if err != nil {
		return Account{}, requireFound(err, "User")
	}
	if err := s.guardLastAdmin(ctx, existing, role, input.Active); err != nil {
		return Account{}, err
	}
	var updated generated.User
	if err := s.store.WithTx(ctx, func(q generated.Querier) error {
		updated, err = q.UpdateUser(ctx, generated.UpdateUserParams{
			DisplayName: display,
			Role:        role,
			IsActive:    boolInt(input.Active),
			UpdatedAt:   s.timestamp(),
			ID:          id,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر تحديث الحساب.")
		}
		eventType := audit.EventUserUpdated
		if existing.IsActive == 1 && !input.Active {
			eventType = audit.EventUserDeactivated
		} else if existing.IsActive == 0 && input.Active {
			eventType = audit.EventUserReactivated
		}
		return s.record(ctx, q, audit.Event{
			Type:       eventType,
			EntityType: "user",
			EntityID:   idString(id),
			Metadata:   map[string]any{"username": updated.Username, "role": updated.Role, "active": input.Active},
		})
	}); err != nil {
		return Account{}, err
	}
	return account(updated), nil
}

// SetActive enables or disables an account without changing its profile.
func (s *UserService) SetActive(ctx context.Context, actorID, id int64, active bool) (Account, error) {
	existing, err := s.store.Queries().GetUser(ctx, id)
	if err != nil {
		return Account{}, requireFound(err, "User")
	}
	return s.Update(ctx, actorID, id, AccountUpdate{
		DisplayName: existing.DisplayName,
		Role:        existing.Role,
		Active:      active,
	})
}

// ResetPassword sets a new password for another account without asking for the
// old one. The caller must be an authenticated admin.
func (s *UserService) ResetPassword(ctx context.Context, id int64, password string) (Account, error) {
	if err := validatePassword(password); err != nil {
		return Account{}, err
	}
	existing, err := s.store.Queries().GetUser(ctx, id)
	if err != nil {
		return Account{}, requireFound(err, "User")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Account{}, err
	}
	var updated generated.User
	if err := s.store.WithTx(ctx, func(q generated.Querier) error {
		updated, err = q.UpdateUserPassword(ctx, generated.UpdateUserPasswordParams{
			PasswordHash: string(hash),
			UpdatedAt:    s.timestamp(),
			ID:           id,
		})
		if err != nil {
			return mapPersistenceError(err, "تعذر إعادة تعيين كلمة المرور.")
		}
		return s.record(ctx, q, audit.Event{
			Type:       audit.EventUserPasswordReset,
			EntityType: "user",
			EntityID:   idString(id),
			Metadata:   map[string]any{"username": existing.Username},
		})
	}); err != nil {
		return Account{}, err
	}
	return account(updated), nil
}

func (s *UserService) guardLastAdmin(ctx context.Context, existing generated.User, nextRole string, nextActive bool) error {
	if existing.Role != "admin" || existing.IsActive != 1 {
		return nil
	}
	if nextRole == "admin" && nextActive {
		return nil
	}
	count, err := s.store.Queries().CountActiveAdmins(ctx)
	if err != nil {
		return err
	}
	if count <= 1 {
		return domainConflict("لا يمكن إزالة آخر مسؤول نشط. أضف مسؤولاً آخر أولاً.")
	}
	return nil
}

func validateUsername(value string) (string, error) {
	username := strings.TrimSpace(value)
	if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 64 {
		return "", domain.NewError(domain.CodeInvalidInput, "username", "اسم المستخدم يجب أن يكون بين 3 و64 حرفاً.")
	}
	if strings.ContainsAny(username, " \t\r\n") || strings.ContainsFunc(username, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
		return "", domain.NewError(domain.CodeInvalidInput, "username", "لا يمكن أن يحتوي اسم المستخدم على مسافات أو رموز تحكم.")
	}
	return username, nil
}

func validateDisplayName(value string) (string, error) {
	display := strings.TrimSpace(value)
	if utf8.RuneCountInString(display) > 120 {
		return "", domain.NewError(domain.CodeInvalidInput, "displayName", "الاسم المعروض طويل جداً. استخدم 120 حرفاً أو أقل.")
	}
	return display, nil
}

func validateRole(value string) (string, error) {
	role := strings.TrimSpace(value)
	if role != "admin" && role != "staff" {
		return "", domain.NewError(domain.CodeInvalidInput, "role", "اختر دوراً صالحاً: مسؤول أو موظف.")
	}
	return role, nil
}

func validatePassword(value string) error {
	if utf8.RuneCountInString(value) < 8 {
		return domain.NewError(domain.CodeInvalidInput, "password", "يجب أن تتكون كلمة المرور من 8 أحرف على الأقل.")
	}
	if len([]byte(value)) > 72 {
		return domain.NewError(domain.CodeInvalidInput, "password", "كلمة المرور طويلة جداً. استخدم 72 بايت أو أقل.")
	}
	return nil
}
