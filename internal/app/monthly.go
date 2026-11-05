package app

import (
	"fmt"

	"vaccineshit/internal/service"
)

// MonthOverview returns the monthly table and setup state.
func (a *App) MonthOverview(month string) (MonthOverviewDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return MonthOverviewDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	overview, err := a.services.Reporting.MonthOverview(ctx, month)
	if err != nil {
		return MonthOverviewDTO{}, a.userError(err)
	}
	return monthOverviewDTO(overview), nil
}

// InitializeInventory explicitly creates a vaccine/month stock row, including
// an explicit zero balance.
func (a *App) InitializeInventory(input InitializeInventoryInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	created, err := a.services.Inventory.Initialize(ctx, service.InitializeInventoryInput{
		VaccineID: input.VaccineID, Month: input.Month,
		OpeningBalance: input.OpeningBalance, StockReceived: input.StockReceived,
		OverrideReason: input.OverrideReason,
	})
	if err != nil {
		return a.actionError(err)
	}
	result := Result("Monthly stock initialized.")
	result.EntityID = created.ID
	return result
}

// UpdateStock changes opening or received stock on an open month.
func (a *App) UpdateStock(input UpdateInventoryInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if _, err := a.services.Inventory.UpdateStock(ctx, service.UpdateInventoryInput{
		VaccineID: input.VaccineID, Month: input.Month,
		OpeningBalance: input.OpeningBalance, StockReceived: input.StockReceived,
		OverrideReason: input.OverrideReason,
	}); err != nil {
		return a.actionError(err)
	}
	return Result("Monthly stock saved.")
}

// AdjustStock applies an audited correction to received stock.
func (a *App) AdjustStock(input AdjustmentInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	if _, err := a.services.Inventory.Adjust(ctx, service.AdjustmentInput{
		VaccineID: input.VaccineID, Month: input.Month,
		Amount: input.Amount, Reason: input.Reason,
	}); err != nil {
		return a.actionError(err)
	}
	return Result("Stock adjustment recorded.")
}

// CloseMonth freezes all open inventory rows for the month.
func (a *App) CloseMonth(month string) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	count, err := a.services.Inventory.CloseMonth(ctx, month)
	if err != nil {
		return a.actionError(err)
	}
	return Result(fmt.Sprintf("Month closed. %d vaccine row(s) frozen.", count))
}

// ReopenMonth requires a reason and is audited.
func (a *App) ReopenMonth(month, reason string) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	count, err := a.services.Inventory.ReopenMonth(ctx, month, reason)
	if err != nil {
		return a.actionError(err)
	}
	return Result(fmt.Sprintf("Month reopened. %d vaccine row(s) unlocked.", count))
}

// UsageForVaccineDay returns private detail for one monthly table cell.
func (a *App) UsageForVaccineDay(vaccineID int64, month, date string) ([]UsageRowDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Usage.ForVaccineDay(ctx, vaccineID, month, date)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]UsageRowDTO, 0, len(items))
	for _, item := range items {
		out = append(out, usageRowDTO(item))
	}
	return out, nil
}

// UsageForVaccineMonth returns private detail for a vaccine/month.
func (a *App) UsageForVaccineMonth(vaccineID int64, month string) ([]UsageRowDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return nil, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	items, err := a.services.Usage.ForVaccineMonth(ctx, vaccineID, month)
	if err != nil {
		return nil, a.userError(err)
	}
	out := make([]UsageRowDTO, 0, len(items))
	for _, item := range items {
		out = append(out, usageRowDTOFromMonth(item))
	}
	return out, nil
}

// CreateUsage records one administration. It is rejected unless explicit
// monthly stock exists, the month is open and stock is sufficient.
func (a *App) CreateUsage(input CreateUsageInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	result, err := a.services.Usage.Create(ctx, service.CreateUsageInput{
		PatientID: input.PatientID, VaccineID: input.VaccineID,
		AdministeredOn: input.AdministeredOn, Quantity: input.Quantity, Note: input.Note,
	})
	if err != nil {
		return a.actionError(err)
	}
	out := Result(fmt.Sprintf("Usage recorded. Remaining stock: %d.", result.Balance))
	out.EntityID = result.Entry.ID
	return out
}

// UpdateUsage corrects an entry, requiring both inventory months to be open.
func (a *App) UpdateUsage(input UpdateUsageInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	result, err := a.services.Usage.Update(ctx, service.UpdateUsageInput{
		ID: input.ID, PatientID: input.PatientID, AdministeredOn: input.AdministeredOn,
		Quantity: input.Quantity, Note: input.Note, Reason: input.Reason,
	})
	if err != nil {
		return a.actionError(err)
	}
	out := Result(fmt.Sprintf("Usage entry corrected. Remaining stock: %d.", result.Balance))
	out.EntityID = result.Entry.ID
	return out
}

// VoidUsage cancels an entry with a required reason.
func (a *App) VoidUsage(input VoidUsageInputDTO) ActionResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return a.actionError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	result, err := a.services.Usage.Void(ctx, service.VoidUsageInput{ID: input.ID, Reason: input.Reason})
	if err != nil {
		return a.actionError(err)
	}
	out := Result(fmt.Sprintf("Usage entry voided. Remaining stock: %d.", result.Balance))
	out.EntityID = result.Entry.ID
	return out
}

// GetUsage returns one usage entry for editing.
func (a *App) GetUsage(id int64) (UsageEntryDTO, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.readyLocked(); err != nil {
		return UsageEntryDTO{}, a.userError(err)
	}
	ctx, cancel := a.opContext()
	defer cancel()
	entry, err := a.services.Usage.Get(ctx, id)
	if err != nil {
		return UsageEntryDTO{}, a.userError(err)
	}
	return usageEntryDTO(entry), nil
}
