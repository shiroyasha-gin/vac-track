import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  CloudOff,
  CloudUpload,
  Eye,
  History,
  Lock,
  RefreshCw,
  ShieldCheck,
  TriangleAlert,
} from 'lucide-react';
import { ActionError, api, errorMessage, publicationAction } from '../lib/api';
import { formatInteger, formatTimestamp, monthLabel } from '../lib/format';
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
  Modal,
  PageHeader,
  Spinner,
  StatusBadge,
} from '../components/ui';

export function PublishPage() {
  const { month } = useMonth();
  const { notify } = useToast();
  const status = useAsync(() => api.PublicationStatus(), []);
  const preview = useAsync(() => api.PublicationPreview(month), [month]);

  const [confirmOpen, setConfirmOpen] = useState(false);
  const [confirmProvisional, setConfirmProvisional] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [unpublishOpen, setUnpublishOpen] = useState(false);
  const [unpublishing, setUnpublishing] = useState(false);

  const configured = Boolean(status.data?.endpointConfigured && status.data?.publisherTokenConfigured);
  const snapshot = preview.data?.snapshot;
  const provisional = preview.data ? preview.data.monthState !== 'closed' : false;

  const publish = async () => {
    setPublishing(true);
    try {
      const result = await publicationAction(api.PublishMonth(month, confirmProvisional));
      notify(result.message || 'تم نشر اللقطة العامة.');
      setConfirmOpen(false);
      setConfirmProvisional(false);
      status.reload();
      preview.reload();
    } catch (error) {
      const message = errorMessage(error);
      notify(message, 'error');
      if (error instanceof ActionError && error.result.code === 'preview_stale') {
        preview.reload();
        setConfirmOpen(false);
      }
    } finally {
      setPublishing(false);
    }
  };

  const unpublish = async () => {
    setUnpublishing(true);
    try {
      const result = await publicationAction(api.Unpublish());
      notify(result.message || 'تمت إزالة اللقطة العامة.');
      setUnpublishOpen(false);
      status.reload();
    } catch (error) {
      notify(errorMessage(error), 'error');
    } finally {
      setUnpublishing(false);
    }
  };

  return (
    <>
      <PageHeader
        title={`النشر · ${monthLabel(month)}`}
        description="راجع حمولة الإجماليات الدقيقة قبل رفعها إلى الخدمة العامة المنفصلة. لا تُرفع قاعدة بيانات سطح المكتب أبداً، ولا تكون بيانات المرضى جزءاً من أي لقطة."
      >
        <IconButton label="تحديث حالة النشر" onClick={status.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button variant="outline" icon={<RefreshCw className="h-4 w-4" />} loading={preview.loading} onClick={preview.reload}>
          تحديث المعاينة
        </Button>
        <Button
          variant="primary"
          icon={<CloudUpload className="h-4 w-4" />}
          disabled={!preview.data?.canPublish || !configured}
          onClick={() => {
            setConfirmProvisional(false);
            setConfirmOpen(true);
          }}
        >
          نشر اللقطة
        </Button>
      </PageHeader>

      <InlineError message={status.error} />

      {!configured && status.data ? (
        <Alert tone="warning" className="mb-4" title="النشر غير مهيأ">
          عيّن عنوان نشر HTTPS ورمز نشر في{' '}
          <Link to="/settings" className="font-medium underline underline-offset-2">
            الإعدادات
          </Link>{' '}
          قبل النشر. يجب نشر الخدمة العامة بشكل منفصل.
        </Alert>
      ) : null}

      {status.data?.latest ? (
        <Card className="mb-4 px-4 py-3.5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2.5">
              {status.data.latest.status === 'published' ? (
                <span className="rounded-lg bg-emerald-500/10 p-2 text-emerald-300 ring-1 ring-inset ring-emerald-500/30">
                  <ShieldCheck className="h-4 w-4" />
                </span>
              ) : (
                <span className="rounded-lg bg-slate-800 p-2 text-slate-400">
                  <CloudOff className="h-4 w-4" />
                </span>
              )}
              <div>
                <p className="text-[13px] font-medium text-slate-100">
                  {status.data.latest.status === 'published' ? 'اللقطة العامة الحالية' : 'آخر لقطة عامة'}
                </p>
                <p className="text-[11px] text-slate-500">
                  {monthLabel(status.data.latest.month)} · الإصدار {status.data.latest.version} ·{' '}
                  {status.data.latest.status === 'published'
                    ? `نُشرت ${formatTimestamp(status.data.latest.publishedAt)}`
                    : `أُزيلت ${formatTimestamp(status.data.latest.unpublishedAt)}`}
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={status.data.latest.status === 'published' ? 'emerald' : 'slate'}>
                {status.data.latest.status === 'published' ? 'مباشر' : 'غير منشور'}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                icon={<CloudOff className="h-3.5 w-3.5" />}
                disabled={status.data.latest.status !== 'published' || !configured}
                onClick={() => setUnpublishOpen(true)}
              >
                إلغاء النشر
              </Button>
            </div>
          </div>
        </Card>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
        <Card>
          <CardHeader
            title="معاينة اللقطة العامة"
            description="هذا هو بالضبط ما ستتلقاه واجهة API العامة. يتضمن إجماليات شهرية وجرعات مجمعة لكل لقاح فقط."
            action={
              <div className="flex items-center gap-2">
                <Badge tone="violet">إجماليات فقط</Badge>
                {preview.data ? <StatusBadge state={preview.data.monthState} /> : null}
              </div>
            }
          />
          <InlineError message={preview.error} />
          {preview.loading && !preview.data ? (
            <Spinner label="جارٍ بناء لقطة الإجماليات…" />
          ) : snapshot ? (
            <div className="space-y-4 p-4">
              {preview.data && preview.data.warnings.length > 0 ? (
                <Alert tone="warning" title="راجع قبل النشر">
                  <ul className="list-inside list-disc space-y-1">
                    {preview.data.warnings.map((warning) => (
                      <li key={warning}>{warning}</li>
                    ))}
                  </ul>
                </Alert>
              ) : null}

              <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
                {[
                  ['اللقاحات', formatInteger(snapshot.totals.vaccineCount)],
                  ['الافتتاحي', formatInteger(snapshot.totals.openingBalance)],
                  ['الوارد', formatInteger(snapshot.totals.stockReceived)],
                  ['الاستخدام', formatInteger(snapshot.totals.totalUsage)],
                  ['المتبقي', formatInteger(snapshot.totals.remaining)],
                ].map(([label, value]) => (
                  <div key={label} className="rounded-xl border border-slate-800 bg-slate-950/50 px-3 py-2.5">
                    <p className="text-[10px] font-semibold text-slate-500">{label}</p>
                    <p className="mt-0.5 text-lg font-semibold tabular-nums text-slate-100">{value}</p>
                  </div>
                ))}
              </div>

              <div className="table-scroll">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>اللقاح</th>
                      <th>الرمز</th>
                      <th className="text-end">الافتتاحي</th>
                      <th className="text-end">الوارد</th>
                      <th className="text-end">الاستخدام</th>
                      <th className="text-end">المتبقي</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snapshot.vaccines.map((vaccine) => (
                      <tr key={`${vaccine.code || ''}-${vaccine.name}`}>
                        <td className="font-medium text-slate-100">{vaccine.name}</td>
                        <td className="text-slate-400">{vaccine.code || '—'}</td>
                        <td className="text-end tabular-nums">{formatInteger(vaccine.openingBalance)}</td>
                        <td className="text-end tabular-nums">{formatInteger(vaccine.stockReceived)}</td>
                        <td className="text-end font-semibold tabular-nums text-emerald-300">{formatInteger(vaccine.totalUsage)}</td>
                        <td className="text-end tabular-nums">{formatInteger(vaccine.remaining)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="rounded-xl border border-slate-800 bg-slate-950/50 px-3.5 py-3 text-[11px] leading-5 text-slate-500">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone="slate">المخطط v{snapshot.schemaVersion}</Badge>
                  <span className="font-mono">{snapshot.snapshotId}</span>
                </div>
                <p className="mt-1">
                  التجميع: <span className="text-slate-300">{snapshot.aggregation === 'monthly' ? 'شهري' : snapshot.aggregation}</span> · أُنشئت{' '}
                  {formatTimestamp(snapshot.generatedAt)}
                </p>
              </div>

              <div className="flex items-start gap-2 rounded-xl border border-emerald-500/25 bg-emerald-500/5 px-3.5 py-3 text-[11px] leading-5 text-emerald-100/85">
                <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
                <span>
                  تستبعد قائمة السماح في اللقطة أسماء المرضى ومعرّفاتهم ورموزهم وأحداث الاستخدام الفردية والملاحظات
                  وأسباب الإلغاء وبيانات التدقيق الوصفية وملف SQLite نفسه.
                </span>
              </div>
            </div>
          ) : null}
        </Card>

        <div className="space-y-4">
          <Card>
            <CardHeader title="إعدادات النشر" description="تُحفظ بيانات الاعتماد خارج قاعدة البيانات." />
            <div className="space-y-3 p-4 text-[13px]">
              <div className="flex items-center justify-between gap-3">
                <span className="text-slate-400">عنوان النشر</span>
                {status.data?.endpointConfigured ? <Badge tone="emerald">مهيأ</Badge> : <Badge tone="rose">مفقود</Badge>}
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-slate-400">رمز النشر</span>
                {status.data?.publisherTokenConfigured ? <Badge tone="emerald">مهيأ</Badge> : <Badge tone="rose">مفقود</Badge>}
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-slate-400">تخزين الرمز</span>
                <span className="text-xs text-slate-300">{storageLabel(status.data?.publisherTokenStorage)}</span>
              </div>
              <Alert tone="info" className="!mt-4">
                يتطلب النشر معاينة تمت مراجعتها. إذا تغيرت بيانات الشهر بعد المعاينة، يُرفض النشر حتى تحدّث المعاينة
                وتراجعها مرة أخرى.
              </Alert>
            </div>
          </Card>

          <Card>
            <CardHeader
              title="سجل النشر"
              description="عمليات الرفع الأخيرة المسجلة في سجل التدقيق المحلي."
              action={<History className="h-4 w-4 text-slate-500" />}
            />
            {!status.data || status.data.history.length === 0 ? (
              <EmptyState icon={<CloudUpload className="h-5 w-5" />} title="لا يوجد سجل نشر بعد" />
            ) : (
              <ul className="divide-y divide-slate-800/60">
                {status.data.history.map((record) => (
                  <li key={record.id} className="flex items-start justify-between gap-3 px-4 py-2.5 text-xs">
                    <div className="min-w-0">
                      <p className="font-medium text-slate-200">
                        {monthLabel(record.month)} · v{record.version}
                      </p>
                      <p className="mt-0.5 truncate font-mono text-[10px] text-slate-500" title={record.snapshotId}>
                        {record.snapshotId}
                      </p>
                      <p className="mt-0.5 text-[10px] text-slate-500">
                        {record.status === 'published'
                          ? `نُشر ${formatTimestamp(record.publishedAt)}`
                          : `أُزيل ${formatTimestamp(record.unpublishedAt)}`}
                      </p>
                    </div>
                    <Badge tone={record.status === 'published' ? 'emerald' : 'slate'}>
                      {record.status === 'published' ? 'منشور' : 'أزيل'}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </div>

      <Modal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        title={`نشر ${monthLabel(month)}؟`}
        description="يرسل هذا لقطة إجماليات واحدة إلى خدمة النشر العامة المهيأة."
        size="md"
        footer={
          <>
            <Button variant="ghost" onClick={() => setConfirmOpen(false)} disabled={publishing}>
              إلغاء
            </Button>
            <Button
              variant="primary"
              icon={<CloudUpload className="h-4 w-4" />}
              loading={publishing}
              disabled={provisional && !confirmProvisional}
              onClick={publish}
            >
              نشر اللقطة
            </Button>
          </>
        }
      >
        <div className="space-y-4 text-[13px] text-slate-300">
          <div className="rounded-xl border border-slate-800 bg-slate-950/50 p-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-slate-500">اللقاحات في اللقطة</span>
              <span className="font-semibold tabular-nums text-slate-100">{formatInteger(snapshot?.totals.vaccineCount ?? 0)}</span>
            </div>
            <div className="mt-1 flex items-center justify-between gap-3">
              <span className="text-slate-500">إجمالي الاستخدام</span>
              <span className="font-semibold tabular-nums text-slate-100">{formatInteger(snapshot?.totals.totalUsage ?? 0)}</span>
            </div>
            <div className="mt-1 flex items-center justify-between gap-3">
              <span className="text-slate-500">معرّف اللقطة</span>
              <span className="truncate font-mono text-[10px] text-slate-400">{snapshot?.snapshotId}</span>
            </div>
          </div>

          <p>
            لا تُدرج أسماء المرضى ولا رموزهم ولا أحداث الإعطاء الفردية ولا الملاحظات. لا تستطيع الخدمة العامة الوصول
            إلى قاعدة بيانات محطة العمل هذه.
          </p>

          {provisional ? (
            <label className="flex items-start gap-2.5 rounded-xl border border-amber-500/30 bg-amber-500/5 p-3 text-amber-100">
              <input
                type="checkbox"
                checked={confirmProvisional}
                onChange={(event) => setConfirmProvisional(event.target.checked)}
                className="mt-0.5 h-4 w-4 rounded border-amber-500/60 bg-slate-950 text-amber-500 focus:ring-amber-400"
              />
              <span>
                <span className="flex items-center gap-1.5 text-xs font-semibold">
                  <TriangleAlert className="h-3.5 w-3.5" /> أرقام مؤقتة
                </span>
                هذا الشهر غير مغلق. أفهم أن الإجماليات المنشورة قد تتغير حتى يُغلق الشهر ويُعاد نشرها.
              </span>
            </label>
          ) : (
            <p className="flex items-center gap-1.5 text-xs text-slate-500">
              <Lock className="h-3.5 w-3.5" /> الشهر مغلق، لذا تُحظر التعديلات الاعتيادية حتى يُعاد فتحه.
            </p>
          )}

          <p className="text-[11px] text-slate-500">
            تتم إزالة تكرار النشر بواسطة معرّف اللقطة. إعادة المحاولة بعد فشل الشبكة آمنة.
          </p>
        </div>
      </Modal>

      <ConfirmDialog
        open={unpublishOpen}
        title="إزالة اللقطة العامة؟"
        message="سيتوقف الموقع العام عن عرض الإحصائيات الشهرية فوراً. لا تُحذف أي بيانات محلية، ويمكنك النشر مجدداً بعد مراجعة معاينة جديدة."
        confirmLabel="إلغاء النشر"
        tone="danger"
        busy={unpublishing}
        onConfirm={unpublish}
        onClose={() => setUnpublishOpen(false)}
      >
        <div className="flex items-start gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-2.5 text-xs text-slate-400">
          <Eye className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          تعرض الواجهة العامة رسالة &quot;لا توجد إحصائيات شهرية منشورة بعد&quot; بعد الإزالة.
        </div>
      </ConfirmDialog>
    </>
  );
}

function storageLabel(storage: string | undefined): string {
  if (storage === 'os_keyring') return 'سلسلة مفاتيح نظام التشغيل';
  if (storage === 'local_file') return 'ملف محلي خاص';
  if (storage) return storage;
  return 'غير مخزن';
}
