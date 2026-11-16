package app

import (
	"encoding/json"

	"vaccineshit/internal/backup"
	"vaccineshit/internal/domain"
	"vaccineshit/internal/publishing"
	"vaccineshit/internal/service"
	"vaccineshit/internal/store/generated"
)

// ActionResult is the uniform result of mutating operations. Errors are
// converted into stable codes and safe user-facing messages; technical details
// are written only to the local application log.
type ActionResult struct {
	OK       bool   `json:"ok"`
	Code     string `json:"code"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	EntityID int64  `json:"entityId"`
}

// Result builds a successful action result.
func Result(message string) ActionResult {
	return ActionResult{OK: true, Message: message}
}

func failedResult(code domain.ErrorCode, field, message string) ActionResult {
	return ActionResult{OK: false, Code: string(code), Field: field, Message: message}
}

// BootstrapInfo describes startup state. Ready is false when the database could
// not be opened; the UI then offers restore-from-backup instead of data entry.
type BootstrapInfo struct {
	Ready                    bool   `json:"ready"`
	FatalError               string `json:"fatalError"`
	DataDir                  string `json:"dataDir"`
	DatabasePath             string `json:"databasePath"`
	Version                  string `json:"version"`
	Today                    string `json:"today"`
	CurrentMonth             string `json:"currentMonth"`
	LowStockThreshold        int64  `json:"lowStockThreshold"`
	PublisherEndpoint        string `json:"publisherEndpoint"`
	PublisherTokenConfigured bool   `json:"publisherTokenConfigured"`
	PublisherTokenStorage    string `json:"publisherTokenStorage"`
}

// VaccineDTO is the private vaccine catalog row.
type VaccineDTO struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Unit        string `json:"unit"`
	Archived    bool   `json:"archived"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// PatientDTO is a local-only patient record. It is never published.
type PatientDTO struct {
	ID          int64  `json:"id"`
	FullName    string `json:"fullName"`
	PatientCode string `json:"patientCode"`
	Archived    bool   `json:"archived"`
	ArchivedAt  string `json:"archivedAt"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// PatientPageDTO is one bounded page of local patient records.
type PatientPageDTO struct {
	Items  []PatientDTO `json:"items"`
	Total  int64        `json:"total"`
	Limit  int64        `json:"limit"`
	Offset int64        `json:"offset"`
}

// VaccineInputDTO is the validated catalog form payload.
type VaccineInputDTO struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Unit        string `json:"unit"`
}

// PatientInputDTO is the minimum patient form payload.
type PatientInputDTO struct {
	FullName    string `json:"fullName"`
	PatientCode string `json:"patientCode"`
}

// MonthRowDTO is one vaccine row of the monthly inventory table.
type MonthRowDTO struct {
	VaccineID           int64   `json:"vaccineId"`
	VaccineName         string  `json:"vaccineName"`
	VaccineCode         string  `json:"vaccineCode"`
	Unit                string  `json:"unit"`
	Archived            bool    `json:"archived"`
	Initialized         bool    `json:"initialized"`
	Status              string  `json:"status"`
	OpeningBalance      int64   `json:"openingBalance"`
	StockReceived       int64   `json:"stockReceived"`
	UsageTotal          int64   `json:"usageTotal"`
	Remaining           int64   `json:"remaining"`
	ClosingBalance      int64   `json:"closingBalance"`
	HasClosingBalance   bool    `json:"hasClosingBalance"`
	CarryoverOverridden bool    `json:"carryoverOverridden"`
	CarryoverReason     string  `json:"carryoverReason"`
	SuggestedOpening    int64   `json:"suggestedOpening"`
	PriorMonth          string  `json:"priorMonth"`
	PriorExists         bool    `json:"priorExists"`
	PriorClosed         bool    `json:"priorClosed"`
	DailyUsage          []int64 `json:"dailyUsage"`
	LowStock            bool    `json:"lowStock"`
}

// MonthTotalsDTO aggregates a monthly table.
type MonthTotalsDTO struct {
	OpeningBalance      int64 `json:"openingBalance"`
	StockReceived       int64 `json:"stockReceived"`
	UsageTotal          int64 `json:"usageTotal"`
	Remaining           int64 `json:"remaining"`
	ActiveVaccines      int   `json:"activeVaccines"`
	InitializedVaccines int   `json:"initializedVaccines"`
	UninitializedActive int   `json:"uninitializedActive"`
	LowStockCount       int   `json:"lowStockCount"`
}

// AdjustmentDTO is one audited stock correction.
type AdjustmentDTO struct {
	ID            int64  `json:"id"`
	VaccineID     int64  `json:"vaccineId"`
	VaccineName   string `json:"vaccineName"`
	Unit          string `json:"unit"`
	Amount        int64  `json:"amount"`
	BalanceBefore int64  `json:"balanceBefore"`
	BalanceAfter  int64  `json:"balanceAfter"`
	Reason        string `json:"reason"`
	CreatedAt     string `json:"createdAt"`
}

// MonthOverviewDTO is the complete monthly table.
type MonthOverviewDTO struct {
	Month             string          `json:"month"`
	Days              int             `json:"days"`
	State             string          `json:"state"`
	Rows              []MonthRowDTO   `json:"rows"`
	Totals            MonthTotalsDTO  `json:"totals"`
	LowStockThreshold int64           `json:"lowStockThreshold"`
	Adjustments       []AdjustmentDTO `json:"adjustments"`
	GeneratedAt       string          `json:"generatedAt"`
}

// InitializeInventoryInputDTO explicitly creates monthly stock.
type InitializeInventoryInputDTO struct {
	VaccineID      int64  `json:"vaccineId"`
	Month          string `json:"month"`
	OpeningBalance int64  `json:"openingBalance"`
	StockReceived  int64  `json:"stockReceived"`
	OverrideReason string `json:"overrideReason"`
}

// UpdateInventoryInputDTO edits opening and received stock.
type UpdateInventoryInputDTO struct {
	VaccineID      int64  `json:"vaccineId"`
	Month          string `json:"month"`
	OpeningBalance int64  `json:"openingBalance"`
	StockReceived  int64  `json:"stockReceived"`
	OverrideReason string `json:"overrideReason"`
}

// AdjustmentInputDTO adds an audited received-stock correction.
type AdjustmentInputDTO struct {
	VaccineID int64  `json:"vaccineId"`
	Month     string `json:"month"`
	Amount    int64  `json:"amount"`
	Reason    string `json:"reason"`
}

// UsageEntryDTO is the editable core of one usage entry.
type UsageEntryDTO struct {
	ID             int64  `json:"id"`
	PatientID      int64  `json:"patientId"`
	VaccineID      int64  `json:"vaccineId"`
	AdministeredOn string `json:"administeredOn"`
	Month          string `json:"month"`
	Quantity       int64  `json:"quantity"`
	Note           string `json:"note"`
	Voided         bool   `json:"voided"`
	VoidReason     string `json:"voidReason"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// UsageRowDTO is private local usage detail.
type UsageRowDTO struct {
	ID              int64  `json:"id"`
	PatientID       int64  `json:"patientId"`
	VaccineID       int64  `json:"vaccineId"`
	AdministeredOn  string `json:"administeredOn"`
	Month           string `json:"month"`
	Quantity        int64  `json:"quantity"`
	Note            string `json:"note"`
	Voided          bool   `json:"voided"`
	VoidReason      string `json:"voidReason"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
	PatientName     string `json:"patientName"`
	PatientCode     string `json:"patientCode"`
	PatientArchived bool   `json:"patientArchived"`
	VaccineName     string `json:"vaccineName"`
	VaccineCode     string `json:"vaccineCode"`
	VaccineUnit     string `json:"vaccineUnit"`
}

// AppointmentItemDTO is one vaccine line in a private appointment.
type AppointmentItemDTO struct {
	ID                  int64  `json:"id"`
	VaccineID           int64  `json:"vaccineId"`
	VaccineName         string `json:"vaccineName"`
	VaccineCode         string `json:"vaccineCode"`
	VaccineUnit         string `json:"vaccineUnit"`
	VaccineArchived     bool   `json:"vaccineArchived"`
	Quantity            int64  `json:"quantity"`
	UsageEntryID        int64  `json:"usageEntryId"`
	UsageVoided         bool   `json:"usageVoided"`
	UsageVoidReason     string `json:"usageVoidReason"`
	UsageAdministeredOn string `json:"usageAdministeredOn"`
	UsageQuantity       int64  `json:"usageQuantity"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

// AppointmentDTO is a private scheduled visit with its vaccine lines and linked
// usage state.
type AppointmentDTO struct {
	ID              int64                `json:"id"`
	PatientID       int64                `json:"patientId"`
	PatientName     string               `json:"patientName"`
	PatientCode     string               `json:"patientCode"`
	PatientArchived bool                 `json:"patientArchived"`
	Items           []AppointmentItemDTO `json:"items"`
	ScheduledOn     string               `json:"scheduledOn"`
	ScheduledTime   string               `json:"scheduledTime"`
	Note            string               `json:"note"`
	Status          string               `json:"status"`
	CancelReason    string               `json:"cancelReason"`
	CompletedAt     string               `json:"completedAt"`
	CancelledAt     string               `json:"cancelledAt"`
	CreatedAt       string               `json:"createdAt"`
	UpdatedAt       string               `json:"updatedAt"`
}

// AppointmentFilterDTO narrows the private appointment list.
type AppointmentFilterDTO struct {
	Status   string `json:"status"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

// AppointmentItemInputDTO is one vaccine line in an appointment form.
type AppointmentItemInputDTO struct {
	VaccineID int64 `json:"vaccineId"`
	Quantity  int64 `json:"quantity"`
}

// AppointmentInputDTO schedules a patient visit and reserves its doses.
type AppointmentInputDTO struct {
	PatientID     int64                     `json:"patientId"`
	ScheduledOn   string                    `json:"scheduledOn"`
	ScheduledTime string                    `json:"scheduledTime"`
	Note          string                    `json:"note"`
	Items         []AppointmentItemInputDTO `json:"items"`
}

// UpdateAppointmentInputDTO edits a scheduled visit and its linked doses.
type UpdateAppointmentInputDTO struct {
	PatientID     int64                     `json:"patientId"`
	ScheduledOn   string                    `json:"scheduledOn"`
	ScheduledTime string                    `json:"scheduledTime"`
	Note          string                    `json:"note"`
	Items         []AppointmentItemInputDTO `json:"items"`
	Reason        string                    `json:"reason"`
}

// CompleteAppointmentInputDTO marks a visit as completed. Doses were already
// deducted at booking; AdministeredOn optionally moves the linked usage rows to
// the date the doses were actually given.
type CompleteAppointmentInputDTO struct {
	ID             int64  `json:"id"`
	AdministeredOn string `json:"administeredOn"`
}

// CancelAppointmentInputDTO cancels a scheduled visit and restores its reserved
// doses to the same monthly stock.
type CancelAppointmentInputDTO struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// PatientSummaryDTO is the compact history summary shown at the top of a
// patient file.
type PatientSummaryDTO struct {
	TotalDoses            int64  `json:"totalDoses"`
	VaccineCount          int    `json:"vaccineCount"`
	LastDoseOn            string `json:"lastDoseOn"`
	ScheduledAppointments int    `json:"scheduledAppointments"`
	NextAppointmentOn     string `json:"nextAppointmentOn"`
	CompletedAppointments int    `json:"completedAppointments"`
}

// PatientDetailDTO is the complete local patient file. It is never published.
type PatientDetailDTO struct {
	Patient      PatientDTO        `json:"patient"`
	Summary      PatientSummaryDTO `json:"summary"`
	Usage        []UsageRowDTO     `json:"usage"`
	Appointments []AppointmentDTO  `json:"appointments"`
}

// CreateUsageInputDTO records one dose administration.
type CreateUsageInputDTO struct {
	PatientID      int64  `json:"patientId"`
	VaccineID      int64  `json:"vaccineId"`
	AdministeredOn string `json:"administeredOn"`
	Quantity       int64  `json:"quantity"`
	Note           string `json:"note"`
}

// UpdateUsageInputDTO corrects an existing entry.
type UpdateUsageInputDTO struct {
	ID             int64  `json:"id"`
	PatientID      int64  `json:"patientId"`
	AdministeredOn string `json:"administeredOn"`
	Quantity       int64  `json:"quantity"`
	Note           string `json:"note"`
	Reason         string `json:"reason"`
}

// VoidUsageInputDTO cancels an entry without deleting it.
type VoidUsageInputDTO struct {
	ID     int64  `json:"id"`
	Reason string `json:"reason"`
}

// VaccineStatDTO is one per-vaccine monthly statistic.
type VaccineStatDTO struct {
	VaccineID     int64  `json:"vaccineId"`
	Name          string `json:"name"`
	Code          string `json:"code"`
	Unit          string `json:"unit"`
	Archived      bool   `json:"archived"`
	Initialized   bool   `json:"initialized"`
	Opening       int64  `json:"opening"`
	Received      int64  `json:"received"`
	UsageTotal    int64  `json:"usageTotal"`
	Remaining     int64  `json:"remaining"`
	PreviousUsage int64  `json:"previousUsage"`
	UsageChange   int64  `json:"usageChange"`
	LowStock      bool   `json:"lowStock"`
}

// DailyTotalDTO is one local usage trend point.
type DailyTotalDTO struct {
	Date  string `json:"date"`
	Day   int    `json:"day"`
	Total int64  `json:"total"`
}

// MonthReportDTO is the local statistics view.
type MonthReportDTO struct {
	Month                  string           `json:"month"`
	State                  string           `json:"state"`
	Totals                 MonthTotalsDTO   `json:"totals"`
	PerVaccine             []VaccineStatDTO `json:"perVaccine"`
	Daily                  []DailyTotalDTO  `json:"daily"`
	PreviousMonth          string           `json:"previousMonth"`
	PreviousUsage          int64            `json:"previousUsage"`
	UsageChange            int64            `json:"usageChange"`
	HasComparison          bool             `json:"hasComparison"`
	ChangePercent          float64          `json:"changePercent"`
	ChangePercentAvailable bool             `json:"changePercentAvailable"`
	AdjustmentCount        int64            `json:"adjustmentCount"`
	GeneratedAt            string           `json:"generatedAt"`
}

// ForecastRowDTO is one transparent demand-planning estimate.
type ForecastRowDTO struct {
	VaccineID         int64    `json:"vaccineId"`
	Name              string   `json:"name"`
	Code              string   `json:"code"`
	Unit              string   `json:"unit"`
	Initialized       bool     `json:"initialized"`
	Method            string   `json:"method"`
	SampleMonths      []string `json:"sampleMonths"`
	SampleCount       int      `json:"sampleCount"`
	EstimatedDemand   int64    `json:"estimatedDemand"`
	OpeningSuggestion int64    `json:"openingSuggestion"`
	PriorMonth        string   `json:"priorMonth"`
	PriorExists       bool     `json:"priorExists"`
	SuggestedRequest  int64    `json:"suggestedRequest"`
}

// ForecastDTO is an explicit planning aid, not a medical recommendation.
type ForecastDTO struct {
	TargetMonth       string           `json:"targetMonth"`
	GeneratedAt       string           `json:"generatedAt"`
	LowStockThreshold int64            `json:"lowStockThreshold"`
	Rows              []ForecastRowDTO `json:"rows"`
}

// AuditEventDTO is one minimized local audit record.
type AuditEventDTO struct {
	ID         int64  `json:"id"`
	EventType  string `json:"eventType"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Month      string `json:"month"`
	Reason     string `json:"reason"`
	Metadata   string `json:"metadata"`
	CreatedAt  string `json:"createdAt"`
}

// AuditPageDTO is a page of audit records.
type AuditPageDTO struct {
	Events []AuditEventDTO `json:"events"`
	Total  int64           `json:"total"`
}

// AuditFilterDTO narrows audit review.
type AuditFilterDTO struct {
	EventType string `json:"eventType"`
	Month     string `json:"month"`
	Limit     int64  `json:"limit"`
	Offset    int64  `json:"offset"`
}

// SettingsDTO exposes local configuration without secrets.
type SettingsDTO struct {
	LowStockThreshold        int64  `json:"lowStockThreshold"`
	PublisherEndpoint        string `json:"publisherEndpoint"`
	PublisherTokenConfigured bool   `json:"publisherTokenConfigured"`
	PublisherTokenStorage    string `json:"publisherTokenStorage"`
	DataDir                  string `json:"dataDir"`
	DatabasePath             string `json:"databasePath"`
	BackupDir                string `json:"backupDir"`
}

// PublicationRecordDTO is local publication history.
type PublicationRecordDTO struct {
	ID            int64  `json:"id"`
	SnapshotID    string `json:"snapshotId"`
	Month         string `json:"month"`
	Status        string `json:"status"`
	Endpoint      string `json:"endpoint"`
	Version       int64  `json:"version"`
	PayloadSHA256 string `json:"payloadSha256"`
	PublishedAt   string `json:"publishedAt"`
	UnpublishedAt string `json:"unpublishedAt"`
	CreatedAt     string `json:"createdAt"`
}

// PublicationPreviewDTO is exactly the aggregate payload that would be sent.
// It contains no patient-level information.
type PublicationPreviewDTO struct {
	Month        string              `json:"month"`
	MonthState   string              `json:"monthState"`
	Snapshot     publishing.Snapshot `json:"snapshot"`
	Warnings     []string            `json:"warnings"`
	CanPublish   bool                `json:"canPublish"`
	IncludedRows int                 `json:"includedRows"`
	GeneratedAt  string              `json:"generatedAt"`
}

// PublicationStatusDTO reports local publication configuration and history.
type PublicationStatusDTO struct {
	EndpointConfigured       bool                   `json:"endpointConfigured"`
	PublisherTokenConfigured bool                   `json:"publisherTokenConfigured"`
	PublisherTokenStorage    string                 `json:"publisherTokenStorage"`
	Latest                   *PublicationRecordDTO  `json:"latest"`
	History                  []PublicationRecordDTO `json:"history"`
}

// PublicationResultDTO reports a publish or unpublish action.
type PublicationResultDTO struct {
	OK          bool   `json:"ok"`
	Code        string `json:"code"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	SnapshotID  string `json:"snapshotId"`
	Version     int64  `json:"version"`
	PublishedAt string `json:"publishedAt"`
	Duplicate   bool   `json:"duplicate"`
}

// BackupResultDTO reports a created backup.
type BackupResultDTO struct {
	OK        bool   `json:"ok"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
	CreatedAt string `json:"createdAt"`
}

// ExportResultDTO reports an exported file.
type ExportResultDTO struct {
	OK      bool   `json:"ok"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
}

func vaccineDTO(v generated.Vaccine) VaccineDTO {
	return VaccineDTO{
		ID: v.ID, Name: v.Name, Code: v.Code, Description: v.Description,
		Category: v.Category, Unit: v.Unit, Archived: v.IsArchived == 1,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

func patientDTO(p generated.Patient) PatientDTO {
	return PatientDTO{
		ID: p.ID, FullName: p.FullName, PatientCode: p.PatientCode,
		Archived: p.IsArchived == 1, ArchivedAt: p.ArchivedAt.String,
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func usageEntryDTO(entry generated.UsageEntry) UsageEntryDTO {
	return UsageEntryDTO{
		ID: entry.ID, PatientID: entry.PatientID, VaccineID: entry.VaccineID,
		AdministeredOn: entry.AdministeredOn, Month: entry.Month,
		Quantity: entry.Quantity, Note: entry.Note, Voided: entry.VoidedAt.Valid,
		VoidReason: entry.VoidReason, CreatedAt: entry.CreatedAt, UpdatedAt: entry.UpdatedAt,
	}
}

func usageRowDTO(row generated.ListUsageEntriesForVaccineDayRow) UsageRowDTO {
	return UsageRowDTO{
		ID: row.ID, PatientID: row.PatientID, VaccineID: row.VaccineID,
		AdministeredOn: row.AdministeredOn, Month: row.Month, Quantity: row.Quantity,
		Note: row.Note, Voided: row.VoidedAt.Valid, VoidReason: row.VoidReason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		PatientName: row.PatientName, PatientCode: row.PatientCode,
		PatientArchived: row.PatientArchived == 1,
		VaccineName:     row.VaccineName, VaccineCode: row.VaccineCode, VaccineUnit: row.VaccineUnit,
	}
}

func appointmentDTO(view service.AppointmentView) AppointmentDTO {
	items := make([]AppointmentItemDTO, 0, len(view.Items))
	for _, item := range view.Items {
		items = append(items, AppointmentItemDTO{
			ID: item.ID, VaccineID: item.VaccineID, VaccineName: item.VaccineName,
			VaccineCode: item.VaccineCode, VaccineUnit: item.VaccineUnit,
			VaccineArchived: item.VaccineArchived, Quantity: item.Quantity,
			UsageEntryID: item.UsageEntryID, UsageVoided: item.UsageVoided,
			UsageVoidReason: item.UsageVoidReason, UsageAdministeredOn: item.UsageAdministeredOn,
			UsageQuantity: item.UsageQuantity, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return AppointmentDTO{
		ID: view.ID, PatientID: view.PatientID, PatientName: view.PatientName,
		PatientCode: view.PatientCode, PatientArchived: view.PatientArchived,
		Items: items, ScheduledOn: view.ScheduledOn, ScheduledTime: view.ScheduledTime,
		Note: view.Note, Status: view.Status, CancelReason: view.CancelReason,
		CompletedAt: view.CompletedAt, CancelledAt: view.CancelledAt,
		CreatedAt: view.CreatedAt, UpdatedAt: view.UpdatedAt,
	}
}

func usageRowPatientDTO(row generated.ListUsageEntriesForPatientRow) UsageRowDTO {
	return UsageRowDTO{
		ID: row.ID, PatientID: row.PatientID, VaccineID: row.VaccineID,
		AdministeredOn: row.AdministeredOn, Month: row.Month, Quantity: row.Quantity,
		Note: row.Note, Voided: row.VoidedAt.Valid, VoidReason: row.VoidReason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		PatientName: row.PatientName, PatientCode: row.PatientCode,
		PatientArchived: row.PatientArchived == 1,
		VaccineName:     row.VaccineName, VaccineCode: row.VaccineCode, VaccineUnit: row.VaccineUnit,
	}
}

func usageRowDTOFromMonth(row generated.ListUsageEntriesForVaccineMonthRow) UsageRowDTO {
	return UsageRowDTO{
		ID: row.ID, PatientID: row.PatientID, VaccineID: row.VaccineID,
		AdministeredOn: row.AdministeredOn, Month: row.Month, Quantity: row.Quantity,
		Note: row.Note, Voided: row.VoidedAt.Valid, VoidReason: row.VoidReason,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		PatientName: row.PatientName, PatientCode: row.PatientCode,
		PatientArchived: row.PatientArchived == 1,
		VaccineName:     row.VaccineName, VaccineCode: row.VaccineCode, VaccineUnit: row.VaccineUnit,
	}
}

func monthTotalsDTO(t service.MonthTotals) MonthTotalsDTO {
	return MonthTotalsDTO{
		OpeningBalance: t.OpeningBalance, StockReceived: t.StockReceived,
		UsageTotal: t.UsageTotal, Remaining: t.Remaining,
		ActiveVaccines: t.ActiveVaccines, InitializedVaccines: t.InitializedVaccines,
		UninitializedActive: t.UninitializedActive, LowStockCount: t.LowStockCount,
	}
}

func monthOverviewDTO(o service.MonthOverview) MonthOverviewDTO {
	rows := make([]MonthRowDTO, 0, len(o.Rows))
	for _, row := range o.Rows {
		rows = append(rows, MonthRowDTO{
			VaccineID: row.VaccineID, VaccineName: row.VaccineName, VaccineCode: row.VaccineCode,
			Unit: row.Unit, Archived: row.Archived, Initialized: row.Initialized, Status: row.Status,
			OpeningBalance: row.OpeningBalance, StockReceived: row.StockReceived,
			UsageTotal: row.UsageTotal, Remaining: row.Remaining,
			ClosingBalance: row.ClosingBalance, HasClosingBalance: row.HasClosingBalance,
			CarryoverOverridden: row.CarryoverOverridden, CarryoverReason: row.CarryoverReason,
			SuggestedOpening: row.SuggestedOpening, PriorMonth: row.PriorMonth,
			PriorExists: row.PriorExists, PriorClosed: row.PriorClosed,
			DailyUsage: row.DailyUsage, LowStock: row.LowStock,
		})
	}
	adjustments := make([]AdjustmentDTO, 0, len(o.Adjustments))
	for _, a := range o.Adjustments {
		adjustments = append(adjustments, AdjustmentDTO{
			ID: a.ID, VaccineID: a.VaccineID, VaccineName: a.VaccineName, Unit: a.Unit,
			Amount: a.Amount, BalanceBefore: a.BalanceBefore, BalanceAfter: a.BalanceAfter,
			Reason: a.Reason, CreatedAt: a.CreatedAt,
		})
	}
	return MonthOverviewDTO{
		Month: o.Month, Days: o.Days, State: string(o.State), Rows: rows,
		Totals: monthTotalsDTO(o.Totals), LowStockThreshold: o.LowStockThreshold,
		Adjustments: adjustments, GeneratedAt: o.GeneratedAt,
	}
}

func monthReportDTO(r service.MonthReport) MonthReportDTO {
	perVaccine := make([]VaccineStatDTO, 0, len(r.PerVaccine))
	for _, v := range r.PerVaccine {
		perVaccine = append(perVaccine, VaccineStatDTO{
			VaccineID: v.VaccineID, Name: v.Name, Code: v.Code, Unit: v.Unit,
			Archived: v.Archived, Initialized: v.Initialized, Opening: v.Opening,
			Received: v.Received, UsageTotal: v.UsageTotal, Remaining: v.Remaining,
			PreviousUsage: v.PreviousUsage, UsageChange: v.UsageChange, LowStock: v.LowStock,
		})
	}
	daily := make([]DailyTotalDTO, 0, len(r.Daily))
	for _, d := range r.Daily {
		daily = append(daily, DailyTotalDTO{Date: d.Date, Day: d.Day, Total: d.Total})
	}
	return MonthReportDTO{
		Month: r.Month, State: string(r.State), Totals: monthTotalsDTO(r.Totals),
		PerVaccine: perVaccine, Daily: daily, PreviousMonth: r.PreviousMonth,
		PreviousUsage: r.PreviousUsage, UsageChange: r.UsageChange,
		HasComparison: r.HasComparison, ChangePercent: r.ChangePercent,
		ChangePercentAvailable: r.ChangePercentAvailable,
		AdjustmentCount:        r.AdjustmentCount, GeneratedAt: r.GeneratedAt,
	}
}

func forecastDTO(f service.Forecast) ForecastDTO {
	rows := make([]ForecastRowDTO, 0, len(f.Rows))
	for _, row := range f.Rows {
		rows = append(rows, ForecastRowDTO{
			VaccineID: row.VaccineID, Name: row.Name, Code: row.Code, Unit: row.Unit,
			Initialized: row.Initialized, Method: row.Method, SampleMonths: row.SampleMonths,
			SampleCount: row.SampleCount, EstimatedDemand: row.EstimatedDemand,
			OpeningSuggestion: row.OpeningSuggestion, PriorMonth: row.PriorMonth,
			PriorExists: row.PriorExists, SuggestedRequest: row.SuggestedRequest,
		})
	}
	return ForecastDTO{
		TargetMonth: f.TargetMonth, GeneratedAt: f.GeneratedAt,
		LowStockThreshold: f.LowStockThreshold, Rows: rows,
	}
}

func publicationRecordDTO(p generated.Publication) PublicationRecordDTO {
	return PublicationRecordDTO{
		ID: p.ID, SnapshotID: p.SnapshotID, Month: p.Month, Status: p.Status,
		Endpoint: p.Endpoint, Version: p.Version, PayloadSHA256: p.PayloadSha256,
		PublishedAt: p.PublishedAt, UnpublishedAt: p.UnpublishedAt.String,
		CreatedAt: p.CreatedAt,
	}
}

func backupResult(info backup.Info) BackupResultDTO {
	return BackupResultDTO{
		OK: true, Message: "أُنشئت النسخة الاحتياطية وتم التحقق منها.",
		Path: info.Path, SizeBytes: info.SizeBytes, SHA256: info.SHA256, CreatedAt: info.CreatedAt,
	}
}

// parseMetadata keeps stored audit metadata as text for display. Malformed
// metadata from older versions is shown as-is rather than breaking the page.
func parseMetadata(value string) string {
	var v any
	if err := json.Unmarshal([]byte(value), &v); err != nil {
		return value
	}
	normalized, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return value
	}
	return string(normalized)
}
