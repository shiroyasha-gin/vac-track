import { useState } from 'react';
import { Archive, ArchiveRestore, Pencil, Plus, RefreshCw, Search, Syringe, Trash2 } from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { formatTimestamp } from '../lib/format';
import { useAsync } from '../lib/hooks';
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
  Select,
  Spinner,
  Textarea,
} from '../components/ui';

interface VaccineForm {
  id?: number;
  name: string;
  code: string;
  category: string;
  unit: string;
  description: string;
}

const emptyForm: VaccineForm = {
  name: '',
  code: '',
  category: '',
  unit: 'جرعة',
  description: '',
};

type Scope = 'all' | 'active' | 'archived';

export function VaccinesPage() {
  const { notify } = useToast();
  const vaccines = useAsync(() => api.ListVaccines(true), []);
  const [query, setQuery] = useState('');
  const [scope, setScope] = useState<Scope>('all');
  const [form, setForm] = useState<VaccineForm | null>(null);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState('');
  const [archiveTarget, setArchiveTarget] = useState<app.VaccineDTO | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<app.VaccineDTO | null>(null);
  const [busy, setBusy] = useState(false);

  const items = vaccines.data ?? [];
  const normalizedQuery = query.trim().toLowerCase();
  const filtered = items.filter((item) => {
    if (scope === 'active' && item.archived) return false;
    if (scope === 'archived' && !item.archived) return false;
    if (!normalizedQuery) return true;
    return [item.name, item.code, item.category, item.description].some((value) =>
      (value || '').toLowerCase().includes(normalizedQuery),
    );
  });

  const openCreate = () => {
    setForm({ ...emptyForm });
    setFormError('');
  };

  const openEdit = (vaccine: app.VaccineDTO) => {
    setForm({
      id: vaccine.id,
      name: vaccine.name,
      code: vaccine.code,
      category: vaccine.category,
      unit: vaccine.unit,
      description: vaccine.description,
    });
    setFormError('');
  };

  const save = async () => {
    if (!form) return;
    if (!form.name.trim()) {
      setFormError('اسم اللقاح مطلوب.');
      return;
    }
    setSaving(true);
    setFormError('');
    try {
      const payload: app.VaccineInputDTO = {
        name: form.name,
        code: form.code,
        category: form.category,
        unit: form.unit,
        description: form.description,
      };
      const result = form.id
        ? await runAction(api.UpdateVaccine(form.id, payload))
        : await runAction(api.CreateVaccine(payload));
      notify(result.message);
      setForm(null);
      vaccines.reload();
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
      const result = await runAction(api.SetVaccineArchived(archiveTarget.id, !archiveTarget.archived));
      notify(result.message);
      setArchiveTarget(null);
      vaccines.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!deleteTarget) return;
    setBusy(true);
    try {
      const result = await runAction(api.DeleteVaccine(deleteTarget.id));
      notify(result.message);
      setDeleteTarget(null);
      vaccines.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHeader
        title="اللقاحات"
      >
        <IconButton label="تحديث اللقاحات" onClick={vaccines.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="primary" icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
          إضافة لقاح
        </Button>
      </PageHeader>

      <InlineError message={vaccines.error} />

      <Card>
        <CardHeader
          title="الكتالوج"
          action={
            <div className="flex flex-wrap items-center gap-2">
              <div className="relative w-full sm:w-56">
                <Search className="pointer-events-none absolute start-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  placeholder="ابحث في الكتالوج…"
                  className="w-full ps-8"
                  aria-label="البحث في اللقاحات"
                />
              </div>
              <Select
                value={scope}
                onChange={(event) => setScope(event.target.value as Scope)}
                aria-label="حالة اللقاح"
                className="w-full sm:w-40"
              >
                <option value="all">كل اللقاحات</option>
                <option value="active">النشطة فقط</option>
                <option value="archived">المؤرشفة فقط</option>
              </Select>
            </div>
          }
        />

        {vaccines.loading && !vaccines.data ? (
          <Spinner label="جارٍ تحميل كتالوج اللقاحات…" />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={<Syringe className="h-5 w-5" />}
            title={items.length === 0 ? 'لا توجد لقاحات بعد' : 'لا توجد لقاحات مطابقة لهذا العرض'}
            action={
              items.length === 0 ? (
                <Button variant="primary" icon={<Plus className="h-4 w-4" />} onClick={openCreate}>
                  إضافة أول لقاح
                </Button>
              ) : undefined
            }
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>الاسم</th>
                  <th>الرمز</th>
                  <th>الفئة</th>
                  <th>الوحدة</th>
                  <th>الوصف</th>
                  <th>الحالة</th>
                  <th>آخر تحديث</th>
                  <th className="text-end">الإجراءات</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((vaccine) => (
                  <tr key={vaccine.id} className={vaccine.archived ? 'opacity-60' : undefined}>
                    <td className="font-medium text-slate-100">{vaccine.name}</td>
                    <td className="text-slate-400">{vaccine.code || '—'}</td>
                    <td className="text-slate-400">{vaccine.category || '—'}</td>
                    <td>
                      <Badge tone="slate">{vaccine.unit}</Badge>
                    </td>
                    <td className="max-w-[320px] truncate text-slate-400" title={vaccine.description}>
                      {vaccine.description || '—'}
                    </td>
                    <td>
                      {vaccine.archived ? <Badge tone="slate">مؤرشف</Badge> : <Badge tone="emerald">نشط</Badge>}
                    </td>
                    <td className="whitespace-nowrap text-xs text-slate-500">{formatTimestamp(vaccine.updatedAt)}</td>
                    <td>
                      <div className="flex justify-end gap-1">
                        <IconButton label={`تعديل ${vaccine.name}`} onClick={() => openEdit(vaccine)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </IconButton>
                        <IconButton
                          label={vaccine.archived ? `إعادة تنشيط ${vaccine.name}` : `أرشفة ${vaccine.name}`}
                          onClick={() => setArchiveTarget(vaccine)}
                        >
                          {vaccine.archived ? <ArchiveRestore className="h-3.5 w-3.5" /> : <Archive className="h-3.5 w-3.5" />}
                        </IconButton>
                        {vaccine.archived ? (
                          <IconButton
                            label={`حذف ${vaccine.name}`}
                            className="text-rose-300 hover:bg-rose-500/10"
                            onClick={() => setDeleteTarget(vaccine)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </IconButton>
                        ) : null}
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
        title={form?.id ? 'تعديل اللقاح' : 'إضافة لقاح'}
        size="md"
        footer={
          <>
            <Button variant="ghost" onClick={() => setForm(null)} disabled={saving}>
              إلغاء
            </Button>
            <Button variant="primary" onClick={save} loading={saving}>
              {form?.id ? 'حفظ اللقاح' : 'إضافة لقاح'}
            </Button>
          </>
        }
      >
        {form ? (
          <div className="space-y-4">
            {formError ? <Alert tone="danger">{formError}</Alert> : null}
            <Field label="اسم اللقاح" required>
              <Input
                value={form.name}
                maxLength={120}
                autoFocus
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                placeholder="مثال: الإنفلونزا الموسمية"
              />
            </Field>
            <div className="grid grid-cols-2 gap-3">
              <Field label="رمز مختصر" hint="اختياري. يُحفظ محلياً ويظهر في الجدول الشهري.">
                <Input
                  value={form.code}
                  maxLength={40}
                  onChange={(event) => setForm({ ...form, code: event.target.value })}
                  placeholder="مثال: FLU"
                />
              </Field>
              <Field label="الفئة" hint="تجميع اختياري مثل الروتيني أو السفر.">
                <Input
                  value={form.category}
                  maxLength={60}
                  onChange={(event) => setForm({ ...form, category: event.target.value })}
                  placeholder="مثال: روتيني"
                />
              </Field>
            </div>
            <Field label="الوحدة" required hint="الكميات أعداد صحيحة دائماً. استخدم وحدة مثل جرعة أو عبوة.">
              <Input
                value={form.unit}
                maxLength={20}
                onChange={(event) => setForm({ ...form, unit: event.target.value })}
                placeholder="جرعة"
              />
            </Field>
            <Field label="وصف محلي" hint="لا يُدرج أبداً في الإحصائيات العامة.">
              <Textarea
                value={form.description}
                maxLength={500}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
                placeholder="ملاحظات اختيارية عن هذا اللقاح"
              />
            </Field>
          </div>
        ) : null}
      </Modal>

      <ConfirmDialog
        open={archiveTarget !== null}
        title={archiveTarget?.archived ? 'إعادة تنشيط اللقاح؟' : 'أرشفة اللقاح؟'}
        message={
          archiveTarget ? (
            archiveTarget.archived ? (
              <>
                سيظهر <strong>{archiveTarget.name}</strong> مجدداً كلقاح نشط يحتاج إلى صف مخزون شهري. لا تتغير
                البيانات التاريخية.
              </>
            ) : (
              <>
                سيُخفى <strong>{archiveTarget.name}</strong> من تهيئة الأشهر الجديدة وإدخال الاستخدام. يبقى كل السجل
                الحالي محفوظاً وقابلاً للتدقيق.
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

      <ConfirmDialog
        open={deleteTarget !== null}
        title="حذف اللقاح نهائياً؟"
        message={
          deleteTarget ? (
            <>
              سيُحذف <strong>{deleteTarget.name}</strong> فقط إذا لم يكن له سجل مخزون أو استخدام. يرفض الخادم الحذف
              بخلاف ذلك. لا يمكن التراجع عن هذا الإجراء.
            </>
          ) : null
        }
        confirmLabel="حذف اللقاح"
        tone="danger"
        busy={busy}
        onConfirm={remove}
        onClose={() => setDeleteTarget(null)}
      >
        <Alert tone="warning" title="الأرشفة أفضل">
          الأرشفة قابلة للتراجع وتحفظ السجل. احذف اللقاح فقط إذا أُنشئ بالخطأ.
        </Alert>
      </ConfirmDialog>
    </>
  );
}
