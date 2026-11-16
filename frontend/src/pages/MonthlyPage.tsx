import { useState } from 'react';
import {
  AlertTriangle,
  Archive,
  Boxes,
  CalendarCheck2,
  CalendarX2,
  Download,
  List,
  Lock,
  PencilLine,
  RefreshCw,
  SlidersHorizontal,
} from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { cn } from '../components/ui';
import { formatDate, formatInteger, formatSigned, formatTimestamp, monthLabel } from '../lib/format';
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
import { AdjustDialog, ReopenMonthDialog, SetupMonthDialog, StockDialog } from '../components/InventoryDialogs';
import { UsageListDialog } from '../components/UsageDialogs';

export function MonthlyPage() {
  const { month } = useMonth();
  const { notify } = useToast();
  const overview = useAsync(() => api.MonthOverview(month), [month]);

  const [usageOpen, setUsageOpen] = useState(false);
  const [usageRow, setUsageRow] = useState<app.MonthRowDTO | null>(null);
  const [usageDate, setUsageDate] = useState<string | undefined>(undefined);
  const [stockRow, setStockRow] = useState<app.MonthRowDTO | null>(null);
  const [adjustRow, setAdjustRow] = useState<app.MonthRowDTO | null>(null);
  const [setupOpen, setSetupOpen] = useState(false);
  const [reopenOpen, setReopenOpen] = useState(false);
  const [closeOpen, setCloseOpen] = useState(false);
  const [closing, setClosing] = useState(false);
  const [exporting, setExporting] = useState(false);

  const data = overview.data;
  const rows = data?.rows ?? [];
  const days = data?.days ?? 0;
  const state = data?.state ?? 'not_initialized';

  const openUsage = (row: app.MonthRowDTO, date?: string) => {
    setUsageRow(row);
    setUsageDate(date);
    setUsageOpen(true);
  };

  const closeUsage = () => {
    setUsageOpen(false);
    setUsageRow(null);
    setUsageDate(undefined);
  };

  const closeMonth = async () => {
    setClosing(true);
    try {
      const result = await runAction(api.CloseMonth(month));
      notify(result.message);
      setCloseOpen(false);
      overview.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setClosing(false);
    }
  };

  const exportAggregate = async () => {
    setExporting(true);
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
      setExporting(false);
    }
  };

  return (
    <>
      <PageHeader
        title={`المخزون الشهري · ${monthLabel(month)}`}
>
        <IconButton label="تحديث الجدول الشهري" onClick={overview.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="outline" icon={<Download className="h-4 w-4" />} loading={exporting} onClick={exportAggregate}>
          تصدير الإجمالي
        </Button>
        {state === 'open' ? (
          <Button variant="outline" icon={<Lock className="h-4 w-4" />} onClick={() => setCloseOpen(true)}>
            إغلاق الشهر
          </Button>
        ) : null}
        {state === 'closed' ? (
          <Button variant="outline" icon={<CalendarX2 className="h-4 w-4" />} onClick={() => setReopenOpen(true)}>
            إعادة فتح الشهر
          </Button>
        ) : null}
        <Button variant="primary" icon={<Boxes className="h-4 w-4" />} onClick={() => setSetupOpen(true)}>
          تهيئة المخزون
        </Button>
      </PageHeader>

      <InlineError message={overview.error} />

      {overview.loading && !data ? (
        <Card>
          <Spinner label="جارٍ بناء الجدول الشهري…" />
        </Card>
      ) : data ? (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge state={data.state} />
            <span className="text-xs text-slate-500">
              تمت تهيئة {data.totals.initializedVaccines} من أصل {data.totals.activeVaccines} لقاحاً نشطاً
            </span>
            {data.lowStockThreshold > 0 ? (
              <span className="text-xs text-slate-500">تنبيه المخزون المنخفض عند ≤ {data.lowStockThreshold} جرعة</span>
            ) : (
              <span className="text-xs text-slate-500">تنبيهات المخزون المنخفض معطلة</span>
            )}
          </div>

          {data.totals.uninitializedActive > 0 ? (
            <Alert tone="warning" title="المخزون الشهري غير مهيأ بالكامل">
              {data.totals.uninitializedActive} لقاح نشط لا يملك صف مخزون صريحاً لهذا الشهر. يُرفض الاستخدام لتلك
              اللقاحات حتى يوجد الصف، حتى لو كان المخزون صفراً.{' '}
              <button type="button" className="underline underline-offset-2 hover:text-slate-100" onClick={() => setSetupOpen(true)}>
                تهيئة المخزون الشهري
              </button>
              .
            </Alert>
          ) : null}

          {data.totals.lowStockCount > 0 ? (
            <Alert tone="warning" title="مخزون منخفض">
              {data.totals.lowStockCount} لقاح عند أو أقل من الحد المكوّن البالغ {data.lowStockThreshold} جرعة.
            </Alert>
          ) : null}

          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <StatCard label="الرصيد الافتتاحي" value={formatInteger(data.totals.openingBalance)} detail="جرعات كاملة مرحّلة" icon={<Boxes className="h-4 w-4" />} />
            <StatCard label="المخزون الوارد" value={formatInteger(data.totals.stockReceived)} detail="التوريدات والتعديلات المدققة" tone="sky" icon={<Archive className="h-4 w-4" />} />
            <StatCard label="الاستخدام الفعلي" value={formatInteger(data.totals.usageTotal)} detail="القيد الملغاة مستبعدة" tone="emerald" icon={<CalendarCheck2 className="h-4 w-4" />} />
            <StatCard
              label="المتبقي"
              value={formatInteger(data.totals.remaining)}
              detail="الافتتاحي + الوارد − الاستخدام"
              tone={data.totals.lowStockCount > 0 ? 'amber' : 'default'}
              icon={<AlertTriangle className="h-4 w-4" />}
            />
          </div>

          <Card>
            <CardHeader
              title="جدول الاستخدام اليومي"
action={<Badge tone="slate">{data.days} يوماً</Badge>}
            />
            {rows.length === 0 ? (
              <EmptyState
                icon={<Boxes className="h-5 w-5" />}
                title="لا توجد لقاحات في الكتالوج"
/>
            ) : (
              <div className="table-scroll">
                <table className="data-table data-table--sticky-first" style={{ minWidth: Math.max(760, 650 + days * 34) }}>
                  <thead>
                    <tr>
                      <th className="min-w-[190px]">اللقاح</th>
                      <th className="text-end">الافتتاحي</th>
                      <th className="text-end">الوارد</th>
                      <th className="text-end">الاستخدام</th>
                      <th className="text-end">المتبقي</th>
                      <th className="text-end">الإجراءات</th>
                      {Array.from({ length: days }, (_, index) => (
                        <th key={index} className="w-[34px] px-1 text-center">
                          {index + 1}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => {
                      const daily = row.dailyUsage ?? [];
                      return (
                        <tr key={row.vaccineId} className={cn(row.archived && 'opacity-60')}>
                          <td>
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span className="font-medium text-slate-100">{row.vaccineName}</span>
                              {row.archived ? <Badge tone="slate">مؤرشف</Badge> : null}
                              {row.lowStock ? <Badge tone="amber">منخفض</Badge> : null}
                              {!row.initialized ? <Badge tone="rose">لا يوجد صف مخزون</Badge> : null}
                            </div>
                            <p className="mt-0.5 text-[11px] text-slate-500">
                              {row.vaccineCode || 'بدون رمز'} · {row.unit}
                              {row.carryoverOverridden ? ' · تم تجاوز الترحيل' : ''}
                            </p>
                          </td>
                          <td className="text-end text-sm tabular-nums">
                            {row.initialized ? formatInteger(row.openingBalance) : '—'}
                          </td>
                          <td className="text-end text-sm tabular-nums">
                            {row.initialized ? formatInteger(row.stockReceived) : '—'}
                          </td>
                          <td className="text-end">
                            {row.initialized ? (
                              <button
                                type="button"
                                onClick={() => openUsage(row)}
                                className="focus-ring rounded-md px-1.5 py-0.5 text-sm font-semibold tabular-nums text-emerald-300 hover:bg-emerald-500/10"
                              >
                                {formatInteger(row.usageTotal)}
                              </button>
                            ) : (
                              <span className="text-slate-500">—</span>
                            )}
                          </td>
                          <td
                            className={cn(
                              'text-end text-sm font-semibold tabular-nums',
                              row.initialized && row.remaining <= 0 && 'text-amber-300',
                            )}
                          >
                            {row.initialized ? formatInteger(row.remaining) : '—'}
                          </td>
                          <td>
                            <div className="flex justify-end gap-1">
                              <IconButton label={`استخدام ${row.vaccineName}`} onClick={() => openUsage(row)}>
                                <List className="h-3.5 w-3.5" />
                              </IconButton>
                              <IconButton label={`المخزون الشهري لـ ${row.vaccineName}`} onClick={() => setStockRow(row)}>
                                <Boxes className="h-3.5 w-3.5" />
                              </IconButton>
                              <IconButton
                                label={`تعديل مخزون ${row.vaccineName}`}
                                disabled={!row.initialized || data.state !== 'open' || row.archived}
                                onClick={() => setAdjustRow(row)}
                              >
                                <SlidersHorizontal className="h-3.5 w-3.5" />
                              </IconButton>
                            </div>
                          </td>
                          {Array.from({ length: days }, (_, index) => {
                            const count = daily[index] ?? 0;
                            const date = `${month}-${String(index + 1).padStart(2, '0')}`;
                            return (
                              <td key={index} className="px-1 text-center">
                                <button
                                  type="button"
                                  onClick={() => openUsage(row, date)}
                                  title={`${formatDate(date)} · ${count} جرعة فعلية`}
                                  className={cn(
                                    'focus-ring mx-auto flex h-7 w-8 items-center justify-center rounded-md text-xs tabular-nums transition-colors',
                                    count > 0
                                      ? 'bg-emerald-500/15 font-semibold text-emerald-200 hover:bg-emerald-500/25'
                                      : 'text-slate-600 hover:bg-slate-800 hover:text-slate-300',
                                  )}
                                >
                                  {count > 0 ? count : '·'}
                                </button>
                              </td>
                            );
                          })}
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          <Card>
            <CardHeader
              title="تعديلات المخزون المدققة"
action={<Badge tone="slate">{data.adjustments.length}</Badge>}
            />
            {data.adjustments.length === 0 ? (
              <EmptyState icon={<PencilLine className="h-5 w-5" />} title="لا توجد تعديلات مخزون هذا الشهر" />
            ) : (
              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>وقت التسجيل</th>
                      <th>اللقاح</th>
                      <th className="text-end">التغيير</th>
                      <th className="text-end">قبل → بعد</th>
                      <th>السبب</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.adjustments.map((adjustment) => (
                      <tr key={adjustment.id}>
                        <td className="whitespace-nowrap text-xs text-slate-400">{formatTimestamp(adjustment.createdAt)}</td>
                        <td>{adjustment.vaccineName}</td>
                        <td
                          className={cn(
                            'text-end font-medium tabular-nums',
                            adjustment.amount >= 0 ? 'text-emerald-300' : 'text-rose-300',
                          )}
                        >
                          {formatSigned(adjustment.amount)}
                        </td>
                        <td className="text-end tabular-nums text-slate-400">
                          {formatInteger(adjustment.balanceBefore)} → {formatInteger(adjustment.balanceAfter)}
                        </td>
                        <td className="max-w-[360px] truncate text-slate-400" title={adjustment.reason}>
                          {adjustment.reason}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>

          <p className="text-end text-[11px] text-slate-600">تم إنشاء الجدول {formatTimestamp(data.generatedAt)}</p>
        </div>
      ) : null}

      <UsageListDialog
        open={usageOpen}
        month={month}
        row={usageRow}
        date={usageDate}
        onClose={closeUsage}
        onChanged={overview.reload}
      />

      <StockDialog
        open={stockRow !== null}
        month={month}
        row={stockRow}
        onClose={() => setStockRow(null)}
        onSaved={() => {
          setStockRow(null);
          overview.reload();
        }}
      />

      <AdjustDialog
        open={adjustRow !== null}
        month={month}
        row={adjustRow}
        onClose={() => setAdjustRow(null)}
        onSaved={() => {
          setAdjustRow(null);
          overview.reload();
        }}
      />

      <SetupMonthDialog open={setupOpen} month={month} rows={rows} onClose={() => setSetupOpen(false)} onChanged={overview.reload} />

      <ReopenMonthDialog
        open={reopenOpen}
        month={month}
        onClose={() => setReopenOpen(false)}
        onConfirmed={() => {
          setReopenOpen(false);
          overview.reload();
        }}
      />

      <ConfirmDialog
        open={closeOpen}
        title={`إغلاق ${monthLabel(month)}؟`}
        message={
          <>
            يؤدي إغلاق الشهر إلى تجميد تعديلات المخزون والاستخدام الاعتيادية. تُسجل عملية الإغلاق في سجل التدقيق. يمكنك
            إعادة فتحه لاحقاً بسبب مطلوب.
          </>
        }
        confirmLabel="إغلاق الشهر"
        tone="primary"
        busy={closing}
        onConfirm={closeMonth}
        onClose={() => setCloseOpen(false)}
      />
    </>
  );
}
