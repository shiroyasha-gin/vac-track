package app

import (
	"vaccineshit/internal/service"
)

// ListVaccines returns the private catalog.
func (a *App) ListVaccines(includeArchived bool) ([]VaccineDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Vaccines.List(ctx, includeArchived)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]VaccineDTO, 0, len(items))
	for _, item := range items {
		out = append(out, vaccineDTO(item))
	}
	return out, nil
}

// CreateVaccine adds a vaccine to the catalog.
func (a *App) CreateVaccine(input VaccineInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	created, err := a.services.Vaccines.Create(ctx, service.VaccineInput{
		Name: input.Name, Code: input.Code, Description: input.Description,
		Category: input.Category, Unit: input.Unit,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تمت إضافة اللقاح.")
	result.EntityID = created.ID
	return result
}

// UpdateVaccine edits catalog metadata without changing history.
func (a *App) UpdateVaccine(id int64, input VaccineInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	updated, err := a.services.Vaccines.Update(ctx, id, service.VaccineInput{
		Name: input.Name, Code: input.Code, Description: input.Description,
		Category: input.Category, Unit: input.Unit,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("تم حفظ بيانات اللقاح.")
	result.EntityID = updated.ID
	return result
}

// SetVaccineArchived archives or reactivates a vaccine.
func (a *App) SetVaccineArchived(id int64, archived bool) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.authReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if _, err := a.services.Vaccines.SetArchived(ctx, id, archived); err != nil {
		return a.actionError(err)
	}
	if archived {
		return Result("تمت أرشفة اللقاح مع الاحتفاظ بسجل الاستخدام والمخزون.")
	}
	return Result("تمت إعادة تنشيط اللقاح.")
}

// DeleteVaccine permanently removes an archived vaccine with no history.
func (a *App) DeleteVaccine(id int64) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.adminReadyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if err := a.services.Vaccines.Delete(ctx, id); err != nil {
		return a.actionError(err)
	}
	return Result("تم حذف اللقاح.")
}
