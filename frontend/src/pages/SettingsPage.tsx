import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Archive,
  BellRing,
  CloudUpload,
  Database,
  DatabaseBackup,
  HardDrive,
  KeyRound,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { api, errorMessage, runAction } from '../lib/api';
import { useAsync } from '../lib/hooks';
import { useToast } from '../components/Toast';
import {
  Alert,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  Field,
  IconButton,
  InlineError,
  Input,
  KeyValue,
  PageHeader,
  Spinner,
} from '../components/ui';

type BusyAction = 'threshold' | 'endpoint' | 'token' | 'clear' | null;

export function SettingsPage() {
  const { notify } = useToast();
  const settings = useAsync(() => api.Settings(), []);

  const [threshold, setThreshold] = useState('10');
  const [endpoint, setEndpoint] = useState('');
  const [token, setToken] = useState('');
  const [busy, setBusy] = useState<BusyAction>(null);
  const [backingUp, setBackingUp] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [restoring, setRestoring] = useState(false);

  useEffect(() => {
    if (settings.data) {
      setThreshold(String(settings.data.lowStockThreshold));
      setEndpoint(settings.data.publisherEndpoint || '');
    }
  }, [settings.data]);

  const saveThreshold = async () => {
    const value = Number(threshold);
    if (!Number.isInteger(value) || value < 0 || value > 1_000_000) {
      notify('أدخل حداً صحيحاً بين 0 و1,000,000. القيمة صفر تعطل التنبيهات.', 'error');
      return;
    }
    setBusy('threshold');
    try {
      const result = await runAction(api.SetLowStockThreshold(value));
      notify(result.message);
      settings.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(null);
    }
  };

  const saveEndpoint = async () => {
    setBusy('endpoint');
    try {
      const result = await runAction(api.SetPublisherEndpoint(endpoint));
      notify(result.message);
      settings.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(null);
    }
  };

  const saveToken = async () => {
    setBusy('token');
    try {
      const result = await runAction(api.SetPublisherToken(token));
      notify(result.message);
      setToken('');
      settings.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(null);
    }
  };

  const clearToken = async () => {
    setBusy('clear');
    try {
      const result = await runAction(api.ClearPublisherToken());
      notify(result.message);
      settings.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBusy(null);
    }
  };

  const createBackup = async () => {
    setBackingUp(true);
    try {
      const result = await api.CreateBackup();
      if (result.ok) {
        notify(result.message);
      } else if (result.code !== 'cancelled') {
        notify(result.message || 'تعذر إنشاء النسخة الاحتياطية.', 'error');
      }
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setBackingUp(false);
    }
  };

  const restoreBackup = async () => {
    setRestoring(true);
    try {
      const result = await api.RestoreBackup();
      if (result.ok) {
        notify(result.message);
        setRestoreOpen(false);
        window.setTimeout(() => window.location.reload(), 900);
      } else if (result.code !== 'cancelled') {
        notify(result.message || 'فشلت الاستعادة.', 'error');
      }
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setRestoring(false);
    }
  };

  const data = settings.data;

  return (
    <>
      <PageHeader
        title="الإعدادات"
        description="إعدادات محطة العمل للتنبيهات والنشر العام الاختياري والنسخ الاحتياطية الموثقة. تُحفظ الأسرار خارج قاعدة بيانات SQLite ولا تُعرض مجدداً بعد الحفظ."
      >
        <IconButton label="تحديث الإعدادات" onClick={settings.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
      </PageHeader>

      <InlineError message={settings.error} />

      {settings.loading && !data ? (
        <Card>
          <Spinner label="جارٍ تحميل الإعدادات المحلية…" />
        </Card>
      ) : data ? (
        <div className="grid gap-4 xl:grid-cols-2">
          <Card>
            <CardHeader
              title="تنبيهات المخزون المنخفض"
              description="يُعلَّم اللقاح عندما يكون مخزونه المتبقي عند هذا الحد أو أقل."
              action={<BellRing className="h-4 w-4 text-slate-500" />}
            />
            <div className="space-y-4 p-4">
              <Field label="الحد (جرعات)" hint="استخدم 0 لتعطيل تنبيهات المخزون المنخفض. أعداد صحيحة فقط.">
                <Input
                  type="number"
                  min={0}
                  max={1_000_000}
                  step={1}
                  value={threshold}
                  onChange={(event) => setThreshold(event.target.value)}
                  className="tabular-nums"
                />
              </Field>
              <Button variant="primary" loading={busy === 'threshold'} onClick={saveThreshold}>
                حفظ الحد
              </Button>
              <p className="text-[11px] text-slate-500">
                تُسجل تغييرات الحد في سجل التدقيق. تتحدث الجداول والتقارير الشهرية الحالية عند فتحها في المرة التالية.
              </p>
            </div>
          </Card>

          <Card>
            <CardHeader
              title="النشر العام"
              description="خدمة HTTPS منفصلة اختيارية. لا يرفع تطبيق سطح المكتب سوى لقطات الإجماليات الشهرية التي تمت مراجعتها."
              action={<CloudUpload className="h-4 w-4 text-slate-500" />}
            />
            <div className="space-y-4 p-4">
              <Field
                label="عنوان النشر"
                hint="عنوان URL الأساسي لخدمة HTTPS العامة. لا يُقبل HTTP العادي إلا لاختبار localhost."
              >
                <Input
                  value={endpoint}
                  onChange={(event) => setEndpoint(event.target.value)}
                  placeholder="https://stats.example.org"
                  spellCheck={false}
                  autoComplete="off"
                />
              </Field>
              <div className="flex flex-wrap items-center gap-2">
                <Button variant="primary" loading={busy === 'endpoint'} onClick={saveEndpoint}>
                  حفظ العنوان
                </Button>
                <Link to="/publish" className="text-xs text-emerald-300 underline underline-offset-2">
                  الانتقال إلى النشر
                </Link>
              </div>

              <div className="border-t border-slate-800/70 pt-4">
                <div className="mb-2 flex items-center justify-between gap-3">
                  <span className="text-xs font-medium text-slate-300">رمز النشر</span>
                  <span className="text-[11px] text-slate-500">
                    {data.publisherTokenConfigured
                      ? `محفوظ في ${storageLabel(data.publisherTokenStorage)}`
                      : 'غير مهيأ'}
                  </span>
                </div>
                <Field label="رمز جديد" hint="24 حرفاً على الأقل. لا يُعرض أبداً بعد الحفظ.">
                  <Input
                    type="password"
                    value={token}
                    onChange={(event) => setToken(event.target.value)}
                    placeholder="ألصق رمز النشر الحامل"
                    autoComplete="new-password"
                    spellCheck={false}
                  />
                </Field>
                <div className="mt-2 flex flex-wrap gap-2">
                  <Button variant="primary" icon={<KeyRound className="h-4 w-4" />} loading={busy === 'token'} onClick={saveToken} disabled={token.trim().length < 24}>
                    حفظ الرمز
                  </Button>
                  <Button
                    variant="outline"
                    icon={<Trash2 className="h-4 w-4" />}
                    loading={busy === 'clear'}
                    onClick={clearToken}
                    disabled={!data.publisherTokenConfigured}
                  >
                    إزالة الرمز
                  </Button>
                </div>
              </div>

              <Alert tone="info">
                يُحفظ الرمز في سلسلة مفاتيح نظام التشغيل عند توفرها، وإلا في ملف خاص بصلاحيات 0600 داخل مجلد بيانات
                التطبيق. لا يُكتب أبداً في قاعدة بيانات SQLite أو سجل التطبيق.
              </Alert>
            </div>
          </Card>

          <Card>
            <CardHeader
              title="مواقع البيانات"
              description="ملفات محلية تملكها محطة العمل هذه. لا تُرفع قاعدة بيانات SQLite أبداً."
              action={<Database className="h-4 w-4 text-slate-500" />}
            />
            <div className="px-4 py-2">
              <KeyValue label="مجلد بيانات التطبيق">
                <span className="font-mono text-[11px]">{data.dataDir}</span>
              </KeyValue>
              <KeyValue label="قاعدة بيانات SQLite">
                <span className="font-mono text-[11px]">{data.databasePath}</span>
              </KeyValue>
              <KeyValue label="مجلد النسخ الاحتياطي والاستعادة">
                <span className="font-mono text-[11px]">{data.backupDir}</span>
              </KeyValue>
              <KeyValue label="وصول قاعدة البيانات العامة">
                <span className="text-emerald-300">لا يوجد</span>
              </KeyValue>
            </div>
          </Card>

          <Card>
            <CardHeader
              title="النسخ الاحتياطي والاحتفاظ"
              description="نسخ متسقة موثقة تُنشأ عبر واجهة النسخ الاحتياطي في SQLite، مع استعادة وتراجع على مراحل."
              action={<DatabaseBackup className="h-4 w-4 text-slate-500" />}
            />
            <div className="space-y-4 p-4">
              <div className="flex flex-wrap gap-2">
                <Button variant="primary" icon={<Archive className="h-4 w-4" />} loading={backingUp} onClick={createBackup}>
                  إنشاء نسخة احتياطية موثقة…
                </Button>
                <Button variant="outline" icon={<HardDrive className="h-4 w-4" />} onClick={() => setRestoreOpen(true)}>
                  استعادة نسخة احتياطية…
                </Button>
              </div>

              <Alert tone="warning" title="سياسة الاحتفاظ الموصى بها">
                احتفظ بنسخ احتياطية يومية على الأقل لآخر 14 يوماً، مع نسخة عند إغلاق كل شهر ونسخة قبل أي استعادة. خزّن
                النسخ على قرص محمي منفصل. لا يحذف هذا التطبيق النسخ الاحتياطية تلقائياً أبداً؛ أزل النسخ القديمة فقط
                وفق سياسة الاحتفاظ المحلية لديك.
              </Alert>

              <p className="text-[11px] leading-5 text-slate-500">
                تتحقق الاستعادة من الملف المختار قبل تغيير أي شيء، وتغلق قاعدة البيانات النشطة، وتحتفظ بنسخة سابقة
                للاستعادة في مجلد الاستعادة، ثم تثبت النسخة الاحتياطية وتعيد فتحها. إذا فشلت إعادة الفتح، تُعاد قاعدة
                البيانات الأصلية تلقائياً.
              </p>
            </div>
          </Card>

          <Card className="xl:col-span-2">
            <CardHeader
              title="حدود الخصوصية"
              description="ما يبقى محلياً وما يمكن أن يغادر محطة العمل هذه."
              action={<ShieldCheck className="h-4 w-4 text-emerald-400" />}
            />
            <div className="grid gap-3 p-4 md:grid-cols-2">
              <div className="rounded-xl border border-rose-500/20 bg-rose-500/5 p-3.5">
                <p className="text-xs font-semibold text-rose-200">لا يغادر محطة العمل هذه أبداً</p>
                <ul className="mt-1.5 list-inside list-disc space-y-1 text-[12px] leading-5 text-rose-100/80">
                  <li>ملف قاعدة بيانات SQLite أو أي نسخة كاملة منه</li>
                  <li>أسماء المرضى ومعرّفاتهم ورموزهم وحالة الأرشفة</li>
                  <li>قيود الاستخدام الفردية والتواريخ والملاحظات وأسباب الإلغاء</li>
                  <li>بيانات التدقيق الوصفية وتصديرات CSV وملفات النسخ الاحتياطي</li>
                </ul>
              </div>
              <div className="rounded-xl border border-emerald-500/20 bg-emerald-500/5 p-3.5">
                <p className="text-xs font-semibold text-emerald-200">قد يُنشر بعد المراجعة</p>
                <ul className="mt-1.5 list-inside list-disc space-y-1 text-[12px] leading-5 text-emerald-100/80">
                  <li>الشهر والرصيد الافتتاحي الإجمالي والمخزون الوارد</li>
                  <li>إجمالي الاستخدام الفعلي والمخزون المتبقي</li>
                  <li>اسم اللقاح والرمز المختصر الاختياري والوحدة</li>
                  <li>إصدار المخطط ومعرّف اللقطة ووقت الإنشاء</li>
                </ul>
              </div>
            </div>
          </Card>
        </div>
      ) : null}

      <ConfirmDialog
        open={restoreOpen}
        title="استعادة نسخة احتياطية موثقة؟"
        message={
          <div className="space-y-2">
            <p>
              ستُحفظ قاعدة البيانات الحالية كنسخة سابقة للاستعادة في مجلد الاستعادة، ثم تُستبدل بالنسخة الاحتياطية
              المختارة. يعيد التطبيق فتح نفسه تلقائياً.
            </p>
            <Alert tone="danger" title="استخدم نسخة احتياطية محلية موثوقة فقط">
              قد تُدخل استعادة بيانات من محطة عمل أخرى أو من ملف غير موثوق سجلات خاطئة. إذا فشل التحقق من النسخة
              الاحتياطية، لا يتغير أي شيء.
            </Alert>
          </div>
        }
        confirmLabel="اختيار النسخة والاستعادة"
        tone="danger"
        busy={restoring}
        onConfirm={restoreBackup}
        onClose={() => setRestoreOpen(false)}
      />
    </>
  );
}

function storageLabel(storage: string): string {
  if (storage === 'os_keyring') return 'سلسلة مفاتيح نظام التشغيل';
  if (storage === 'local_file') return 'ملف محلي خاص';
  return storage || 'تخزين غير معروف';
}
