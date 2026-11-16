import { useEffect, useState } from 'react';
import {
  Archive,
  BellRing,
  Database,
  DatabaseBackup,
  HardDrive,
  KeyRound,
  RefreshCw,
  ShieldCheck,
} from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { useAsync } from '../lib/hooks';
import { useToast } from '../components/Toast';
import { AccountManagement } from '../components/AccountManagement';
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

type BusyAction = 'threshold' | null;

export function SettingsPage({
  role,
  currentUserId,
  onAuthUpdated,
}: {
  role: string;
  currentUserId: number;
  onAuthUpdated: (state: app.AuthStateDTO) => void;
}) {
  const { notify } = useToast();
  const isAdmin = role === 'admin';
  const settings = useAsync(() => (isAdmin ? api.Settings() : Promise.resolve(null)), [isAdmin]);

  const [threshold, setThreshold] = useState('10');
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [changingPassword, setChangingPassword] = useState(false);
  const [passwordError, setPasswordError] = useState('');
  const [busy, setBusy] = useState<BusyAction>(null);
  const [backingUp, setBackingUp] = useState(false);
  const [restoreOpen, setRestoreOpen] = useState(false);
  const [restoring, setRestoring] = useState(false);

  const changePassword = async () => {
    setPasswordError('');
    if (!currentPassword) {
      setPasswordError('أدخل كلمة المرور الحالية.');
      return;
    }
    if ([...newPassword].length < 8) {
      setPasswordError('يجب أن تتكون كلمة المرور الجديدة من 8 أحرف على الأقل.');
      return;
    }
    if (newPassword !== confirmPassword) {
      setPasswordError('كلمتا المرور الجديدتان غير متطابقتين.');
      return;
    }
    setChangingPassword(true);
    try {
      const state = await api.ChangePassword(currentPassword, newPassword);
      onAuthUpdated(state);
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      notify('تم تغيير كلمة المرور. استخدمها في المرة القادمة.');
    } catch (error) {
      setPasswordError(errorMessage(error));
    } finally {
      setChangingPassword(false);
    }
  };

  useEffect(() => {
    if (settings.data) {
      setThreshold(String(settings.data.lowStockThreshold));
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

  const securityCard = (
    <Card>
      <CardHeader
        title="أمان الحساب"
        description="غيّر كلمة المرور المحلية لهذا المشغّل."
        action={<ShieldCheck className="h-4 w-4 text-slate-500" />}
      />
      <div className="space-y-4 p-4">
        {passwordError ? <InlineError message={passwordError} /> : null}
        <Field label="كلمة المرور الحالية" htmlFor="current-password">
          <Input
            id="current-password"
            type="password"
            autoComplete="current-password"
            value={currentPassword}
            onChange={(event) => setCurrentPassword(event.target.value)}
            disabled={changingPassword}
          />
        </Field>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="كلمة المرور الجديدة" htmlFor="new-password" hint="8 أحرف على الأقل.">
            <Input
              id="new-password"
              type="password"
              autoComplete="new-password"
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
              disabled={changingPassword}
            />
          </Field>
          <Field label="تأكيد كلمة المرور" htmlFor="confirm-password">
            <Input
              id="confirm-password"
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              disabled={changingPassword}
            />
          </Field>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <Button
            variant="primary"
            icon={<KeyRound className="h-4 w-4" />}
            loading={changingPassword}
            onClick={changePassword}
          >
            تغيير كلمة المرور
          </Button>
          <p className="text-[11px] leading-5 text-slate-500">
            تُحفظ كلمة المرور كتجزئة bcrypt محلية، ولا تُرسل عبر الشبكة.
          </p>
        </div>
      </div>
    </Card>
  );

  return (
    <>
      <PageHeader title="الإعدادات" description={isAdmin ? 'إعدادات محطة العمل والمستخدمين والنسخ الاحتياطية.' : 'أمان حسابك المحلي.'}>
        {isAdmin ? (
          <IconButton label="تحديث الإعدادات" onClick={settings.reload}>
            <RefreshCw className="h-4 w-4" />
          </IconButton>
        ) : null}
      </PageHeader>

      <InlineError message={settings.error} />

      {!isAdmin ? (
        <div className="grid gap-4 xl:grid-cols-2">{securityCard}</div>
      ) : settings.loading && !data ? (
        <Card>
          <Spinner label="جارٍ تحميل الإعدادات المحلية…" />
        </Card>
      ) : data ? (
        <div className="space-y-4">
          <AccountManagement currentUserId={currentUserId} onAuthUpdated={onAuthUpdated} />

          <div className="grid gap-4 xl:grid-cols-2">
            {securityCard}

            <Card>
              <CardHeader
                title="تنبيهات المخزون المنخفض"
                description="يُعلَّم اللقاح عندما يكون مخزونه المتبقي عند هذا الحد أو أقل."
                action={<BellRing className="h-4 w-4 text-slate-500" />}
              />
              <div className="space-y-4 p-4">
                <Field label="الحد (جرعات)" hint="استخدم 0 لتعطيل التنبيهات.">
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
              </div>
            </Card>

            <Card>
              <CardHeader
                title="النسخ الاحتياطي والاحتفاظ"
                description="نسخ متسقة موثقة مع استعادة وتراجع على مراحل."
                action={<DatabaseBackup className="h-4 w-4 text-slate-500" />}
              />
              <div className="space-y-4 p-4">
                <div className="flex flex-wrap gap-2">
                  <Button variant="primary" icon={<Archive className="h-4 w-4" />} loading={backingUp} onClick={createBackup}>
                    إنشاء نسخة احتياطية…
                  </Button>
                  <Button variant="outline" icon={<HardDrive className="h-4 w-4" />} onClick={() => setRestoreOpen(true)}>
                    استعادة نسخة احتياطية…
                  </Button>
                </div>
                <p className="text-[11px] leading-5 text-slate-500">
                  تتحقق الاستعادة من الملف المختار قبل تغيير أي شيء، وتغلق قاعدة البيانات النشطة، وتحتفظ بنسخة سابقة
                  للاستعادة. إذا فشلت إعادة الفتح، تُعاد قاعدة البيانات الأصلية تلقائياً.
                </p>
              </div>
            </Card>

            <Card className="xl:col-span-2">
              <CardHeader
                title="مواقع البيانات"
                description="ملفات محلية تملكها محطة العمل هذه."
                action={<Database className="h-4 w-4 text-slate-500" />}
              />
              <div className="px-4 py-2">
                <KeyValue label="مجلد بيانات التطبيق">
                  <span className="font-mono text-[11px]">{data.dataDir}</span>
                </KeyValue>
                <KeyValue label="قاعدة بيانات SQLite">
                  <span className="font-mono text-[11px]">{data.databasePath}</span>
                </KeyValue>
                <KeyValue label="مجلد النسخ الاحتياطي">
                  <span className="font-mono text-[11px]">{data.backupDir}</span>
                </KeyValue>
              </div>
            </Card>
          </div>
        </div>
      ) : null}

      <ConfirmDialog
        open={restoreOpen}
        title="استعادة نسخة احتياطية موثقة؟"
        message={
          <div className="space-y-2">
            <p>
              ستُحفظ قاعدة البيانات الحالية كنسخة سابقة للاستعادة، ثم تُستبدل بالنسخة الاحتياطية المختارة. يعيد
              التطبيق فتح نفسه تلقائياً.
            </p>
            <Alert tone="danger" title="استخدم نسخة احتياطية محلية موثوقة فقط">
              قد تُدخل استعادة بيانات من محطة عمل أخرى سجلات خاطئة. إذا فشل التحقق، لا يتغير أي شيء.
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