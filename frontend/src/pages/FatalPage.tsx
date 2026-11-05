import { useState } from 'react';
import { AlertTriangle, DatabaseBackup, Loader2, RotateCcw, ShieldAlert } from 'lucide-react';
import { api, errorMessage, type app } from '../lib/api';
import { Alert, Button, Card, KeyValue } from '../components/ui';

export function FatalPage({ bootstrap, onReload }: { bootstrap: app.BootstrapInfo; onReload: () => void }) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState('');

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

  const retry = () => {
    setBusy(true);
    onReload();
  };

  return (
    <div className="flex min-h-full items-center justify-center bg-slate-950 px-6 py-10">
      <div className="w-full max-w-2xl">
        <div className="mb-6 flex items-center gap-3">
          <div className="rounded-xl bg-rose-500/10 p-2.5 text-rose-300 ring-1 ring-inset ring-rose-500/30">
            <ShieldAlert className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-slate-100">قاعدة البيانات المحلية غير متاحة</h1>
            <p className="mt-0.5 text-[13px] text-slate-400">
              لم تُفقد أي بيانات. استعد نسخة احتياطية تم التحقق منها، أو أعد المحاولة بعد إصلاح مجلد البيانات.
            </p>
          </div>
        </div>

        <Card className="p-5">
          <Alert tone="danger" title="خطأ في بدء التشغيل">
            <p className="whitespace-pre-wrap break-words">{bootstrap.fatalError || 'خطأ بدء تشغيل غير معروف.'}</p>
          </Alert>

          <div className="mt-4 rounded-xl border border-slate-800 bg-slate-950/50 px-4">
            <KeyValue label="مجلد بيانات التطبيق">{bootstrap.dataDir || 'غير متاح'}</KeyValue>
            <KeyValue label="ملف قاعدة البيانات">{bootstrap.databasePath || 'غير متاح'}</KeyValue>
            <KeyValue label="إصدار التطبيق">v{bootstrap.version}</KeyValue>
          </div>

          <div className="mt-4 rounded-xl border border-amber-500/25 bg-amber-500/5 p-3 text-xs leading-5 text-amber-100/90">
            <p className="flex items-center gap-1.5 font-medium text-amber-200">
              <AlertTriangle className="h-3.5 w-3.5" /> سلامة الاستعادة
            </p>
            <p className="mt-1">
              تتحقق عملية الاستعادة من النسخة الاحتياطية قبل تغيير أي شيء. تُحفظ قاعدة البيانات الحالية في مجلد النسخ
              الاحتياطية كنسخة سابقة للاستعادة. لا تستعد إلا نسخة احتياطية من محطة العمل هذه.
            </p>
          </div>

          {message ? (
            <p className="mt-4 rounded-lg bg-slate-800/70 px-3 py-2 text-xs text-slate-200" role="status">
              {message}
            </p>
          ) : null}

          <div className="mt-5 flex flex-wrap items-center gap-2">
            <Button variant="primary" onClick={restore} disabled={busy} icon={busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <DatabaseBackup className="h-4 w-4" />}>
              استعادة نسخة احتياطية…
            </Button>
            <Button variant="outline" onClick={retry} disabled={busy} icon={<RotateCcw className="h-4 w-4" />}>
              إعادة محاولة البدء
            </Button>
          </div>
        </Card>

        <p className="mt-4 text-center text-[11px] text-slate-600">
          تُضاف تفاصيل استكشاف الأخطاء إلى application.log داخل مجلد البيانات. لا تُرسل أي بيانات مرضى إلى أي جهة.
        </p>
      </div>
    </div>
  );
}
