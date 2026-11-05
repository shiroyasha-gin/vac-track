import { useState } from 'react';
import { ChevronLeft, ChevronRight, RefreshCw, ScrollText } from 'lucide-react';
import { api, type app } from '../lib/api';
import { formatInteger, formatTimestamp } from '../lib/format';
import { useAsync } from '../lib/hooks';
import { useMonth } from '../components/MonthContext';
import {
  Badge,
  Button,
  Card,
  CardHeader,
  EmptyState,
  Field,
  IconButton,
  InlineError,
  Input,
  PageHeader,
  Select,
  Spinner,
} from '../components/ui';

const eventTypes = [
  'vaccine.created',
  'vaccine.updated',
  'vaccine.archived',
  'vaccine.reactivated',
  'vaccine.deleted',
  'patient.created',
  'patient.updated',
  'patient.archived',
  'patient.reactivated',
  'inventory.initialized',
  'inventory.stock_updated',
  'inventory.adjusted',
  'inventory.carryover_overridden',
  'month.closed',
  'month.reopened',
  'usage.created',
  'usage.updated',
  'usage.voided',
  'backup.created',
  'backup.restored',
  'publication.published',
  'publication.unpublished',
  'publication.failed',
  'settings.updated',
  'export.completed',
];

const eventLabels: Record<string, string> = {
  'vaccine.created': 'إضافة لقاح',
  'vaccine.updated': 'تعديل لقاح',
  'vaccine.archived': 'أرشفة لقاح',
  'vaccine.reactivated': 'إعادة تنشيط لقاح',
  'vaccine.deleted': 'حذف لقاح',
  'patient.created': 'إضافة مريض',
  'patient.updated': 'تعديل مريض',
  'patient.archived': 'أرشفة مريض',
  'patient.reactivated': 'إعادة تنشيط مريض',
  'inventory.initialized': 'تهيئة مخزون',
  'inventory.stock_updated': 'تحديث مخزون',
  'inventory.adjusted': 'تعديل مخزون',
  'inventory.carryover_overridden': 'تجاوز الترحيل',
  'month.closed': 'إغلاق شهر',
  'month.reopened': 'إعادة فتح شهر',
  'usage.created': 'تسجيل استخدام',
  'usage.updated': 'تعديل استخدام',
  'usage.voided': 'إلغاء استخدام',
  'backup.created': 'إنشاء نسخة احتياطية',
  'backup.restored': 'استعادة نسخة احتياطية',
  'publication.published': 'نشر لقطة',
  'publication.unpublished': 'إلغاء نشر',
  'publication.failed': 'فشل النشر',
  'settings.updated': 'تحديث إعدادات',
  'export.completed': 'إكمال تصدير',
};

const entityLabels: Record<string, string> = {
  vaccine: 'لقاح',
  patient: 'مريض',
  inventory: 'مخزون',
  month: 'شهر',
  usage: 'استخدام',
  backup: 'نسخة احتياطية',
  publication: 'نشر',
  settings: 'إعدادات',
  export: 'تصدير',
  app: 'تطبيق',
};

const PAGE_SIZE = 50;

function eventLabel(type: string): string {
  return eventLabels[type] ?? type;
}

function entityLabel(type: string): string {
  return entityLabels[type] ?? type;
}

export function AuditPage() {
  const { month } = useMonth();
  const [eventType, setEventType] = useState('');
  const [filterMonth, setFilterMonth] = useState('');
  const [offset, setOffset] = useState(0);

  const events = useAsync(
    () =>
      api.AuditEvents({
        eventType,
        month: filterMonth,
        limit: PAGE_SIZE,
        offset,
      }),
    [eventType, filterMonth, offset],
  );

  const page = events.data;
  const items = page?.events ?? [];
  const total = page?.total ?? 0;

  const changeEventType = (value: string) => {
    setEventType(value);
    setOffset(0);
  };

  const changeMonth = (value: string) => {
    setFilterMonth(value);
    setOffset(0);
  };

  return (
    <>
      <PageHeader
        title="سجل التدقيق"
        description="سجل محلي مختصر لتغييرات المخزون والاستخدام والنشر والنسخ الاحتياطي والإعدادات. لا تُنسخ أسماء المرضى أو ملاحظات الاستخدام إلى بيانات التدقيق الوصفية."
      >
        <IconButton label="تحديث أحداث التدقيق" onClick={events.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
      </PageHeader>

      <InlineError message={events.error} />

      <Card>
        <CardHeader
          title="الأحداث المسجلة"
          description="الأحدث أولاً. السجل للإضافة فقط في التشغيل الاعتيادي؛ ولا يُنشر أبداً."
          action={
            <div className="flex flex-wrap items-end gap-2">
              <Field label="نوع الحدث">
                <Select value={eventType} onChange={(event) => changeEventType(event.target.value)} aria-label="نوع الحدث">
                  <option value="">كل أنواع الأحداث</option>
                  {eventTypes.map((type) => (
                    <option key={type} value={type}>
                      {eventLabel(type)}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field label="الشهر">
                <Input
                  type="month"
                  value={filterMonth}
                  onChange={(event) => changeMonth(event.target.value)}
                  aria-label="التصفية حسب الشهر"
                  className="w-[150px]"
                />
              </Field>
              <Button variant="ghost" size="sm" onClick={() => changeMonth('')} disabled={!filterMonth}>
                كل الأشهر
              </Button>
              <Button variant="ghost" size="sm" onClick={() => changeMonth(month)}>
                الشهر المحدد
              </Button>
            </div>
          }
        />

        {events.loading && !page ? (
          <Spinner label="جارٍ قراءة سجل التدقيق المحلي…" />
        ) : items.length === 0 ? (
          <EmptyState
            icon={<ScrollText className="h-5 w-5" />}
            title="لا توجد أحداث تدقيق مطابقة لهذا المرشح"
            description="غيّر نوع الحدث أو مرشح الشهر لمراجعة إجراءات أخرى."
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>الوقت</th>
                  <th>الحدث</th>
                  <th>الكيان</th>
                  <th>الشهر</th>
                  <th>السبب</th>
                  <th>التفاصيل</th>
                </tr>
              </thead>
              <tbody>
                {items.map((event: app.AuditEventDTO) => (
                  <tr key={event.id}>
                    <td className="whitespace-nowrap text-xs text-slate-400">{formatTimestamp(event.createdAt)}</td>
                    <td>
                      <Badge tone={eventTone(event.eventType)} className="font-medium">
                        {eventLabel(event.eventType)}
                      </Badge>
                    </td>
                    <td className="text-xs text-slate-400">
                      {entityLabel(event.entityType)}
                      {event.entityId ? <span className="ms-1 text-slate-600">#{event.entityId}</span> : null}
                    </td>
                    <td className="text-xs text-slate-400">{event.month || '—'}</td>
                    <td className="max-w-[260px] truncate text-slate-400" title={event.reason}>
                      {event.reason || '—'}
                    </td>
                    <td className="max-w-[300px]">
                      {event.metadata ? (
                        <details>
                          <summary className="cursor-pointer text-[11px] text-sky-300 hover:text-sky-200">عرض</summary>
                          <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md bg-slate-950/70 p-2 text-[10px] leading-4 text-slate-400">
                            {event.metadata}
                          </pre>
                        </details>
                      ) : (
                        <span className="text-xs text-slate-600">—</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-slate-800/70 px-4 py-3 text-xs text-slate-500">
          <span>
            {total === 0
              ? 'لا توجد أحداث'
              : `عرض ${formatInteger(offset + 1)}–${formatInteger(Math.min(offset + items.length, total))} من ${formatInteger(total)}`}
          </span>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              icon={<ChevronRight className="h-3.5 w-3.5" />}
              disabled={offset === 0 || events.loading}
              onClick={() => setOffset((value) => Math.max(0, value - PAGE_SIZE))}
            >
              السابق
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + items.length >= total || events.loading}
              onClick={() => setOffset((value) => value + PAGE_SIZE)}
            >
              التالي
              <ChevronLeft className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      </Card>
    </>
  );
}

function eventTone(type: string): 'slate' | 'emerald' | 'amber' | 'rose' | 'sky' | 'violet' {
  if (type.startsWith('usage.')) return 'emerald';
  if (type.startsWith('inventory.')) return 'sky';
  if (type.startsWith('month.')) return 'violet';
  if (type.startsWith('publication.')) return 'amber';
  if (type === 'backup.restored') return 'rose';
  if (type.startsWith('backup.')) return 'slate';
  if (type.startsWith('patient.')) return 'sky';
  if (type.startsWith('vaccine.')) return 'slate';
  return 'slate';
}
