import { useEffect, useState } from 'react';
import { CalendarPlus, List, Pencil, Plus, ShieldAlert, Trash2 } from 'lucide-react';
import {
  api,
  errorMessage,
  runAction,
  type app,
} from '../lib/api';
import { formatDate, formatInteger, monthDays, monthLabel, todayISO } from '../lib/format';
import { useAsync } from '../lib/hooks';
import { useToast } from './Toast';
import { PatientPicker, type PatientChoice } from './PatientPicker';
import {
  Alert,
  Badge,
  Button,
  ConfirmDialog,
  EmptyState,
  Field,
  InlineError,
  Input,
  Modal,
  Spinner,
  StatusBadge,
  Textarea,
} from './ui';

interface UsageListDialogProps {
  open: boolean;
  month: string;
  row: app.MonthRowDTO | null;
  date?: string;
  onClose: () => void;
  onChanged: () => void;
}

export function UsageListDialog({ open, month, row, date, onClose, onChanged }: UsageListDialogProps) {
  const { notify } = useToast();
  const [editing, setEditing] = useState<app.UsageRowDTO | null>(null);
  const [creating, setCreating] = useState(false);
  const [voidTarget, setVoidTarget] = useState<app.UsageRowDTO | null>(null);
  const [voidReason, setVoidReason] = useState('');
  const [busy, setBusy] = useState(false);

  const entries = useAsync(
    () => {
      if (!open || !row) {
        return Promise.resolve([] as app.UsageRowDTO[]);
      }
      if (date) {
        return api.UsageForVaccineDay(row.vaccineId, month, date);
      }
      return api.UsageForVaccineMonth(row.vaccineId, month);
    },
    [open, row?.vaccineId, month, date],
  );

  useEffect(() => {
    if (!open) {
      setEditing(null);
      setCreating(false);
      setVoidTarget(null);
      setVoidReason('');
    }
  }, [open]);

  const canEdit = row?.initialized && row.status === 'open' && !row.archived;

  const refresh = () => {
    entries.reload();
    onChanged();
  };

  const voidEntry = async () => {
    if (!voidTarget) {
      return;
    }
    setBusy(true);
    try {
      await runAction(api.VoidUsage({ id: voidTarget.id, reason: voidReason }));
      notify('تم إلغاء قيد الاستخدام.');
      setVoidTarget(null);
      setVoidReason('');
      refresh();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(false);
    }
  };

  const title = row ? `${row.vaccineName} · ${date ? formatDate(date) : monthLabel(month)}` : 'الاستخدام';

  return (
    <>
      <Modal
        open={open}
        onClose={onClose}
        title={title}
        description={
          row ? (
            <span className="flex flex-wrap items-center gap-2">
              <span>تفاصيل استخدام محلية خاصة. لا تُرفع أو تُنشر أبداً.</span>
              {row.initialized ? <StatusBadge state={row.status} /> : null}
            </span>
          ) : null
        }
        size="lg"
        footer={
          <>
            <Button variant="ghost" onClick={onClose}>
              إغلاق
            </Button>
            {canEdit ? (
              <Button
                variant="primary"
                icon={<CalendarPlus className="h-4 w-4" />}
                onClick={() => setCreating(true)}
              >
                إضافة استخدام
              </Button>
            ) : null}
          </>
        }
      >
        {!row || row.initialized ? null : (
          <Alert tone="warning" title="المخزون الشهري غير مهيأ">
            لا يمكن ربط أي استخدام بهذا اللقاح حتى يوجد صف مخزون شهري صريح.
          </Alert>
        )}
        {row?.archived ? (
          <Alert tone="warning" className="mb-3">
            هذا اللقاح مؤرشف. يُعرض السجل الحالي، لكن التعديلات الاعتيادية محظورة.
          </Alert>
        ) : null}
        {row?.status === 'closed' ? (
          <Alert tone="warning" className="mb-3">
            هذا الشهر مغلق. أعد فتحه بسبب قبل تعديل الاستخدام أو إضافته.
          </Alert>
        ) : null}
        <InlineError message={entries.error} />
        {entries.loading ? (
          <Spinner label="جارٍ تحميل الاستخدام المحلي…" />
        ) : entries.data && entries.data.length > 0 ? (
          <div className="table-scroll max-h-[420px]">
            <table className="data-table">
              <thead>
                <tr>
                  <th>التاريخ</th>
                  <th>المريض</th>
                  <th className="text-end">الجرعات</th>
                  <th>ملاحظة</th>
                  <th>الحالة</th>
                  <th className="text-end">الإجراءات</th>
                </tr>
              </thead>
              <tbody>
                {entries.data.map((entry) => (
                  <tr key={entry.id} className={entry.voided ? 'opacity-60' : undefined}>
                    <td className="whitespace-nowrap">{formatDate(entry.administeredOn)}</td>
                    <td>
                      <span className={entry.voided ? 'line-through' : undefined}>{entry.patientName}</span>
                      {entry.patientCode ? <span className="ms-1.5 text-[11px] text-slate-500">{entry.patientCode}</span> : null}
                    </td>
                    <td className="text-end font-medium tabular-nums">{formatInteger(entry.quantity)}</td>
                    <td className="max-w-[240px] truncate text-slate-400" title={entry.note}>
                      {entry.note || '—'}
                    </td>
                    <td>
                      {entry.voided ? (
                        <Badge tone="rose" className="max-w-[160px]">
                          <span className="truncate">ملغى: {entry.voidReason}</span>
                        </Badge>
                      ) : (
                        <Badge tone="emerald">مسجل</Badge>
                      )}
                    </td>
                    <td>
                      {!entry.voided && canEdit ? (
                        <div className="flex justify-end gap-1">
                          <Button
                            size="sm"
                            variant="ghost"
                            icon={<Pencil className="h-3.5 w-3.5" />}
                            onClick={() => setEditing(entry)}
                          >
                            تعديل
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            className="text-rose-300 hover:bg-rose-500/10"
                            icon={<Trash2 className="h-3.5 w-3.5" />}
                            onClick={() => setVoidTarget(entry)}
                          >
                            إلغاء
                          </Button>
                        </div>
                      ) : (
                        <span className="block text-end text-xs text-slate-600">—</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState
            icon={<List className="h-5 w-5" />}
            title="لا يوجد استخدام في هذا العرض"
            description={canEdit ? 'سجل إعطاء جرعة لتخفيض المخزون المتاح.' : 'لا يمكن إضافة استخدام اعتيادي ما دام هذا الشهر مغلقاً.'}
            action={
              canEdit ? (
                <Button variant="secondary" size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setCreating(true)}>
                  إضافة استخدام
                </Button>
              ) : undefined
            }
          />
        )}
      </Modal>

      <UsageEditorDialog
        open={creating || editing !== null}
        month={month}
        row={row}
        defaultDate={date}
        entry={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={() => {
          setCreating(false);
          setEditing(null);
          refresh();
        }}
      />

      <ConfirmDialog
        open={voidTarget !== null}
        title="إلغاء قيد الاستخدام؟"
        message={
          voidTarget ? (
            <>
              إلغاء <strong>{formatInteger(voidTarget.quantity)}</strong> جرعة للمريض <strong>{voidTarget.patientName}</strong> بتاريخ{' '}
              {formatDate(voidTarget.administeredOn)}. يبقى القيد محفوظاً للتدقيق ويُستبعد من إجماليات المخزون.
            </>
          ) : null
        }
        confirmLabel="إلغاء القيد"
        tone="danger"
        busy={busy}
        onConfirm={voidEntry}
        onClose={() => setVoidTarget(null)}
      >
        <Field label="السبب" required>
          <Textarea value={voidReason} onChange={(event) => setVoidReason(event.target.value)} placeholder="لماذا يُلغى هذا القيد؟" />
        </Field>
      </ConfirmDialog>
    </>
  );
}

interface UsageEditorDialogProps {
  open: boolean;
  month: string;
  row: app.MonthRowDTO | null;
  defaultDate?: string;
  entry?: app.UsageRowDTO | null;
  onClose: () => void;
  onSaved: () => void;
}

export function UsageEditorDialog({ open, month, row, defaultDate, entry, onClose, onSaved }: UsageEditorDialogProps) {
  const { notify } = useToast();
  const [patient, setPatient] = useState<PatientChoice | null>(null);
  const [administeredOn, setAdministeredOn] = useState('');
  const [quantity, setQuantity] = useState('1');
  const [note, setNote] = useState('');
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    setError('');
    setBusy(false);
    setPatient(
      entry
        ? {
            id: entry.patientId,
            fullName: entry.patientName,
            patientCode: entry.patientCode,
            archived: entry.patientArchived,
          }
        : null,
    );
    setAdministeredOn(entry?.administeredOn || defaultDate || (todayISO().startsWith(month) ? todayISO() : `${month}-01`));
    setQuantity(entry ? String(entry.quantity) : '1');
    setNote(entry?.note || '');
    setReason('');
  }, [open, entry, defaultDate, month]);

  const isEditing = Boolean(entry);

  const submit = async () => {
    setError('');
    if (!patient) {
      setError('اختر مريضاً أولاً.');
      return;
    }
    setBusy(true);
    try {
      const parsedQuantity = Number(quantity);
      if (!Number.isInteger(parsedQuantity) || parsedQuantity <= 0) {
        setError('أدخل عدداً صحيحاً من الجرعات أكبر من صفر.');
        return;
      }
      if (isEditing && entry) {
        await runAction(
          api.UpdateUsage({
            id: entry.id,
            patientId: patient.id,
            administeredOn,
            quantity: parsedQuantity,
            note,
            reason,
          }),
        );
        notify('تم تصحيح قيد الاستخدام.');
      } else {
        if (!row) {
          setError('اختر لقاحاً مهيأ أولاً.');
          return;
        }
        await runAction(
          api.CreateUsage({
            patientId: patient.id,
            vaccineId: row.vaccineId,
            administeredOn,
            quantity: parsedQuantity,
            note,
          }),
        );
        notify('تم تسجيل الاستخدام.');
      }
      onSaved();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  const patientMissing = isEditing && entry?.patientArchived;

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEditing ? 'تصحيح قيد استخدام' : 'تسجيل استخدام'}
      description={
        row
          ? `${row.vaccineName} · ${monthLabel(month)}. هذه بيانات مرضى محلية خاصة ولا تُنشر أبداً.`
          : 'اختر لقاحاً وسجّل إعطاءً محلياً.'
      }
      size="md"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            إلغاء
          </Button>
          <Button variant="primary" onClick={submit} loading={busy}>
            {isEditing ? 'حفظ التصحيح' : 'تسجيل الاستخدام'}
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error ? <Alert tone="danger">{error}</Alert> : null}
        {patientMissing ? (
          <Alert tone="warning" title="المريض مؤرشف">
            ينتمي هذا القيد إلى مريض مؤرشف ولا يمكن إعادة إسناده إليه. ألغِ القيد بدلاً من ذلك، أو اختر مريضاً نشطاً
            آخر إذا أُدخل القيد لشخص خاطئ.
          </Alert>
        ) : null}

        <Field label="المريض" required hint="ابحث بالاسم أو الرمز. تُعرض المرضى النشطون فقط.">
          <PatientPicker value={patient} onChange={setPatient} autoFocus onError={setError} />
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="تاريخ الإعطاء" required hint="يجب أن يقع داخل شهر المخزون المحدد.">
            <Input
              type="date"
              value={administeredOn}
              min={`${month}-01`}
              max={`${month}-${String(monthDays(month)).padStart(2, '0')}`}
              onChange={(event) => setAdministeredOn(event.target.value)}
            />
          </Field>
          <Field label="الجرعات" required hint="جرعات كاملة فقط.">
            <Input
              type="number"
              min={1}
              step={1}
              value={quantity}
              onChange={(event) => setQuantity(event.target.value)}
              className="tabular-nums"
            />
          </Field>
        </div>

        <Field label="ملاحظة" hint="ملاحظة محلية اختيارية. لا تُدرج أبداً في الإحصائيات العامة أو بيانات التدقيق الوصفية.">
          <Textarea value={note} onChange={(event) => setNote(event.target.value)} maxLength={300} />
        </Field>

        {isEditing ? (
          <Field label="سبب التصحيح" hint="يُنصح به لمراجعة التدقيق لاحقاً. يُسجل التعديل في سجل التدقيق.">
            <Input value={reason} onChange={(event) => setReason(event.target.value)} maxLength={500} />
          </Field>
        ) : null}

        <div className="flex items-start gap-2 rounded-lg border border-slate-700/60 bg-slate-900 px-3 py-2 text-[11px] text-slate-500">
          <ShieldAlert className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" aria-hidden />
          يُرفض الاستخدام إذا لم يُهيأ المخزون الشهري صراحةً، أو كان الشهر مغلقاً، أو كانت الكمية المسجلة ستجعل المخزون
          المتبقي سالباً.
        </div>
      </div>
    </Modal>
  );
}
