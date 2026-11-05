import { useState } from 'react';
import {
  Archive,
  ArchiveRestore,
  History,
  Pencil,
  Plus,
  RefreshCw,
  Search,
  ShieldAlert,
  UserRound,
  Users,
} from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { formatDate, formatInteger, formatTimestamp, initials } from '../lib/format';
import { useAsync, useDebouncedValue } from '../lib/hooks';
import { useToast } from '../components/Toast';
import {
  Alert,
  Badge,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  EmptyState,
  Field,
  IconButton,
  InlineError,
  Input,
  Modal,
  PageHeader,
  Spinner,
} from '../components/ui';

interface PatientForm {
  id?: number;
  fullName: string;
  patientCode: string;
}

export function PatientsPage() {
  const { notify } = useToast();
  const [query, setQuery] = useState('');
  const [includeArchived, setIncludeArchived] = useState(false);
  const debouncedQuery = useDebouncedValue(query, 250);
  const patients = useAsync(() => api.SearchPatients(debouncedQuery, includeArchived), [debouncedQuery, includeArchived]);

  const [form, setForm] = useState<PatientForm | null>(null);
  const [formError, setFormError] = useState('');
  const [saving, setSaving] = useState(false);
  const [archiveTarget, setArchiveTarget] = useState<app.PatientDTO | null>(null);
  const [historyTarget, setHistoryTarget] = useState<app.PatientDTO | null>(null);
  const [busy, setBusy] = useState(false);

  const items = patients.data ?? [];

  const openCreate = () => {
    setForm({ fullName: '', patientCode: '' });
    setFormError('');
  };

  const openEdit = (patient: app.PatientDTO) => {
    setForm({ id: patient.id, fullName: patient.fullName, patientCode: patient.patientCode });
    setFormError('');
  };

  const save = async () => {
    if (!form) return;
    if (!form.fullName.trim()) {
      setFormError('اسم المريض مطلوب.');
      return;
    }
    setSaving(true);
    setFormError('');
    try {
      const payload: app.PatientInputDTO = {
        fullName: form.fullName,
        patientCode: form.patientCode,
      };
      const result = form.id
        ? await runAction(api.UpdatePatient(form.id, payload))
        : await runAction(api.CreatePatient(payload));
      notify(result.message);
      setForm(null);
      patients.reload();
    } catch (error) {
      setFormError(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const toggleArchived = async () => {
    if (!archiveTarget) return;
    setBusy(true);
    try {
      const result = await runAction(api.SetPatientArchived(archiveTarget.id, !archiveTarget.archived));
      notify(result.message);
      setArchiveTarget(null);
      patients.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHeader
        title="المرضى"
        description="سجلات هوية محلية للمرضى تُستخدم فقط لإبقاء الاستخدام منسوباً على محطة العمل هذه. الاسم هو الحقل الوحيد المطلوب؛ ويمكن لرمز اختياري تمييز الأسماء المتشابهة."
      >
        <IconButton label="تحديث المرضى" onClick={patients.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="primary" icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
          إضافة مريض
        </Button>
      </PageHeader>

      <Alert tone="info" className="mb-4" title="بيانات محلية خاصة">
        <span className="flex items-start gap-2">
          <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" aria-hidden />
          <span>
            لا تغادر أسماء المرضى ورموزهم وأحداث الاستخدام الفردية والملاحظات محطة العمل هذه أبداً. تتلقى الخدمة العامة
            إجماليات شهرية مجمعة فقط.
          </span>
        </span>
      </Alert>

      <InlineError message={patients.error} />

      <Card>
        <CardHeader
          title="سجل المرضى"
          description="يجري البحث في قاعدة بيانات SQLite المحلية فقط. يحتفظ المرضى المؤرشفون بسجلهم ولا يمكنهم استلام استخدام جديد."
          action={
            <div className="flex flex-wrap items-center gap-3">
              <div className="relative">
                <Search className="pointer-events-none absolute start-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="ابحث بالاسم أو الرمز…"
                  className="w-64 ps-8"
                  aria-label="البحث في المرضى"
                  autoFocus
                />
              </div>
              <label className="flex items-center gap-2 text-xs text-slate-400">
                <input
                  type="checkbox"
                  checked={includeArchived}
                  onChange={(event) => setIncludeArchived(event.target.checked)}
                  className="h-3.5 w-3.5 rounded border-slate-600 bg-slate-950 text-emerald-500 focus:ring-emerald-400"
                />
                تضمين المؤرشف
              </label>
            </div>
          }
        />

        {patients.loading && !patients.data ? (
          <Spinner label="جارٍ البحث في سجلات المرضى المحلية…" />
        ) : items.length === 0 ? (
          <EmptyState
            icon={<Users className="h-5 w-5" />}
            title={debouncedQuery ? 'لا يوجد مرضى مطابقون' : 'لا يوجد مرضى بعد'}
            description={
              debouncedQuery
                ? 'جرّب اسماً أو رمز مريض مختلفاً، أو ضمّن السجلات المؤرشفة.'
                : 'أضف مريضاً قبل تسجيل الاستخدام. الاسم هو الحقل المطلوب الوحيد.'
            }
            action={
              !debouncedQuery ? (
                <Button variant="primary" icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
                  إضافة أول مريض
                </Button>
              ) : undefined
            }
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>المريض</th>
                  <th>رمز المريض</th>
                  <th>الحالة</th>
                  <th>أُضيف</th>
                  <th>آخر تحديث</th>
                  <th className="text-end">الإجراءات</th>
                </tr>
              </thead>
              <tbody>
                {items.map((patient) => (
                  <tr key={patient.id} className={patient.archived ? 'opacity-60' : undefined}>
                    <td>
                      <div className="flex items-center gap-2.5">
                        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-800 text-[11px] font-semibold text-slate-300">
                          {initials(patient.fullName) || <UserRound className="h-3.5 w-3.5" />}
                        </span>
                        <span className="font-medium text-slate-100">{patient.fullName}</span>
                      </div>
                    </td>
                    <td className="text-slate-400">{patient.patientCode || '—'}</td>
                    <td>
                      {patient.archived ? <Badge tone="slate">مؤرشف</Badge> : <Badge tone="emerald">نشط</Badge>}
                    </td>
                    <td className="whitespace-nowrap text-xs text-slate-500">{formatTimestamp(patient.createdAt)}</td>
                    <td className="whitespace-nowrap text-xs text-slate-500">{formatTimestamp(patient.updatedAt)}</td>
                    <td>
                      <div className="flex justify-end gap-1">
                        <IconButton label={`سجل استخدام ${patient.fullName}`} onClick={() => setHistoryTarget(patient)}>
                          <History className="h-3.5 w-3.5" />
                        </IconButton>
                        <IconButton label={`تعديل ${patient.fullName}`} onClick={() => openEdit(patient)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </IconButton>
                        <IconButton
                          label={patient.archived ? `إعادة تنشيط ${patient.fullName}` : `أرشفة ${patient.fullName}`}
                          onClick={() => setArchiveTarget(patient)}
                        >
                          {patient.archived ? <ArchiveRestore className="h-3.5 w-3.5" /> : <Archive className="h-3.5 w-3.5" />}
                        </IconButton>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Modal
        open={form !== null}
        onClose={() => setForm(null)}
        title={form?.id ? 'تعديل المريض' : 'إضافة مريض'}
        description="تُحفظ بيانات المريض في قاعدة بيانات SQLite المحلية فقط ولا تُنشر أبداً."
        size="sm"
        footer={
          <>
            <Button variant="ghost" onClick={() => setForm(null)} disabled={saving}>
              إلغاء
            </Button>
            <Button variant="primary" onClick={save} loading={saving}>
              {form?.id ? 'حفظ المريض' : 'إضافة مريض'}
            </Button>
          </>
        }
      >
        {form ? (
          <div className="space-y-4">
            {formError ? <Alert tone="danger">{formError}</Alert> : null}
            <Field label="الاسم الكامل" required>
              <Input
                value={form.fullName}
                maxLength={160}
                autoFocus
                onChange={(event) => setForm({ ...form, fullName: event.target.value })}
                placeholder="اسم المريض"
              />
            </Field>
            <Field
              label="رمز المريض"
              hint="معرّف محلي اختياري. لا تُعد استخدام أرقام الهوية الوطنية إلا إذا تطلبت سياستك المحلية ذلك."
            >
              <Input
                value={form.patientCode}
                maxLength={80}
                onChange={(event) => setForm({ ...form, patientCode: event.target.value })}
                placeholder="مثال: رقم ملف العيادة"
              />
            </Field>
          </div>
        ) : null}
      </Modal>

      <PatientHistoryDialog patient={historyTarget} onClose={() => setHistoryTarget(null)} />

      <ConfirmDialog
        open={archiveTarget !== null}
        title={archiveTarget?.archived ? 'إعادة تنشيط المريض؟' : 'أرشفة المريض؟'}
        message={
          archiveTarget ? (
            archiveTarget.archived ? (
              <>
                سيظهر <strong>{archiveTarget.fullName}</strong> مجدداً في قائمة المرضى النشطين ويمكنه استلام قيود
                استخدام جديدة.
              </>
            ) : (
              <>
                سيُخفى <strong>{archiveTarget.fullName}</strong> من إدخال الاستخدام الجديد. يبقى الاستخدام التاريخي
                متاحاً وقابلاً للتدقيق.
              </>
            )
          ) : null
        }
        confirmLabel={archiveTarget?.archived ? 'إعادة تنشيط' : 'أرشفة'}
        tone={archiveTarget?.archived ? 'primary' : 'danger'}
        busy={busy}
        onConfirm={toggleArchived}
        onClose={() => setArchiveTarget(null)}
      />
    </>
  );
}

function PatientHistoryDialog({ patient, onClose }: { patient: app.PatientDTO | null; onClose: () => void }) {
  const history = useAsync(
    () => (patient ? api.PatientUsage(patient.id) : Promise.resolve([] as app.UsageRowDTO[])),
    [patient?.id],
  );
  const entries = history.data ?? [];

  return (
    <Modal
      open={patient !== null}
      onClose={onClose}
      title={patient ? `${patient.fullName} · سجل الاستخدام` : 'سجل الاستخدام'}
      description="سجل محلي سري. لا يُرفع أو يُنشر أبداً."
      size="lg"
      footer={
        <Button variant="ghost" onClick={onClose}>
          إغلاق
        </Button>
      }
    >
      <InlineError message={history.error} />
      {patient?.archived ? (
        <Alert tone="warning" className="mb-3">
          هذا المريض مؤرشف. السجل الحالي للقراءة فقط هنا؛ ولا يمكن للمرضى المؤرشفين استلام استخدام جديد.
        </Alert>
      ) : null}
      {history.loading ? (
        <Spinner label="جارٍ تحميل سجل الاستخدام المحلي…" />
      ) : entries.length === 0 ? (
        <EmptyState icon={<History className="h-5 w-5" />} title="لا يوجد استخدام مسجل لهذا المريض" />
      ) : (
        <div className="table-scroll max-h-[440px]">
          <table className="data-table">
            <thead>
              <tr>
                <th>التاريخ</th>
                <th>اللقاح</th>
                <th className="text-end">الجرعات</th>
                <th>ملاحظة</th>
                <th>الحالة</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry) => (
                <tr key={entry.id} className={entry.voided ? 'opacity-60' : undefined}>
                  <td className="whitespace-nowrap">{formatDate(entry.administeredOn)}</td>
                  <td>
                    {entry.vaccineName}
                    {entry.vaccineCode ? <span className="ms-1.5 text-[11px] text-slate-500">{entry.vaccineCode}</span> : null}
                  </td>
                  <td className="text-end font-medium tabular-nums">{formatInteger(entry.quantity)}</td>
                  <td className="max-w-[260px] truncate text-slate-400" title={entry.note}>
                    {entry.note || '—'}
                  </td>
                  <td>
                    {entry.voided ? (
                      <Badge tone="rose" className="max-w-[180px]">
                        <span className="truncate">ملغى: {entry.voidReason}</span>
                      </Badge>
                    ) : (
                      <Badge tone="emerald">مسجل</Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Modal>
  );
}
