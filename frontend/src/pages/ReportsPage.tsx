import { useMemo, useRef, useState, type ReactNode } from 'react';
import {
  Activity,
  BarChart3,
  CalendarClock,
  FileSpreadsheet,
  Package,
  Printer,
  RefreshCw,
  Search,
  ShieldAlert,
  SlidersHorizontal,
  TrendingDown,
  TrendingUp,
  Trophy,
  X,
} from 'lucide-react';
import { api, app, errorMessage } from '../lib/api';
import { cn } from '../components/ui';
import {
  currentMonth,
  formatDate,
  formatInteger,
  formatPercent,
  formatSigned,
  formatTimestamp,
  monthLabel,
  shiftMonth,
} from '../lib/format';
import { useAsync } from '../lib/hooks';
import { printReportNode } from '../lib/print';
import {
  defaultReportFilters,
  filterForecastRows,
  filterVaccineStats,
  isDefaultFilters,
  reportSortLabels,
  reportStatusLabels,
  type ReportFilters,
  type ReportSort,
  type ReportStatusFilter,
} from '../lib/reportFilters';
import { useMonth } from '../components/MonthContext';
import { useToast } from '../components/Toast';
import { DailyUsageChart, StockHealthChart, UsageComparisonChart } from '../components/Charts';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  EmptyState,
  Field,
  Modal,
  IconButton,
  InlineError,
  Input,
  PageHeader,
  Select,
  Spinner,
  StatCard,
  StatusBadge,
} from '../components/ui';

function methodLabel(method: string): string {
  switch (method) {
    case 'three_month_average':
      return 'متوسط ثلاثة أشهر';
    case 'most_recent_complete_month':
      return 'آخر شهر مكتمل';
    case 'no_history':
      return 'لا يوجد سجل';
    default:
      return method.replaceAll('_', ' ');
  }
}

function sampleMonthsLabel(count: number): string {
  if (count === 1) {
    return 'شهر واحد';
  }
  if (count === 2) {
    return 'شهران';
  }
  if (count > 2 && count <= 10) {
    return `${formatInteger(count)} أشهر`;
  }
  return `${formatInteger(count)} شهراً`;
}

function SummaryMetric({
  label,
  value,
  detail,
  tone = 'default',
  icon,
}: {
  label: string;
  value: string;
  detail: string;
  tone?: 'default' | 'emerald' | 'amber' | 'rose' | 'sky';
  icon: ReactNode;
}) {
  const toneClass = {
    default: 'text-slate-100',
    emerald: 'text-emerald-300',
    amber: 'text-amber-300',
    rose: 'text-rose-300',
    sky: 'text-sky-300',
  }[tone];
  return (
    <div className="bg-slate-950 px-4 py-3.5">
      <div className="flex items-center gap-1.5 text-slate-500">
        <span className="[&>svg]:h-3.5 [&>svg]:w-3.5">{icon}</span>
        <span className="text-[11px]">{label}</span>
      </div>
      <p className={cn('report-summary-metric-value mt-1.5 text-xl font-semibold leading-none tabular-nums', toneClass)}>{value}</p>
      <p className="mt-1 truncate text-[11px] text-slate-500" title={detail}>
        {detail}
      </p>
    </div>
  );
}

function FilterChip({ label, onRemove }: { label: string; onRemove: () => void }) {
  return (
    <span className="inline-flex items-center gap-1 rounded-full border border-slate-700 bg-slate-900 py-0.5 pe-1 ps-2.5 text-[11px] text-slate-300">
      {label}
      <button
        type="button"
        aria-label={`إزالة ${label}`}
        onClick={onRemove}
        className="focus-ring rounded-full p-0.5 text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-200"
      >
        <X className="h-3 w-3" />
      </button>
    </span>
  );
}

export function ReportsPage() {
  const { month } = useMonth();
  const { notify } = useToast();
  const bundle = useAsync(async () => {
    const [report, forecast] = await Promise.all([api.MonthReport(month), api.Forecast(shiftMonth(month, 1))]);
    return { report, forecast };
  }, [month]);

  const [exporting, setExporting] = useState<'aggregate' | 'patient' | null>(null);
  const [patientExportOpen, setPatientExportOpen] = useState(false);
  const [vaccineReportId, setVaccineReportId] = useState<number | null>(null);
  const reportRef = useRef<HTMLDivElement>(null);
  const [filters, setFilters] = useState<ReportFilters>(defaultReportFilters);

  const report = bundle.data?.report;
  const forecast = bundle.data?.forecast;
  const daily = report?.daily ?? [];
  const perVaccine = useMemo(() => report?.perVaccine ?? [], [report]);
  const forecastRows = useMemo(() => forecast?.rows ?? [], [forecast]);

  const now = new Date();
  const elapsedDays =
    month === currentMonth()
      ? Math.min(now.getDate(), daily.length)
      : month < currentMonth()
        ? daily.length
        : 0;
  const averageDaily = elapsedDays > 0 && report ? report.totals.usageTotal / elapsedDays : 0;
  const daysOfStock =
    report && averageDaily > 0 ? Math.floor(report.totals.remaining / averageDaily) : null;
  const topVaccine = [...perVaccine].sort((a, b) => b.usageTotal - a.usageTotal)[0];
  const healthyStock = report
    ? Math.max(
        0,
        report.totals.activeVaccines -
          report.totals.lowStockCount -
          report.totals.uninitializedActive,
      )
    : 0;
  const lowStockRows = perVaccine
    .filter((row) => row.lowStock)
    .sort((a, b) => a.remaining - b.remaining);
  const forecastById = new Map(forecastRows.map((row) => [row.vaccineId, row]));
  const suggestedRequests = forecastRows.reduce((sum, row) => sum + row.suggestedRequest, 0);

  const visiblePerVaccine = useMemo(() => filterVaccineStats(perVaccine, filters), [perVaccine, filters]);
  const visibleIds = useMemo(
    () => new Set(visiblePerVaccine.map((row) => row.vaccineId)),
    [visiblePerVaccine],
  );
  const visibleForecast = useMemo(
    () => filterForecastRows(forecastRows, visibleIds),
    [forecastRows, visibleIds],
  );
  const filtersActive = !isDefaultFilters(filters);
  const filterSummary = useMemo(() => {
    const parts: string[] = [];
    if (filters.query.trim()) {
      parts.push(`بحث: ${filters.query.trim()}`);
    }
    if (filters.status !== 'all') {
      parts.push(reportStatusLabels[filters.status]);
    }
    if (filters.hideZeroUsage) {
      parts.push('إخفاء بلا استخدام');
    }
    if (filters.sort !== 'usage_desc') {
      parts.push(`الترتيب: ${reportSortLabels[filters.sort]}`);
    }
    return parts.join(' · ') || 'كل اللقاحات';
  }, [filters]);

  const setStatus = (status: ReportStatusFilter) => setFilters((current) => ({ ...current, status }));
  const setSort = (sort: ReportSort) => setFilters((current) => ({ ...current, sort }));

  const printReport = () => {
    if (!report) {
      return;
    }
    printReportNode(reportRef.current, { title: `VacTrack-${month}-report`, orientation: 'landscape' });
  };

  const exportAggregate = async () => {
    setExporting('aggregate');
    try {
      const result = await api.ExportAggregateCSV(month);
      if (result.ok) {
        notify(result.message);
      } else if (result.code !== 'cancelled') {
        notify(result.message || 'تعذر تصدير ملف الإجمالي.', 'error');
      }
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setExporting(null);
    }
  };

  const exportPatient = async () => {
    setExporting('patient');
    try {
      const result = await api.ExportPatientUsageCSV(month);
      if (result.ok) {
        notify(result.message);
        setPatientExportOpen(false);
      } else if (result.code !== 'cancelled') {
        notify(result.message || 'تعذر تصدير ملف CSV على مستوى المريض.', 'error');
      }
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setExporting(null);
    }
  };

  return (
    <>
      <PageHeader className="no-print" title={`التقارير · ${monthLabel(month)}`}>
        <IconButton label="تحديث التقارير" onClick={bundle.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="outline" icon={<Printer className="h-4 w-4" />} onClick={printReport} disabled={!report}>
          طباعة / حفظ PDF
        </Button>
        <Button
          variant="outline"
          icon={<FileSpreadsheet className="h-4 w-4" />}
          loading={exporting === 'aggregate'}
          onClick={exportAggregate}
        >
          CSV الإجمالي
        </Button>
        <Button
          variant="outline"
          icon={<ShieldAlert className="h-4 w-4" />}
          onClick={() => setPatientExportOpen(true)}
        >
          CSV سري
        </Button>
      </PageHeader>

      <InlineError message={bundle.error} />

      {bundle.loading && !bundle.data ? (
        <Card>
          <Spinner label="جارٍ حساب الإحصائيات المحلية…" />
        </Card>
      ) : report ? (
        <div ref={reportRef} className="print-report space-y-4">
          <header className="print-only print-header">
            <div className="flex items-center justify-between gap-6 border-b-2 border-emerald-700 pb-3">
              <div className="flex items-center gap-3">
                <span className="print-brand-mark">V</span>
                <div className="leading-tight">
                  <p className="text-base font-bold tracking-tight text-slate-900">VacTrack</p>
                  <p className="text-[11px] text-slate-500">إدارة مخزون اللقاحات</p>
                </div>
              </div>
              <div className="text-end">
                <h1 className="text-xl font-bold text-slate-900">تقرير المخزون والاستخدام الشهري</h1>
                <p className="mt-1 text-sm text-slate-700">
                  {monthLabel(month)} · {report.state === 'closed' ? 'شهر مغلق' : 'شهر مفتوح'}
                </p>
              </div>
            </div>
            <div className="mt-2 grid grid-cols-3 gap-3 text-[10.5px] leading-5 text-slate-600">
              <p>
                <span className="font-semibold text-slate-800">أُنشئ: </span>
                {formatTimestamp(report.generatedAt)}
              </p>
              <p className="text-center">
                <span className="font-semibold text-slate-800">اللقاحات: </span>
                {formatInteger(visiblePerVaccine.length)} من {formatInteger(perVaccine.length)}
              </p>
              <p className="text-end">
                <span className="font-semibold text-slate-800">الفلاتر: </span>
                {filterSummary}
              </p>
            </div>
          </header>

          <Card className="no-print overflow-hidden">
            <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 border-b border-slate-700/60 px-4 py-3">
              <div className="flex items-center gap-2">
                <SlidersHorizontal className="h-4 w-4 text-slate-500" aria-hidden />
                <h2 className="text-sm font-semibold text-slate-100">تصفية الجدولين</h2>
                <Badge tone={filtersActive ? 'amber' : 'slate'}>
                  {filtersActive ? 'مفعّلة' : 'كل اللقاحات'}
                </Badge>
              </div>
              <p className="text-[11px] text-slate-500">
                عرض {formatInteger(visiblePerVaccine.length)} من {formatInteger(perVaccine.length)} لقاح
              </p>
            </div>

            <div className="grid gap-3 p-4 sm:grid-cols-2 xl:grid-cols-[minmax(0,1.7fr)_170px_190px_auto]">
              <Field label="بحث" htmlFor="report-filter-query">
                <div className="relative">
                  <Search
                    className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500"
                    aria-hidden
                  />
                  <Input
                    id="report-filter-query"
                    value={filters.query}
                    onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
                    placeholder="اسم اللقاح أو الرمز"
                    className="ps-9 pe-9"
                  />
                  {filters.query ? (
                    <IconButton
                      label="مسح البحث"
                      onClick={() => setFilters((current) => ({ ...current, query: '' }))}
                      className="absolute end-1 top-1/2 -translate-y-1/2"
                    >
                      <X className="h-3.5 w-3.5" />
                    </IconButton>
                  ) : null}
                </div>
              </Field>
              <Field label="الحالة" htmlFor="report-filter-status">
                <Select
                  id="report-filter-status"
                  value={filters.status}
                  onChange={(event) => setStatus(event.target.value as ReportStatusFilter)}
                >
                  {(Object.keys(reportStatusLabels) as ReportStatusFilter[]).map((status) => (
                    <option key={status} value={status}>
                      {reportStatusLabels[status]}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field label="الترتيب" htmlFor="report-filter-sort">
                <Select
                  id="report-filter-sort"
                  value={filters.sort}
                  onChange={(event) => setSort(event.target.value as ReportSort)}
                >
                  {(Object.keys(reportSortLabels) as ReportSort[]).map((sort) => (
                    <option key={sort} value={sort}>
                      {reportSortLabels[sort]}
                    </option>
                  ))}
                </Select>
              </Field>
              <div className="flex flex-wrap items-end gap-2">
                <label className="flex h-9 cursor-pointer items-center gap-2 rounded-lg border border-slate-700/80 bg-slate-950 px-3 text-xs text-slate-300 transition-colors hover:border-slate-600">
                  <input
                    type="checkbox"
                    checked={filters.hideZeroUsage}
                    onChange={(event) =>
                      setFilters((current) => ({ ...current, hideZeroUsage: event.target.checked }))
                    }
                    className="h-4 w-4 rounded border-slate-600 accent-emerald-500"
                  />
                  إخفاء بلا استخدام
                </label>
                <Button
                  variant="ghost"
                  icon={<X className="h-3.5 w-3.5" />}
                  onClick={() => setFilters(defaultReportFilters)}
                  disabled={!filtersActive}
                >
                  مسح الفلاتر
                </Button>
              </div>
            </div>

            {filtersActive ? (
              <div className="flex flex-wrap items-center gap-1.5 border-t border-slate-700/60 px-4 py-2.5">
                <span className="text-[11px] text-slate-500">الفلاتر النشطة</span>
                {filters.query.trim() ? (
                  <FilterChip
                    label={`بحث: ${filters.query.trim()}`}
                    onRemove={() => setFilters((current) => ({ ...current, query: '' }))}
                  />
                ) : null}
                {filters.status !== 'all' ? (
                  <FilterChip
                    label={reportStatusLabels[filters.status]}
                    onRemove={() => setFilters((current) => ({ ...current, status: 'all' }))}
                  />
                ) : null}
                {filters.hideZeroUsage ? (
                  <FilterChip
                    label="إخفاء بلا استخدام"
                    onRemove={() => setFilters((current) => ({ ...current, hideZeroUsage: false }))}
                  />
                ) : null}
                {filters.sort !== 'usage_desc' ? (
                  <FilterChip
                    label={`الترتيب: ${reportSortLabels[filters.sort]}`}
                    onRemove={() => setFilters((current) => ({ ...current, sort: 'usage_desc' }))}
                  />
                ) : null}
              </div>
            ) : null}

            <p className="border-t border-slate-700/60 px-4 py-2 text-[11px] leading-5 text-slate-500">
              تؤثر الفلاتر على جدول اللقاحات وجدول تخطيط الطلب. الملخص والرسوم تعرض إجمالي الشهر. لتصدير PDF
              اختر «الطباعة إلى ملف» في نافذة الطباعة.
            </p>
          </Card>

          <section className="report-summary overflow-hidden rounded-xl border border-slate-700/70 bg-slate-950 shadow-panel">
            <div className="report-summary-main grid lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)_minmax(0,1fr)]">
              <div className="border-b border-slate-700/60 p-5 lg:border-b-0 lg:border-e">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-[11px] font-medium text-slate-500">الجرعات المستخدمة هذا الشهر</p>
                  <StatusBadge state={report.state} />
                </div>
                <div className="mt-3 flex flex-wrap items-end gap-x-4 gap-y-2">
                  <p className="report-summary-lead text-[46px] font-semibold leading-none tabular-nums text-emerald-300">
                    {formatInteger(report.totals.usageTotal)}
                  </p>
                  {report.hasComparison ? (
                    <div className="pb-1">
                      <span
                        className={cn(
                          'inline-flex items-center gap-1 rounded-md px-1.5 py-0.5 text-xs font-semibold tabular-nums',
                          report.usageChange > 0
                            ? 'bg-amber-500/10 text-amber-300'
                            : report.usageChange < 0
                              ? 'bg-emerald-500/10 text-emerald-300'
                              : 'bg-slate-800 text-slate-400',
                        )}
                      >
                        {report.usageChange > 0 ? (
                          <TrendingUp className="h-3.5 w-3.5" aria-hidden />
                        ) : report.usageChange < 0 ? (
                          <TrendingDown className="h-3.5 w-3.5" aria-hidden />
                        ) : null}
                        {formatSigned(report.usageChange)}
                      </span>
                      <p className="mt-1 text-[11px] text-slate-500">
                        مقابل {monthLabel(report.previousMonth)}: {formatInteger(report.previousUsage)}
                        {report.changePercentAvailable ? ` · ${formatPercent(report.changePercent)}` : ''}
                      </p>
                    </div>
                  ) : (
                    <p className="pb-1 text-[11px] text-slate-500">لا توجد مقارنة بالشهر السابق</p>
                  )}
                </div>
                <p className="mt-3 text-[11px] leading-5 text-slate-500">
                  أُنشئ {formatTimestamp(report.generatedAt)} · {formatInteger(report.totals.activeVaccines)} لقاحاً
                  نشطاً · {formatInteger(report.adjustmentCount)} تعديل مدقق
                </p>
              </div>

              <div className="border-b border-slate-700/60 p-5 lg:border-b-0 lg:border-e">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-[11px] font-medium text-slate-500">المخزون المتبقي</p>
                  <Package className="h-4 w-4 text-slate-500" aria-hidden />
                </div>
                <p className="report-summary-secondary mt-3 text-[32px] font-semibold leading-none tabular-nums text-sky-300">
                  {formatInteger(report.totals.remaining)}
                </p>
                <p className="mt-2 text-[11px] leading-5 text-slate-500">
                  عبر {formatInteger(report.totals.initializedVaccines)} لقاحاً مهيأ لهذا الشهر
                </p>
              </div>

              <div className="p-5">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-[11px] font-medium text-slate-500">مخزون منخفض</p>
                  <ShieldAlert
                    className={cn('h-4 w-4', report.totals.lowStockCount > 0 ? 'text-amber-400' : 'text-slate-500')}
                    aria-hidden
                  />
                </div>
                <p
                  className={cn(
                    'report-summary-secondary mt-3 text-[32px] font-semibold leading-none tabular-nums',
                    report.totals.lowStockCount > 0 ? 'text-amber-300' : 'text-emerald-300',
                  )}
                >
                  {formatInteger(report.totals.lowStockCount)}
                </p>
                <p className="mt-2 text-[11px] leading-5 text-slate-500">
                  {report.totals.lowStockCount > 0 ? 'لقاحات عند الحد أو أقل' : 'كل اللقاحات فوق الحد'}
                </p>
              </div>
            </div>

            <div className="report-summary-metrics grid grid-cols-2 gap-px border-t border-slate-700/60 bg-slate-700/60 lg:grid-cols-4">
              <SummaryMetric
                label="متوسط الاستخدام اليومي"
                value={averageDaily > 0 ? formatInteger(Math.round(averageDaily)) : '—'}
                detail={elapsedDays > 0 ? `خلال ${formatInteger(elapsedDays)} يوماً` : 'لم يبدأ الشهر بعد'}
                icon={<Activity />}
              />
              <SummaryMetric
                label="تغطية المخزون"
                value={daysOfStock === null ? '—' : `${formatInteger(daysOfStock)} يوم`}
                detail={averageDaily > 0 ? 'بالمعدل اليومي الحالي' : 'لا يوجد استخدام كافٍ للتقدير'}
                tone={
                  daysOfStock !== null && daysOfStock <= 7
                    ? 'rose'
                    : daysOfStock !== null && daysOfStock <= 14
                      ? 'amber'
                      : 'default'
                }
                icon={<CalendarClock />}
              />
              <SummaryMetric
                label="أعلى استخدام"
                value={topVaccine ? formatInteger(topVaccine.usageTotal) : '—'}
                detail={topVaccine ? topVaccine.name : 'لا يوجد استخدام'}
                tone="sky"
                icon={<Trophy />}
              />
              <SummaryMetric
                label="طلب الشهر القادم"
                value={formatInteger(suggestedRequests)}
                detail={forecast ? `المقترح لشهر ${monthLabel(forecast.targetMonth)}` : 'لا تتوفر توقعات'}
                tone="emerald"
                icon={<Package />}
              />
            </div>
          </section>

          <div className="report-charts grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <Card>
              <CardHeader
                title="الاتجاه اليومي"
                description="الجرعات المسجلة يوماً بيوم خلال الشهر."
                action={<Badge tone="emerald">{formatInteger(report.totals.usageTotal)} جرعة</Badge>}
              />
              <div className="px-3 py-4">
                <DailyUsageChart data={daily} />
              </div>
            </Card>

            <Card>
              <CardHeader
                title="صحة المخزون"
                description="توزيع اللقاحات النشطة حسب حالة المخزون."
                action={<Badge tone="slate">{formatInteger(report.totals.activeVaccines)} لقاح</Badge>}
              />
              <StockHealthChart
                healthy={healthyStock}
                low={report.totals.lowStockCount}
                uninitialized={report.totals.uninitializedActive}
              />
              <div className="border-t border-slate-700/60 px-4 py-2.5 text-[11px] text-slate-500">
                {report.totals.initializedVaccines} من {report.totals.activeVaccines} لقاحاً مهيأ لهذا الشهر
              </div>
            </Card>
          </div>

          <div className="report-charts grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <Card>
              <CardHeader
                title="الاستخدام حسب اللقاح · الحالي مقابل السابق"
                description="أعلى ثمانية لقاحات استخداماً هذا الشهر."
              />
              <UsageComparisonChart rows={perVaccine} />
            </Card>

            <Card>
              <CardHeader
                title="تنبيهات المخزون"
                description="اللقاحات عند الحد المنخفض أو أقل."
                action={
                  <Badge tone={lowStockRows.length > 0 ? 'amber' : 'emerald'}>
                    {formatInteger(lowStockRows.length)}
                  </Badge>
                }
              />
              {lowStockRows.length === 0 ? (
                <EmptyState
                  icon={<ShieldAlert className="h-5 w-5" />}
                  title="لا توجد لقاحات منخفضة المخزون"
                />
              ) : (
                <div className="divide-y divide-slate-800/70">
                  {lowStockRows.slice(0, 8).map((row) => {
                    const request = forecastById.get(row.vaccineId);
                    return (
                      <div key={row.vaccineId} className="flex items-center gap-3 px-4 py-2.5">
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-sm text-slate-100">{row.name}</p>
                          <p className="text-[11px] text-slate-500">
                            المتبقي {formatInteger(row.remaining)} {row.unit}
                          </p>
                        </div>
                        {request && request.suggestedRequest > 0 ? (
                          <Badge tone="emerald">اطلب {formatInteger(request.suggestedRequest)}</Badge>
                        ) : (
                          <Badge tone="amber">منخفض</Badge>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
              <div className="border-t border-slate-700/60 px-4 py-2.5 text-[11px] text-slate-500">
                الحد المنخفض {formatInteger(forecast?.lowStockThreshold ?? 0)} جرعة
              </div>
            </Card>
          </div>

          <Card>
            <CardHeader
              title="الاستخدام حسب اللقاح"
              description="الافتتاحي والوارد والاستخدام والمتبقي لكل لقاح."
              action={
                <Badge tone={filtersActive ? 'amber' : 'slate'}>
                  {formatInteger(visiblePerVaccine.length)} من {formatInteger(perVaccine.length)}
                </Badge>
              }
            />
            {visiblePerVaccine.length === 0 ? (
              <EmptyState
                icon={<BarChart3 className="h-5 w-5" />}
                title={perVaccine.length === 0 ? 'لا توجد إحصائيات لقاحات لهذا الشهر' : 'لا توجد لقاحات تطابق الفلاتر'}
                description={perVaccine.length === 0 ? undefined : 'غيّر البحث أو الحالة أو امسح الفلاتر.'}
              />
            ) : (
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>اللقاح</th>
                      <th className="text-end">الافتتاحي</th>
                      <th className="text-end">الوارد</th>
                      <th className="text-end">الاستخدام</th>
                      <th className="text-end">السابق</th>
                      <th className="text-end">التغير</th>
                      <th className="text-end">المتبقي</th>
                      <th>تقرير</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visiblePerVaccine.map((vaccine) => (
                      <tr key={vaccine.vaccineId}>
                        <td>
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="font-medium text-slate-100">{vaccine.name}</span>
                            {vaccine.lowStock ? <Badge tone="amber">منخفض</Badge> : null}
                            {vaccine.archived ? <Badge tone="slate">مؤرشف</Badge> : null}
                            {!vaccine.initialized ? <Badge tone="rose">لا يوجد صف مخزون</Badge> : null}
                          </div>
                          <p className="mt-0.5 text-[11px] text-slate-500">
                            {vaccine.code || 'بدون رمز'} · {vaccine.unit}
                          </p>
                        </td>
                        <td className="text-end tabular-nums text-slate-300">
                          {formatInteger(vaccine.opening)}
                        </td>
                        <td className="text-end tabular-nums text-slate-300">
                          {formatInteger(vaccine.received)}
                        </td>
                        <td className="text-end font-semibold tabular-nums text-emerald-300">
                          {formatInteger(vaccine.usageTotal)}
                        </td>
                        <td className="text-end tabular-nums text-slate-400">
                          {formatInteger(vaccine.previousUsage)}
                        </td>
                        <td
                          className={cn(
                            'text-end font-medium tabular-nums',
                            vaccine.usageChange > 0
                              ? 'text-amber-300'
                              : vaccine.usageChange < 0
                                ? 'text-emerald-300'
                                : 'text-slate-400',
                          )}
                        >
                          {report.hasComparison ? formatSigned(vaccine.usageChange) : '—'}
                        </td>
                        <td
                          className={cn(
                            'text-end font-semibold tabular-nums',
                            vaccine.lowStock ? 'text-amber-300' : 'text-slate-100',
                          )}
                        >
                          {formatInteger(vaccine.remaining)}
                        </td>
                        <td>
                          <Button variant="ghost" size="sm" onClick={() => setVaccineReportId(vaccine.vaccineId)}>
                            عرض
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          {forecast ? (
            <Card>
              <CardHeader
                title={`تخطيط الطلب · ${monthLabel(forecast.targetMonth)}`}
                description="تقدير الطلب من متوسط الأشهر المغلقة. ليس توصية طبية."
                action={<Badge tone="slate">{formatInteger(visibleForecast.length)} لقاح</Badge>}
              />
              {visibleForecast.length === 0 ? (
                <EmptyState
                  icon={<TrendingUp className="h-5 w-5" />}
                  title={forecast.rows.length === 0 ? 'لا توجد صفوف توقعات' : 'لا توجد توقعات تطابق الفلاتر'}
                  description={forecast.rows.length === 0 ? undefined : 'غيّر البحث أو الحالة أو امسح الفلاتر.'}
                />
              ) : (
                <div className="table-scroll">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>اللقاح</th>
                        <th>الطريقة</th>
                        <th className="text-end">العينة</th>
                        <th className="text-end">الطلب المتوقع</th>
                        <th className="text-end">اقتراح الافتتاحي</th>
                        <th className="text-end">الطلب المقترح</th>
                      </tr>
                    </thead>
                    <tbody>
                      {visibleForecast.map((row) => (
                        <tr key={row.vaccineId}>
                          <td>
                            <span className="font-medium text-slate-100">{row.name}</span>
                            <p className="mt-0.5 text-[11px] text-slate-500">
                              {row.code || 'بدون رمز'} · {row.unit}
                            </p>
                          </td>
                          <td className="text-slate-400">{methodLabel(row.method)}</td>
                          <td className="text-end tabular-nums text-slate-400">
                            {row.sampleCount > 0 ? sampleMonthsLabel(row.sampleCount) : '—'}
                          </td>
                          <td className="text-end tabular-nums">
                            {formatInteger(row.estimatedDemand)}
                          </td>
                          <td className="text-end tabular-nums text-slate-400">
                            {formatInteger(row.openingSuggestion)}
                          </td>
                          <td className="text-end font-semibold tabular-nums text-emerald-300">
                            {formatInteger(row.suggestedRequest)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <div className="border-t border-slate-700/60 px-4 py-2.5 text-[11px] text-slate-500">
                حد المخزون المنخفض {formatInteger(forecast.lowStockThreshold)} جرعة · أُنشئ{' '}
                {formatTimestamp(forecast.generatedAt)}
              </div>
            </Card>
          ) : null}

          <footer className="print-only print-footer">
            <span>تقرير محلي سري · VacTrack · {monthLabel(month)}</span>
            <span>لا تشارك بيانات المرضى خارج محطة العمل المعتمدة.</span>
          </footer>
        </div>
      ) : null}

      <VaccineReportModal
        open={vaccineReportId !== null}
        vaccineId={vaccineReportId}
        month={month}
        onClose={() => setVaccineReportId(null)}
      />

      <ConfirmDialog
        open={patientExportOpen}
        title="تصدير CSV سري على مستوى المريض؟"
        message={
          <div className="space-y-2">
            <p>
              يتضمن هذا الملف أسماء المرضى ورموزهم وأحداث إعطاء فردية لشهر {monthLabel(month)}. تُحذف
              ملاحظات الاستخدام.
            </p>
            <Alert tone="danger" title="بيانات محلية سرية">
              خزّن التصدير فقط على محطة عمل محمية أو تخزين محلي معتمد. لا ترفعه إلى الخدمة العامة، ولا
              ترسله بالبريد الإلكتروني، ولا تضمنه في التحكم بالمصدر.
            </Alert>
          </div>
        }
        confirmLabel="تصدير CSV السري"
        tone="danger"
        busy={exporting === 'patient'}
        onConfirm={exportPatient}
        onClose={() => setPatientExportOpen(false)}
      />
    </>
  );
}

function VaccineReportModal({
  open,
  vaccineId,
  month,
  onClose,
}: {
  open: boolean;
  vaccineId: number | null;
  month: string;
  onClose: () => void;
}) {
  const bundle = useAsync(async () => {
    if (!open || !vaccineId) {
      return null;
    }
    const [report, overview, usage, forecast] = await Promise.all([
      api.MonthReport(month),
      api.MonthOverview(month),
      api.UsageForVaccineMonth(vaccineId, month),
      api.Forecast(shiftMonth(month, 1)),
    ]);
    return { report, overview, usage, forecast };
  }, [open, vaccineId, month]);

  const contentRef = useRef<HTMLDivElement>(null);
  const vaccine = bundle.data?.report.perVaccine.find((item) => item.vaccineId === vaccineId);
  const row = bundle.data?.overview.rows.find((item) => item.vaccineId === vaccineId);
  const forecastRow = bundle.data?.forecast.rows.find((item) => item.vaccineId === vaccineId);
  const adjustments = (bundle.data?.overview.adjustments ?? []).filter((item) => item.vaccineId === vaccineId);
  const daily = (row?.dailyUsage ?? []).map((total, index) => ({
    date: `${month}-${String(index + 1).padStart(2, '0')}`,
    day: index + 1,
    total,
  }));

  const printVaccineReport = () => {
    printReportNode(contentRef.current, {
      title: `VacTrack-${month}-${(vaccine?.code || vaccine?.name || 'vaccine').replace(/\s+/g, '-')}-report`,
      orientation: 'portrait',
    });
  };

  return (
    <Modal open={open} onClose={onClose} title={vaccine?.name ?? 'تقرير اللقاح'} size="xl">
      {bundle.loading && !bundle.data ? (
        <Spinner label="جارٍ تحميل تقرير اللقاح…" />
      ) : bundle.error ? (
        <InlineError message={bundle.error} />
      ) : !vaccine || !row ? (
        <EmptyState icon={<BarChart3 className="h-5 w-5" />} title="لا توجد بيانات لهذا اللقاح في هذا الشهر" />
      ) : (
        <div ref={contentRef} className="vaccine-report space-y-4">
          <header className="print-only print-header">
            <div className="flex items-center justify-between gap-6 border-b-2 border-emerald-700 pb-3">
              <div className="flex items-center gap-3">
                <span className="print-brand-mark">V</span>
                <div className="leading-tight">
                  <p className="text-base font-bold tracking-tight text-slate-900">VacTrack</p>
                  <p className="text-[11px] text-slate-500">إدارة مخزون اللقاحات</p>
                </div>
              </div>
              <div className="text-end">
                <h1 className="text-xl font-bold text-slate-900">تقرير اللقاح الشهري</h1>
                <p className="mt-1 text-sm text-slate-700">{monthLabel(month)}</p>
              </div>
            </div>
            <div className="mt-2 grid grid-cols-3 gap-3 text-[10.5px] leading-5 text-slate-600">
              <p>
                <span className="font-semibold text-slate-800">اللقاح: </span>
                {vaccine.name}
              </p>
              <p className="text-center">
                <span className="font-semibold text-slate-800">الرمز: </span>
                {vaccine.code || 'بدون رمز'} · {vaccine.unit}
              </p>
              <p className="text-end">
                <span className="font-semibold text-slate-800">أُنشئ: </span>
                {formatTimestamp(bundle.data?.report.generatedAt ?? '')}
              </p>
            </div>
          </header>

          <div className="no-print flex flex-wrap items-center justify-between gap-2 rounded-xl border border-slate-700/60 bg-slate-900/60 px-3.5 py-2.5">
            <p className="text-[11.5px] leading-5 text-slate-500">
              تقرير فردي لهذا اللقاح. اطبع أو احفظ PDF بمقاس A4 عمودي.
            </p>
            <Button
              variant="outline"
              size="sm"
              icon={<Printer className="h-4 w-4" />}
              onClick={printVaccineReport}
            >
              طباعة / حفظ PDF
            </Button>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <Badge tone={row.archived ? 'slate' : 'emerald'}>{row.archived ? 'مؤرشف' : 'نشط'}</Badge>
            <Badge tone={row.initialized ? 'sky' : 'rose'}>{row.initialized ? 'مخزون مهيأ' : 'غير مهيأ'}</Badge>
            <Badge tone={row.lowStock ? 'amber' : 'emerald'}>{row.lowStock ? 'مخزون منخفض' : 'مخزون كافٍ'}</Badge>
            <span className="text-xs text-slate-500">
              {vaccine.code || 'بدون رمز'} · {vaccine.unit} · {monthLabel(month)}
            </span>
          </div>

          <div className="print-stat-grid grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatCard label="الافتتاحي" value={formatInteger(row.openingBalance)} detail="بداية الشهر" />
            <StatCard label="الوارد" value={formatInteger(row.stockReceived)} detail="المضاف هذا الشهر" />
            <StatCard label="الاستخدام" value={formatInteger(row.usageTotal)} detail="الجرعات المستخدمة" tone="emerald" />
            <StatCard label="المتبقي" value={formatInteger(row.remaining)} detail="الرصيد الحالي" tone="sky" />
          </div>

          <div className="print-stat-grid grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatCard
              label="الشهر السابق"
              value={formatInteger(vaccine.previousUsage)}
              detail={bundle.data?.report.hasComparison ? 'للمقارنة' : 'لا توجد مقارنة'}
            />
            <StatCard
              label="التغير"
              value={bundle.data?.report.hasComparison ? formatSigned(vaccine.usageChange) : '—'}
              detail="الحالي مقابل السابق"
              tone={vaccine.usageChange > 0 ? 'amber' : 'default'}
            />
            <StatCard
              label="الطلب المتوقع"
              value={formatInteger(forecastRow?.estimatedDemand ?? 0)}
              detail={forecastRow ? methodLabel(forecastRow.method) : 'لا تتوفر توقعات'}
            />
            <StatCard
              label="الطلب المقترح"
              value={formatInteger(forecastRow?.suggestedRequest ?? 0)}
              detail={forecastRow ? `اقتراح افتتاحي ${formatInteger(forecastRow.openingSuggestion)}` : '—'}
              tone="emerald"
            />
          </div>

          <Card>
            <CardHeader title="الاتجاه اليومي" action={<Badge tone="emerald">{formatInteger(row.usageTotal)} جرعة</Badge>} />
            <div className="px-3 py-4">
              <DailyUsageChart data={daily as app.DailyTotalDTO[]} />
            </div>
          </Card>

          <Card>
            <CardHeader title="الاستخدام حسب المريض" />
            {(bundle.data?.usage ?? []).length === 0 ? (
              <EmptyState icon={<BarChart3 className="h-5 w-5" />} title="لا توجد جرعات مسجلة لهذا اللقاح" />
            ) : (
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>التاريخ</th>
                      <th>المريض</th>
                      <th className="text-end">الجرعات</th>
                      <th>الحالة</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(bundle.data?.usage ?? []).map((entry) => (
                      <tr key={entry.id} className={entry.voided ? 'opacity-60' : undefined}>
                        <td className="whitespace-nowrap">{formatDate(entry.administeredOn)}</td>
                        <td>
                          {entry.patientName}
                          {entry.patientCode ? <span className="ms-1.5 text-[11px] text-slate-500">{entry.patientCode}</span> : null}
                        </td>
                        <td className="text-end font-medium tabular-nums">{formatInteger(entry.quantity)}</td>
                        <td>
                          {entry.voided ? <Badge tone="rose">ملغى</Badge> : <Badge tone="emerald">مسجل</Badge>}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          {adjustments.length > 0 ? (
            <Card>
              <CardHeader title="التعديلات" />
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>التاريخ</th>
                      <th className="text-end">المبلغ</th>
                      <th className="text-end">قبل</th>
                      <th className="text-end">بعد</th>
                      <th>السبب</th>
                    </tr>
                  </thead>
                  <tbody>
                    {adjustments.map((adjustment) => (
                      <tr key={adjustment.id}>
                        <td className="whitespace-nowrap">{formatTimestamp(adjustment.createdAt)}</td>
                        <td className="text-end font-medium tabular-nums">{formatInteger(adjustment.amount)}</td>
                        <td className="text-end tabular-nums text-slate-400">{formatInteger(adjustment.balanceBefore)}</td>
                        <td className="text-end tabular-nums text-slate-400">{formatInteger(adjustment.balanceAfter)}</td>
                        <td className="text-slate-400">{adjustment.reason || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>
          ) : null}

          <footer className="print-only print-footer">
            <span>تقرير لقاح فردي · VacTrack · {monthLabel(month)}</span>
            <span>بيانات محلية سرية. لا تشاركها خارج محطة العمل المعتمدة.</span>
          </footer>
        </div>
      )}
    </Modal>
  );
}
