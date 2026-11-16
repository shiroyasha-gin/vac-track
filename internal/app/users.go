package app

import (
	"vaccineshit/internal/service"
)

// UserDTO is the admin-facing account row. Password hashes never leave the
// service layer.
type UserDTO struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreateUserInputDTO creates a local operator account.
type CreateUserInputDTO struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

// UpdateUserInputDTO edits an existing account.
type UpdateUserInputDTO struct {
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Active      bool   `json:"active"`
}

func userDTO(account service.Account) UserDTO {
	return UserDTO{
		ID:          account.ID,
		Username:    account.Username,
		DisplayName: account.DisplayName,
		Role:        account.Role,
		Active:      account.Active,
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
	}
}

// Users lists every local account. Admin only.
func (a *App) Users() ([]UserDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	accounts, err := a.services.Users.List(ctx)
	if err != nil {
		return nil, a.userError(err)
	}
	users := make([]UserDTO, 0, len(accounts))
	for _, account := range accounts {
		users = append(users, userDTO(account))
	}
	return users, nil
}

// CreateUser adds an account. Admin only.
func (a *App) CreateUser(input CreateUserInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	account, err := a.services.Users.Create(ctx, service.AccountInput{
		Username:    input.Username,
		DisplayName: input.DisplayName,
		Password:    input.Password,
		Role:        input.Role,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تمت إضافة المستخدم.")
	result.EntityID = account.ID
	return result
}

// UpdateUser edits a display name, role or access state. Admin only.
func (a *App) UpdateUser(id int64, input UpdateUserInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	account, err := a.services.Users.Update(ctx, a.session.ID, id, service.AccountUpdate{
		DisplayName: input.DisplayName,
		Role:        input.Role,
		Active:      input.Active,
	})
	if err != nil {
		return a.actionError(err)
	}
	a.refreshSelfLocked(account)
	return Result("تم تحديث الحساب.")
}

// SetUserActive enables or disables an account. Admin only.
func (a *App) SetUserActive(id int64, active bool) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	account, err := a.services.Users.SetActive(ctx, a.session.ID, id, active)
	if err != nil {
		return a.actionError(err)
	}
	a.refreshSelfLocked(account)
	if active {
		return Result("تم تفعيل الحساب.")
	}
	return Result("تم تعطيل الحساب.")
}

// ResetUserPassword sets a new password for another account. Admin only.
func (a *App) ResetUserPassword(id int64, password string) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	account, err := a.services.Users.ResetPassword(ctx, id, password)
	if err != nil {
		return a.actionError(err)
	}
	if a.session != nil && a.session.ID == id {
		a.session.UsingDefaultPassword = password == service.DefaultAdminPassword
	}
	a.refreshSelfLocked(account)
	return Result("تمت إعادة تعيين كلمة المرور.")
}

// refreshSelfLocked keeps the in-memory session in step when an admin edits
// their own account.
func (a *App) refreshSelfLocked(account service.Account) {
	if a.session == nil || a.session.ID != account.ID {
		return
	}
	a.session.Username = account.Username
	a.session.DisplayName = account.DisplayName
	a.session.Role = account.Role
	a.session.Active = account.Active
}
