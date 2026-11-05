import { useState } from 'react';
import {
  BarChart3,
  CalendarRange,
  Download,
  FileSpreadsheet,
  RefreshCw,
  ShieldAlert,
  TrendingDown,
  TrendingUp,
} from 'lucide-react';
import { api, errorMessage } from '../lib/api';
import { cn } from '../components/ui';
import { formatInteger, formatPercent, formatSigned, formatTimestamp, monthLabel } from '../lib/format';
import { useAsync } from '../lib/hooks';
import { useMonth } from '../components/MonthContext';
import { useToast } from '../components/Toast';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  EmptyState,
  IconButton,
  InlineError,
  PageHeader,
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

export function ReportsPage() {
  const { month } = useMonth();
  const { notify } = useToast();
  const bundle = useAsync(async () => {
    const [report, forecast] = await Promise.all([api.MonthReport(month), api.Forecast(month)]);
    return { report, forecast };
  }, [month]);

  const [exporting, setExporting] = useState<'aggregate' | 'patient' | null>(null);
  const [patientExportOpen, setPatientExportOpen] = useState(false);

  const report = bundle.data?.report;
  const forecast = bundle.data?.forecast;
  const daily = report?.daily ?? [];
  const maxDaily = Math.max(1, ...daily.map((point) => point.total));

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
      <PageHeader
        title={`التقارير · ${monthLabel(month)}`}
        description="إحصائيات تشغيلية محلية. تصدير الإجمالي آمن للاستخدام الداخلي؛ أما التصدير على مستوى المريض فسري ويجب أن يبقى على محطة العمل هذه."
      >
        <IconButton label="تحديث التقارير" onClick={bundle.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="outline" icon={<FileSpreadsheet className="h-4 w-4" />} loading={exporting === 'aggregate'} onClick={exportAggregate}>
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
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge state={report.state} />
            <span className="text-xs text-slate-500">
              {formatInteger(report.totals.activeVaccines)} لقاحاً نشطاً · {formatInteger(report.adjustmentCount)} تعديل مدقق
            </span>
          </div>

          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatCard label="الجرعات المستخدمة" value={formatInteger(report.totals.usageTotal)} detail="الاستخدام الفعلي هذا الشهر" tone="emerald" icon={<BarChart3 className="h-4 w-4" />} />
            <StatCard
              label="التغير عن السابق"
              value={
                report.hasComparison ? (
                  <span className="flex items-baseline gap-1.5">
                    {formatSigned(report.usageChange)}
                    {report.usageChange <= 0 ? (
                      <TrendingDown className="h-4 w-4 text-emerald-300" aria-hidden />
                    ) : (
                      <TrendingUp className="h-4 w-4 text-amber-300" aria-hidden />
                    )}
                  </span>
                ) : (
                  '—'
                )
              }
              detail={
                report.hasComparison
                  ? `${monthLabel(report.previousMonth)}: ${formatInteger(report.previousUsage)}${
                      report.changePercentAvailable ? ` · ${formatPercent(report.changePercent)}` : ''
                    }`
                  : 'لا توجد مقارنة بالشهر السابق'
              }
              tone={report.hasComparison && report.usageChange > 0 ? 'amber' : 'default'}
              icon={<CalendarRange className="h-4 w-4" />}
            />
            <StatCard label="المخزون المتبقي" value={formatInteger(report.totals.remaining)} detail="عبر اللقاحات المهيأة" tone="sky" icon={<Download className="h-4 w-4" />} />
            <StatCard
              label="مخزون منخفض"
              value={formatInteger(report.totals.lowStockCount)}
              detail="لقاحات عند الحد أو أقل"
              tone={report.totals.lowStockCount > 0 ? 'amber' : 'default'}
              icon={<ShieldAlert className="h-4 w-4" />}
            />
          </div>

          <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <Card>
              <CardHeader
                title="الاستخدام حسب اللقاح"
                description="يستبعد الاستخدام الفعلي القيود الملغاة. تستخدم مقارنات الشهر السابق إجمالي الشهر الكامل."
              />
              {report.perVaccine.length === 0 ? (
                <EmptyState icon={<BarChart3 className="h-5 w-5" />} title="لا توجد إحصائيات لقاحات لهذا الشهر" />
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
                      </tr>
                    </thead>
                    <tbody>
                      {report.perVaccine.map((vaccine) => (
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
                          <td className="text-end tabular-nums text-slate-300">{formatInteger(vaccine.opening)}</td>
                          <td className="text-end tabular-nums text-slate-300">{formatInteger(vaccine.received)}</td>
                          <td className="text-end font-semibold tabular-nums text-emerald-300">{formatInteger(vaccine.usageTotal)}</td>
                          <td className="text-end tabular-nums text-slate-400">{formatInteger(vaccine.previousUsage)}</td>
                          <td
                            className={cn(
                              'text-end font-medium tabular-nums',
                              vaccine.usageChange > 0 ? 'text-amber-300' : vaccine.usageChange < 0 ? 'text-emerald-300' : 'text-slate-400',
                            )}
                          >
                            {report.hasComparison ? formatSigned(vaccine.usageChange) : '—'}
                          </td>
                          <td className="text-end font-semibold tabular-nums">{formatInteger(vaccine.remaining)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>

            <Card>
              <CardHeader title="الاتجاه اليومي" description="الجرعات الفعلية المسجلة يومياً؛ في الأشهر المغلقة والمفتوحة على حد سواء." />
              {daily.length === 0 ? (
                <EmptyState icon={<CalendarRange className="h-5 w-5" />} title="لا توجد بيانات يومية" />
              ) : (
                <div className="max-h-[420px] space-y-1.5 overflow-y-auto px-4 py-4">
                  {daily.map((point) => (
                    <div key={point.date} className="grid grid-cols-[2.25rem_minmax(0,1fr)_2.5rem] items-center gap-2">
                      <span className="text-end text-[11px] tabular-nums text-slate-500">{point.day}</span>
                      <div className="h-2 overflow-hidden rounded-full bg-slate-800">
                        <div
                          className={cn('h-full rounded-full', point.total > 0 ? 'bg-emerald-400/80' : 'bg-transparent')}
                          style={{ width: `${(point.total / maxDaily) * 100}%` }}
                        />
                      </div>
                      <span className={cn('text-end text-[11px] tabular-nums', point.total > 0 ? 'text-slate-300' : 'text-slate-600')}>
                        {formatInteger(point.total)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>

          {forecast ? (
            <Card>
              <CardHeader
                title={`تخطيط الطلب · ${monthLabel(forecast.targetMonth)}`}
                description="أداة تخطيط شفافة تعتمد على الأشهر المغلقة والرصيد المرحّل الحالي. وهي ليست توصية طبية أو طلب شراء."
                action={<Badge tone="slate">{formatInteger(forecast.rows.length)} لقاح</Badge>}
              />
              {forecast.rows.length === 0 ? (
                <EmptyState icon={<TrendingUp className="h-5 w-5" />} title="لا توجد صفوف توقعات" />
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
                      {forecast.rows.map((row) => (
                        <tr key={row.vaccineId}>
                          <td>
                            <span className="font-medium text-slate-100">{row.name}</span>
                            <p className="mt-0.5 text-[11px] text-slate-500">
                              {row.code || 'بدون رمز'} · {row.unit}
                            </p>
                          </td>
                          <td className="text-slate-400">{methodLabel(row.method)}</td>
                          <td className="text-end tabular-nums text-slate-400">
                            {row.sampleCount > 0 ? `${formatInteger(row.sampleCount)} شهر` : '—'}
                          </td>
                          <td className="text-end tabular-nums">{formatInteger(row.estimatedDemand)}</td>
                          <td className="text-end tabular-nums text-slate-400">{formatInteger(row.openingSuggestion)}</td>
                          <td className="text-end font-semibold tabular-nums text-emerald-300">
                            {formatInteger(row.suggestedRequest)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <div className="border-t border-slate-800/70 px-4 py-2.5 text-[11px] text-slate-500">
                حد المخزون المنخفض {formatInteger(forecast.lowStockThreshold)} جرعة · أُنشئ{' '}
                {formatTimestamp(forecast.generatedAt)}
              </div>
            </Card>
          ) : null}
        </div>
      ) : null}

      <ConfirmDialog
        open={patientExportOpen}
        title="تصدير CSV سري على مستوى المريض؟"
        message={
          <div className="space-y-2">
            <p>
              يتضمن هذا الملف أسماء المرضى ورموزهم وأحداث إعطاء فردية لشهر {monthLabel(month)}. تُحذف ملاحظات
              الاستخدام.
            </p>
            <Alert tone="danger" title="بيانات محلية سرية">
              خزّن التصدير فقط على محطة عمل محمية أو تخزين محلي معتمد. لا ترفعه إلى الخدمة العامة، ولا ترسله بالبريد
              الإلكتروني، ولا تضمنه في التحكم بالمصدر.
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
