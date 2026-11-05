package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"

	"vaccineshit/internal/domain"
)

// CSVService produces safe, explicitly scoped exports. Aggregate export never
// queries patient data; patient-level export is a separate local-only path.
type CSVService struct {
	base
	reporting *ReportingService
}

// AggregateCSV exports the monthly vaccine/month summary. It contains no
// patient identifiers.
func (s *CSVService) AggregateCSV(ctx context.Context, monthStr string) ([]byte, error) {
	overview, err := s.reporting.MonthOverview(ctx, monthStr)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	// A UTF-8 BOM keeps Arabic text readable when the CSV is opened in Excel.
	buf.WriteString("\uFEFF")
	w := csv.NewWriter(&buf)
	records := [][]string{
		{"مخزون اللقاحات - ملخص شهري مجمّع"},
		{"الشهر", overview.Month},
		{"الوحدات", "جرعات كاملة (راجع عمود وحدة كل لقاح)"},
		{"حالة الشهر", inventoryStatusLabel(string(overview.State))},
		{"وقت الإنشاء (UTC)", overview.GeneratedAt},
		{"الخصوصية", "بيانات مجمّعة فقط. لا تتضمن أي معلومات على مستوى المريض."},
		{},
		{"اللقاح", "الرمز", "الوحدة", "الرصيد الافتتاحي", "المخزون الوارد", "إجمالي الاستخدام", "المتبقي", "الحالة", "مؤرشف"},
	}
	for _, row := range overview.Rows {
		status := row.Status
		if !row.Initialized {
			status = string(domain.StatusMissing)
		}
		records = append(records, []string{
			row.VaccineName,
			row.VaccineCode,
			row.Unit,
			strconv.FormatInt(row.OpeningBalance, 10),
			strconv.FormatInt(row.StockReceived, 10),
			strconv.FormatInt(row.UsageTotal, 10),
			strconv.FormatInt(row.Remaining, 10),
			inventoryStatusLabel(status),
			boolWord(row.Archived),
		})
	}
	records = append(records, []string{
		"الإجمالي",
		"",
		"",
		strconv.FormatInt(overview.Totals.OpeningBalance, 10),
		strconv.FormatInt(overview.Totals.StockReceived, 10),
		strconv.FormatInt(overview.Totals.UsageTotal, 10),
		strconv.FormatInt(overview.Totals.Remaining, 10),
		"",
		"",
	})
	for _, record := range records {
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PatientUsageCSV exports local patient-level usage. The caller must display a
// privacy warning and confirm the export. Notes are intentionally omitted.
func (s *CSVService) PatientUsageCSV(ctx context.Context, monthStr string) ([]byte, error) {
	if _, err := domain.ParseMonth(monthStr); err != nil {
		return nil, err
	}
	rows, err := s.store.Queries().ListUsageEntriesForMonthExport(ctx, monthStr)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	// A UTF-8 BOM keeps Arabic text readable when the CSV is opened in Excel.
	buf.WriteString("\uFEFF")
	w := csv.NewWriter(&buf)
	records := [][]string{
		{"مخزون اللقاحات - استخدام على مستوى المريض (محلي / سري)"},
		{"تحذير", "يحتوي على معلومات صحية على مستوى المريض. أبقِ هذا الملف محلياً ولا تشاركه إلا مع الموظفين المصرح لهم."},
		{"الشهر", monthStr},
		{"الوحدات", "جرعات كاملة (راجع عمود وحدة كل لقاح)"},
		{"وقت الإنشاء (UTC)", s.timestamp()},
		{"الخصوصية", "للاستخدام المحلي المصرح به فقط. يجب عدم نشر هذا الملف أو رفعه."},
		{},
		{"التاريخ", "المريض", "رمز المريض", "اللقاح", "رمز اللقاح", "الجرعات", "الوحدة", "الحالة"},
	}
	for _, row := range rows {
		status := "مسجل"
		if row.VoidedAt.Valid {
			status = "ملغى"
		}
		records = append(records, []string{
			row.AdministeredOn,
			row.PatientName,
			row.PatientCode,
			row.VaccineName,
			row.VaccineCode,
			strconv.FormatInt(row.Quantity, 10),
			row.VaccineUnit,
			status,
		})
	}
	for _, record := range records {
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func inventoryStatusLabel(status string) string {
	switch status {
	case string(domain.StatusOpen):
		return "مفتوح"
	case string(domain.StatusClosed):
		return "مغلق"
	case string(domain.StatusMissing):
		return "غير مهيأ"
	default:
		return status
	}
}

func boolWord(value bool) string {
	if value {
		return "نعم"
	}
	return "لا"
}
