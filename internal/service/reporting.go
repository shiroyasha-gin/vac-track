package service

import (
	"context"
	"sort"
	"time"

	"vaccineshit/internal/domain"
	plan "vaccineshit/internal/reporting"
	"vaccineshit/internal/store/generated"
)

// MonthRow is one vaccine row in the monthly overview table.
type MonthRow struct {
	VaccineID           int64
	VaccineName         string
	VaccineCode         string
	Unit                string
	Archived            bool
	Initialized         bool
	Status              string
	OpeningBalance      int64
	StockReceived       int64
	UsageTotal          int64
	Remaining           int64
	ClosingBalance      int64
	HasClosingBalance   bool
	CarryoverOverridden bool
	CarryoverReason     string
	SuggestedOpening    int64
	PriorMonth          string
	PriorExists         bool
	PriorClosed         bool
	DailyUsage          []int64
	LowStock            bool
}

// MonthTotals aggregates the monthly table.
type MonthTotals struct {
	OpeningBalance      int64
	StockReceived       int64
	UsageTotal          int64
	Remaining           int64
	ActiveVaccines      int
	InitializedVaccines int
	UninitializedActive int
	LowStockCount       int
}

// AdjustmentInfo is a private local stock correction summary.
type AdjustmentInfo struct {
	ID            int64
	VaccineID     int64
	VaccineName   string
	Unit          string
	Amount        int64
	BalanceBefore int64
	BalanceAfter  int64
	Reason        string
	CreatedAt     string
}

// MonthOverview is the complete monthly table and setup state.
type MonthOverview struct {
	Month             string
	Days              int
	State             domain.MonthState
	Rows              []MonthRow
	Totals            MonthTotals
	LowStockThreshold int64
	Adjustments       []AdjustmentInfo
	GeneratedAt       string
}

// VaccineStat is a monthly statistic per vaccine.
type VaccineStat struct {
	VaccineID     int64
	Name          string
	Code          string
	Unit          string
	Archived      bool
	Initialized   bool
	Opening       int64
	Received      int64
	UsageTotal    int64
	Remaining     int64
	PreviousUsage int64
	UsageChange   int64
	LowStock      bool
}

// DailyTotal is one day in the local trend.
type DailyTotal struct {
	Date  string
	Day   int
	Total int64
}

// MonthReport combines totals, per-vaccine statistics, and trends.
type MonthReport struct {
	Month                  string
	State                  domain.MonthState
	Totals                 MonthTotals
	PerVaccine             []VaccineStat
	Daily                  []DailyTotal
	PreviousMonth          string
	PreviousUsage          int64
	UsageChange            int64
	HasComparison          bool
	ChangePercent          float64
	ChangePercentAvailable bool
	AdjustmentCount        int64
	GeneratedAt            string
}

// ForecastRow is one vaccine's transparent next-month planning estimate.
type ForecastRow struct {
	VaccineID         int64
	Name              string
	Code              string
	Unit              string
	Initialized       bool
	Method            string
	SampleMonths      []string
	SampleCount       int
	EstimatedDemand   int64
	OpeningSuggestion int64
	PriorMonth        string
	PriorExists       bool
	SuggestedRequest  int64
}

// Forecast is a planning estimate for a target month. It is explicitly not a
// medical recommendation.
type Forecast struct {
	TargetMonth       string
	GeneratedAt       string
	LowStockThreshold int64
	Rows              []ForecastRow
}

// ReportingService derives figures from source inventory and non-voided usage
// records. It does not maintain duplicated mutable totals.
type ReportingService struct {
	base
	settings  *SettingsService
	inventory *InventoryService
	usage     *UsageService
}

// MonthOverview builds the monthly table, including day columns and setup
// suggestions.
func (s *ReportingService) MonthOverview(ctx context.Context, monthStr string) (MonthOverview, error) {
	month, err := domain.ParseMonth(monthStr)
	if err != nil {
		return MonthOverview{}, err
	}
	threshold, err := s.settings.LowStockThreshold(ctx)
	if err != nil {
		return MonthOverview{}, err
	}
	q := s.store.Queries()
	vaccines, err := q.ListVaccines(ctx, 1)
	if err != nil {
		return MonthOverview{}, err
	}
	inventoryRows, err := q.ListMonthlyInventoryForMonth(ctx, month.String())
	if err != nil {
		return MonthOverview{}, err
	}
	usageTotals, err := q.ListUsageTotalsByVaccineForMonth(ctx, month.String())
	if err != nil {
		return MonthOverview{}, err
	}
	dailyTotals, err := q.ListDailyUsageTotalsForMonth(ctx, month.String())
	if err != nil {
		return MonthOverview{}, err
	}
	prevMonth := month.Previous()
	prevRows, err := q.ListMonthlyInventoryForMonth(ctx, prevMonth.String())
	if err != nil {
		return MonthOverview{}, err
	}
	prevUsageTotals, err := q.ListUsageTotalsByVaccineForMonth(ctx, prevMonth.String())
	if err != nil {
		return MonthOverview{}, err
	}
	adjustmentRows, err := q.ListInventoryAdjustmentsForMonth(ctx, month.String())
	if err != nil {
		return MonthOverview{}, err
	}

	inventoryByVaccine := make(map[int64]generated.ListMonthlyInventoryForMonthRow, len(inventoryRows))
	for _, row := range inventoryRows {
		inventoryByVaccine[row.VaccineID] = row
	}
	usageByVaccine := make(map[int64]int64, len(usageTotals))
	for _, row := range usageTotals {
		usageByVaccine[row.VaccineID] = row.Total
	}
	prevInventory := make(map[int64]generated.ListMonthlyInventoryForMonthRow, len(prevRows))
	for _, row := range prevRows {
		prevInventory[row.VaccineID] = row
	}
	prevUsage := make(map[int64]int64, len(prevUsageTotals))
	for _, row := range prevUsageTotals {
		prevUsage[row.VaccineID] = row.Total
	}

	days := month.Days()
	dailyByVaccine := make(map[int64][]int64)
	for _, row := range dailyTotals {
		date, err := domain.ParseDate(row.AdministeredOn)
		if err != nil {
			return MonthOverview{}, err
		}
		if date.Year() != month.Year || date.Month() != month.Month {
			return MonthOverview{}, domain.NewError(domain.CodeInvalidDate, "", "تاريخ الاستخدام المخزن لا يطابق شهر المخزون.")
		}
		series, ok := dailyByVaccine[row.VaccineID]
		if !ok {
			series = make([]int64, days)
			dailyByVaccine[row.VaccineID] = series
		}
		series[date.Day()-1] += row.Total
	}

	var rows []MonthRow
	var totals MonthTotals
	for _, vaccine := range vaccines {
		inv, initialized := inventoryByVaccine[vaccine.ID]
		usage := usageByVaccine[vaccine.ID]
		if vaccine.IsArchived == 1 && !initialized && usage == 0 {
			continue
		}
		row := MonthRow{
			VaccineID:        vaccine.ID,
			VaccineName:      vaccine.Name,
			VaccineCode:      vaccine.Code,
			Unit:             vaccine.Unit,
			Archived:         vaccine.IsArchived == 1,
			Initialized:      initialized,
			Status:           string(domain.StatusMissing),
			UsageTotal:       usage,
			PriorMonth:       prevMonth.String(),
			SuggestedOpening: 0,
			DailyUsage:       dailyByVaccine[vaccine.ID],
		}
		if row.DailyUsage == nil {
			row.DailyUsage = make([]int64, days)
		}
		if initialized {
			row.Status = inv.Status
			row.OpeningBalance = inv.OpeningBalance
			row.StockReceived = inv.StockReceived
			row.CarryoverOverridden = inv.CarryoverOverridden == 1
			row.CarryoverReason = inv.CarryoverReason
			row.Remaining = inv.OpeningBalance + inv.StockReceived - usage
			if inv.ClosingBalance.Valid {
				row.ClosingBalance = inv.ClosingBalance.Int64
				row.HasClosingBalance = true
				if inv.Status == string(domain.StatusClosed) {
					row.Remaining = inv.ClosingBalance.Int64
				}
			}
		}
		if previous, ok := prevInventory[vaccine.ID]; ok {
			row.PriorExists = true
			row.PriorClosed = previous.Status == string(domain.StatusClosed)
			if row.PriorClosed && previous.ClosingBalance.Valid {
				row.SuggestedOpening = previous.ClosingBalance.Int64
			} else {
				row.SuggestedOpening = previous.OpeningBalance + previous.StockReceived - prevUsage[vaccine.ID]
			}
		}
		row.LowStock = vaccine.IsArchived == 0 && initialized && domain.IsLowStock(row.Remaining, threshold)
		if vaccine.IsArchived == 0 {
			totals.ActiveVaccines++
			if !initialized {
				totals.UninitializedActive++
			} else {
				totals.InitializedVaccines++
			}
		}
		if row.LowStock {
			totals.LowStockCount++
		}
		totals.OpeningBalance += row.OpeningBalance
		totals.StockReceived += row.StockReceived
		totals.UsageTotal += row.UsageTotal
		totals.Remaining += row.Remaining
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Archived != rows[j].Archived {
			return !rows[i].Archived
		}
		if rows[i].VaccineName == rows[j].VaccineName {
			return rows[i].VaccineID < rows[j].VaccineID
		}
		return rows[i].VaccineName < rows[j].VaccineName
	})

	state := domain.MonthNotInitialized
	if len(inventoryRows) > 0 {
		state = domain.MonthClosed
		for _, row := range inventoryRows {
			if row.Status == string(domain.StatusOpen) {
				state = domain.MonthOpen
				break
			}
		}
	}

	adjustments := make([]AdjustmentInfo, 0, len(adjustmentRows))
	for _, row := range adjustmentRows {
		adjustments = append(adjustments, AdjustmentInfo{
			ID:            row.ID,
			VaccineID:     row.VaccineID,
			VaccineName:   row.VaccineName,
			Unit:          row.VaccineUnit,
			Amount:        row.Amount,
			BalanceBefore: row.BalanceBefore,
			BalanceAfter:  row.BalanceAfter,
			Reason:        row.Reason,
			CreatedAt:     row.CreatedAt,
		})
	}

	return MonthOverview{
		Month:             month.String(),
		Days:              days,
		State:             state,
		Rows:              rows,
		Totals:            totals,
		LowStockThreshold: threshold,
		Adjustments:       adjustments,
		GeneratedAt:       s.timestamp(),
	}, nil
}

// Report builds the local monthly statistics view.
func (s *ReportingService) Report(ctx context.Context, monthStr string) (MonthReport, error) {
	month, err := domain.ParseMonth(monthStr)
	if err != nil {
		return MonthReport{}, err
	}
	overview, err := s.MonthOverview(ctx, monthStr)
	if err != nil {
		return MonthReport{}, err
	}
	q := s.store.Queries()
	prevMonth := month.Previous()
	prevUsageRows, err := q.ListUsageTotalsByVaccineForMonth(ctx, prevMonth.String())
	if err != nil {
		return MonthReport{}, err
	}
	prevByVaccine := make(map[int64]int64, len(prevUsageRows))
	for _, row := range prevUsageRows {
		prevByVaccine[row.VaccineID] = row.Total
	}
	prevTotal, err := q.SumActiveUsageForMonth(ctx, prevMonth.String())
	if err != nil {
		return MonthReport{}, err
	}
	prevInventory, err := q.ListMonthlyInventoryRowsForMonth(ctx, prevMonth.String())
	if err != nil {
		return MonthReport{}, err
	}
	prevAdjustments, err := q.CountInventoryAdjustmentsForMonth(ctx, month.String())
	if err != nil {
		return MonthReport{}, err
	}

	perVaccine := make([]VaccineStat, 0, len(overview.Rows))
	for _, row := range overview.Rows {
		previous := prevByVaccine[row.VaccineID]
		if !row.Initialized && row.UsageTotal == 0 && previous == 0 {
			continue
		}
		perVaccine = append(perVaccine, VaccineStat{
			VaccineID:     row.VaccineID,
			Name:          row.VaccineName,
			Code:          row.VaccineCode,
			Unit:          row.Unit,
			Archived:      row.Archived,
			Initialized:   row.Initialized,
			Opening:       row.OpeningBalance,
			Received:      row.StockReceived,
			UsageTotal:    row.UsageTotal,
			Remaining:     row.Remaining,
			PreviousUsage: previous,
			UsageChange:   row.UsageTotal - previous,
			LowStock:      row.LowStock,
		})
	}

	dailyTotals, err := q.ListDailyUsageTotalsForMonth(ctx, month.String())
	if err != nil {
		return MonthReport{}, err
	}
	byDay := make(map[int]int64)
	for _, row := range dailyTotals {
		date, err := domain.ParseDate(row.AdministeredOn)
		if err != nil {
			return MonthReport{}, err
		}
		byDay[date.Day()] += row.Total
	}
	daily := make([]DailyTotal, 0, overview.Days)
	for day := 1; day <= overview.Days; day++ {
		date := time.Date(month.Year, month.Month, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		daily = append(daily, DailyTotal{Date: date, Day: day, Total: byDay[day]})
	}

	hasComparison := len(prevInventory) > 0 || prevTotal > 0
	change := overview.Totals.UsageTotal - prevTotal
	percent := 0.0
	percentAvailable := hasComparison && prevTotal > 0
	if percentAvailable {
		percent = float64(change) / float64(prevTotal) * 100
	}

	return MonthReport{
		Month:                  month.String(),
		State:                  overview.State,
		Totals:                 overview.Totals,
		PerVaccine:             perVaccine,
		Daily:                  daily,
		PreviousMonth:          prevMonth.String(),
		PreviousUsage:          prevTotal,
		UsageChange:            change,
		HasComparison:          hasComparison,
		ChangePercent:          percent,
		ChangePercentAvailable: percentAvailable,
		AdjustmentCount:        prevAdjustments,
		GeneratedAt:            s.timestamp(),
	}, nil
}

// Forecast estimates demand for a target month from up to three previous
// complete (closed) months. The method and sample window are returned.
func (s *ReportingService) Forecast(ctx context.Context, targetMonthStr string) (Forecast, error) {
	target, err := domain.ParseMonth(targetMonthStr)
	if err != nil {
		return Forecast{}, err
	}
	vaccines, err := s.store.Queries().ListVaccines(ctx, 0)
	if err != nil {
		return Forecast{}, err
	}
	threshold, err := s.settings.LowStockThreshold(ctx)
	if err != nil {
		return Forecast{}, err
	}
	rows := make([]ForecastRow, 0, len(vaccines))
	for _, vaccine := range vaccines {
		historyRows, err := s.store.Queries().ListClosedMonthUsageHistory(ctx, generated.ListClosedMonthUsageHistoryParams{
			VaccineID: vaccine.ID,
			Month:     target.String(),
			Limit:     3,
		})
		if err != nil {
			return Forecast{}, err
		}
		history := make([]plan.MonthUsage, 0, len(historyRows))
		for _, row := range historyRows {
			history = append(history, plan.MonthUsage{Month: row.Month, Doses: row.UsageTotal})
		}
		estimate := plan.EstimateDemand(history)
		suggestion, err := s.inventory.Suggestion(ctx, vaccine.ID, target.String())
		if err != nil {
			return Forecast{}, err
		}
		rows = append(rows, ForecastRow{
			VaccineID:         vaccine.ID,
			Name:              vaccine.Name,
			Code:              vaccine.Code,
			Unit:              vaccine.Unit,
			Method:            estimate.Method,
			SampleMonths:      estimate.SampleMonths,
			SampleCount:       len(estimate.SampleMonths),
			EstimatedDemand:   estimate.Doses,
			OpeningSuggestion: suggestion.Amount,
			PriorMonth:        suggestion.PriorMonth,
			PriorExists:       suggestion.PriorExists,
			SuggestedRequest:  plan.SuggestedRequest(estimate.Doses, suggestion.Amount),
		})
	}
	return Forecast{
		TargetMonth:       target.String(),
		GeneratedAt:       s.timestamp(),
		LowStockThreshold: threshold,
		Rows:              rows,
	}, nil
}
