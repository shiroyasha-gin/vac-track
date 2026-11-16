import { useState } from 'react';
import {
  AlertTriangle,
  DatabaseBackup,
  FilePlus2,
  Loader2,
  RotateCcw,
  ShieldAlert,
} from 'lucide-react';
import { api, errorMessage, type app } from '../lib/api';
import { Alert, Button, Card, ConfirmDialog, KeyValue } from '../components/ui';

export function FatalPage({ bootstrap, onReload }: { bootstrap: app.BootstrapInfo; onReload: () => void }) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');
  const [confirmInit, setConfirmInit] = useState(false);

  const restore = async () => {
    setBusy(true);
    setMessage('');
    try {
      const result = await api.RestoreBackup();
      if (!result.ok) {
        setMessage(result.message || 'تم إلغاء الاستعادة أو فشلت.');
        return;
      }
      setMessage(result.message);
      window.setTimeout(onReload, 900);
    } catch (error) {
      setMessage(errorMessage(error));
    } finally {
      setBusy(false);
    }
  };

  const initialize = async () => {
    setConfirmInit(false);
    setBusy(true);
    setMessage('');
    try {
      const result = await api.InitializeDatabase();
      if (!result.ok) {
        setMessage(result.message || 'تعذر إنشاء قاعدة بيانات جديدة.');
        return;
      }
      setMessage(result.message);
      window.setTimeout(onReload, 1200);
    } catch (error) {
      setMessage(errorMessage(error));
    } finally {
      setBusy(false);
    }
  };

  const retry = () => {
    setBusy(true);
    onReload();
  };

  return (
    <div className="flex min-h-full items-center justify-center bg-slate-900 px-6 py-10">
      <div className="w-full max-w-2xl">
        <div className="mb-6 flex items-center gap-3">
          <div className="rounded-xl bg-rose-500/10 p-2.5 text-rose-300 ring-1 ring-inset ring-rose-500/25">
            <ShieldAlert className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-slate-100">قاعدة البيانات المحلية غير متاحة</h1>
            <p className="mt-0.5 text-[13px] text-slate-500">
              إذا لم تكن قاعدة البيانات موجودة يُنشئها التطبيق تلقائياً عند البدء. هذا الشاشة تظهر عندما يفشل فتح
              الملف الموجود. أعد المحاولة، أو استعد نسخة احتياطية، أو ابدأ قاعدة جديدة مع الاحتفاظ بالملفات القديمة.
            </p>
          </div>
        </div>

        <Card className="p-5">
          <Alert tone="danger" title="خطأ في بدء التشغيل">
            <p className="whitespace-pre-wrap break-words">{bootstrap.fatalError || 'خطأ بدء تشغيل غير معروف.'}</p>
          </Alert>

          <div className="mt-4 rounded-xl border border-slate-700/70 bg-slate-900 px-4">
            <KeyValue label="مجلد بيانات التطبيق">{bootstrap.dataDir || 'غير متاح'}</KeyValue>
            <KeyValue label="ملف قاعدة البيانات">{bootstrap.databasePath || 'غير متاح'}</KeyValue>
            <KeyValue label="إصدار التطبيق">v{bootstrap.version}</KeyValue>
          </div>

          <div className="mt-4 rounded-xl border border-amber-500/25 bg-amber-500/[0.07] p-3 text-xs leading-5 text-slate-200">
            <p className="flex items-center gap-1.5 font-medium text-amber-300">
              <AlertTriangle className="h-3.5 w-3.5" /> سلامة الاستعادة
            </p>
            <p className="mt-1">
              تتحقق عملية الاستعادة من النسخة الاحتياطية قبل تغيير أي شيء. تُحفظ قاعدة البيانات الحالية في مجلد النسخ
              الاحتياطية كنسخة سابقة للاستعادة. لا تستعد إلا نسخة احتياطية من محطة العمل هذه.
            </p>
          </div>

          {message ? (
            <p className="mt-4 rounded-lg bg-slate-900 px-3 py-2 text-xs text-slate-200" role="status">
              {message}
            </p>
          ) : null}

          <div className="mt-5 flex flex-wrap items-center gap-2">
            <Button
              variant="primary"
              onClick={restore}
              disabled={busy}
              icon={busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <DatabaseBackup className="h-4 w-4" />}
            >
              استعادة نسخة احتياطية…
            </Button>
            <Button variant="outline" onClick={retry} disabled={busy} icon={<RotateCcw className="h-4 w-4" />}>
              إعادة محاولة البدء
            </Button>
          </div>

          <div className="mt-5 border-t border-slate-700/60 pt-4">
            <p className="text-xs font-medium text-slate-200">بدء قاعدة بيانات جديدة</p>
            <p className="mt-1 text-[11px] leading-5 text-slate-500">
              تُستخدم عندما يكون الملف تالفاً ولا توجد نسخة احتياطية صالحة. تُنقل ملفات قاعدة البيانات الحالية إلى
              مجلد النسخ الاحتياطية أولاً، ثم تُنشأ قاعدة فارغة. إذا كان الملف مقفلاً ولا يمكن نقله، تُنشأ قاعدة جديدة
              بجانبه ويُسجَّل مسارها للبدء القادم. لا يمكن التراجع من داخل التطبيق، لكن الملفات القديمة تبقى في مكانها
              أو في مجلد النسخ الاحتياطية.
            </p>
            <Button
              variant="danger"
              className="mt-3"
              onClick={() => setConfirmInit(true)}
              disabled={busy}
              icon={<FilePlus2 className="h-4 w-4" />}
            >
              إنشاء قاعدة بيانات جديدة…
            </Button>
          </div>
        </Card>

        <p className="mt-4 text-center text-[11px] text-slate-600">
          تُضاف تفاصيل استكشاف الأخطاء إلى application.log داخل مجلد البيانات. لا تُرسل أي بيانات مرضى إلى أي جهة.
        </p>
      </div>

      <ConfirmDialog
        open={confirmInit}
        title="إنشاء قاعدة بيانات جديدة؟"
        message="ستُنقل قاعدة البيانات الحالية وملفاتها المؤقتة إلى مجلد النسخ الاحتياطية، ثم تُنشأ قاعدة بيانات فارغة بحساب admin الافتراضي. استخدم هذا الخيار فقط عندما تكون متأكداً أن الملف الحالي تالف."
        confirmLabel="إنشاء قاعدة جديدة"
        tone="danger"
        busy={busy}
        onConfirm={initialize}
        onClose={() => setConfirmInit(false)}
      >
        <div className="rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-[11px] leading-5 text-slate-400">
          بعد الإنشاء سجّل الدخول باسم <span className="font-mono text-slate-200">admin</span> وكلمة المرور{' '}
          <span className="font-mono text-slate-200">admin123</span> ثم غيّر كلمة المرور.
        </div>
      </ConfirmDialog>
    </div>
  );
}
