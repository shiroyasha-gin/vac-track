import { useEffect, useMemo, useState } from 'react';
import { Boxes, Check, SlidersHorizontal } from 'lucide-react';
import { api, errorMessage, runAction, type app } from '../lib/api';
import { formatInteger, monthLabel } from '../lib/format';
import { useToast } from './Toast';
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Field,
  Input,
  Modal,
  Textarea,
} from './ui';

interface StockDialogProps {
  open: boolean;
  month: string;
  row: app.MonthRowDTO | null;
  onClose: () => void;
  onSaved: () => void;
}

export function StockDialog({ open, month, row, onClose, onSaved }: StockDialogProps) {
  const { notify } = useToast();
  const [opening, setOpening] = useState('0');
  const [received, setReceived] = useState('0');
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open || !row) {
      return;
    }
    setOpening(String(row.initialized ? row.openingBalance : row.suggestedOpening));
    setReceived(String(row.initialized ? row.stockReceived : 0));
    setReason(row.carryoverReason || '');
    setError('');
    setBusy(false);
  }, [open, row]);

  const overriding = Boolean(row?.priorExists) && Number(opening) !== (row?.suggestedOpening ?? 0);
  const initializing = row ? !row.initialized : false;

  const submit = async () => {
    if (!row) {
      return;
    }
    setBusy(true);
    setError('');
    try {
      const openingValue = Number(opening);
      const receivedValue = Number(received);
      if (!Number.isInteger(openingValue) || openingValue < 0 || !Number.isInteger(receivedValue) || receivedValue < 0) {
        setError('يجب أن يكون الرصيد الافتتاحي والمخزون الوارد أعداداً صحيحة صفراً أو أكثر.');
        return;
      }
      if (initializing) {
        await runAction(
          api.InitializeInventory({
            vaccineId: row.vaccineId,
            month,
            openingBalance: openingValue,
            stockReceived: receivedValue,
            overrideReason: reason,
          }),
        );
        notify('تمت تهيئة المخزون الشهري.');
      } else {
        await runAction(
          api.UpdateStock({
            vaccineId: row.vaccineId,
            month,
            openingBalance: openingValue,
            stockReceived: receivedValue,
            overrideReason: reason,
          }),
        );
        notify('تم حفظ المخزون الشهري.');
      }
      onSaved();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initializing ? 'تهيئة المخزون الشهري' : 'تعديل المخزون الشهري'}
      description={
        row
          ? `${row.vaccineName} · ${monthLabel(month)}. ${initializing ? 'لا يوجد صف مخزون صريح بعد.' : 'تُسجل التغييرات في سجل التدقيق.'}`
          : undefined
      }
      size="md"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            إلغاء
          </Button>
          <Button variant="primary" onClick={submit} loading={busy}>
            {initializing ? 'تهيئة المخزون' : 'حفظ المخزون'}
          </Button>
        </>
      }
    >
      {row ? (
        <div className="space-y-4">
          {error ? <Alert tone="danger">{error}</Alert> : null}
          {initializing ? (
            <Alert tone="info" title="بوابة شهرية صريحة">
              ينشئ هذا صف المخزون المطلوب قبل تسجيل أي استخدام. الرصيد الافتتاحي الصفري والوارد الصفري قيمة صالحة؛
              سيظل الاستخدام مرفوضاً حتى يتوفر المخزون.
            </Alert>
          ) : null}

          <div className="rounded-xl border border-slate-700/60 bg-slate-900 px-4 py-3 text-xs text-slate-500">
            <div className="flex items-center justify-between gap-3">
              <span>الترحيل المقترح</span>
              <span className="font-semibold tabular-nums text-slate-100">{formatInteger(row.suggestedOpening)}</span>
            </div>
            <div className="mt-1 flex items-center justify-between gap-3">
              <span>المصدر</span>
              <span>
                {row.priorExists
                  ? `${monthLabel(row.priorMonth)} ${row.priorClosed ? 'الرصيد الختامي' : 'المتبقي المحسوب'}`
                  : 'لا يوجد صف للشهر السابق'}
              </span>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Field label="الرصيد الافتتاحي" required hint="جرعات كاملة مرحّلة إلى هذا الشهر.">
              <Input
                type="number"
                min={0}
                step={1}
                value={opening}
                onChange={(event) => setOpening(event.target.value)}
                className="tabular-nums"
              />
            </Field>
            <Field label="المخزون الوارد" required hint="جرعات كاملة وردت هذا الشهر.">
              <Input
                type="number"
                min={0}
                step={1}
                value={received}
                onChange={(event) => setReceived(event.target.value)}
                className="tabular-nums"
              />
            </Field>
          </div>

          {overriding ? (
            <Field
              label="سبب تجاوز الترحيل"
              required
              hint="مطلوب لأن الرصيد الافتتاحي يختلف عن الترحيل المحسوب."
            >
              <Textarea value={reason} onChange={(event) => setReason(event.target.value)} placeholder="اشرح الفرق." />
            </Field>
          ) : null}

          <p className="text-[11px] text-slate-500">
            المتبقي دائماً هو الافتتاحي + الوارد − الاستخدام الفعلي. يُرفض أي استخدام يجعل المتبقي سالباً.
          </p>
        </div>
      ) : null}
    </Modal>
  );
}

interface AdjustDialogProps {
  open: boolean;
  month: string;
  row: app.MonthRowDTO | null;
  onClose: () => void;
  onSaved: () => void;
}

export function AdjustDialog({ open, month, row, onClose, onSaved }: AdjustDialogProps) {
  const { notify } = useToast();
  const [amount, setAmount] = useState('0');
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open) {
      setAmount('0');
      setReason('');
      setError('');
      setBusy(false);
    }
  }, [open]);

  const submit = async () => {
    if (!row) {
      return;
    }
    setBusy(true);
    setError('');
    try {
      const value = Number(amount);
      if (!Number.isInteger(value) || value === 0) {
        setError('أدخل تصحيحاً صحيحاً غير صفري للجرعات. القيم السالبة تخفض المخزون الوارد.');
        return;
      }
      await runAction(
        api.AdjustStock({
          vaccineId: row.vaccineId,
          month,
          amount: value,
          reason,
        }),
      );
      notify('تم تسجيل تعديل المخزون.');
      onSaved();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="تعديل مخزون مدقق"
      description={row ? `${row.vaccineName} · ${monthLabel(month)}` : undefined}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            إلغاء
          </Button>
          <Button variant="primary" onClick={submit} loading={busy}>
            تسجيل التعديل
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error ? <Alert tone="danger">{error}</Alert> : null}
        <Alert tone="info">
          تصحح التعديلات المخزون الوارد دون إعادة كتابة السجل. يمكن تسجيل مقدار سالب فقط إذا بقي المخزون الوارد
          والرصيد الناتج غير سالبين.
        </Alert>
        <Field label="مقدار التعديل" required hint="القيمة الموجبة تضيف إلى المخزون الوارد؛ والسالبة تخصم منه.">
          <Input
            type="number"
            step={1}
            value={amount}
            onChange={(event) => setAmount(event.target.value)}
            className="tabular-nums"
          />
        </Field>
        <Field label="السبب" required hint="يُحفظ مع التعديل وحدث التدقيق.">
          <Textarea value={reason} onChange={(event) => setReason(event.target.value)} placeholder="مثال: تصحيح عدد التوريد" />
        </Field>
        {row ? (
          <div className="flex items-center justify-between rounded-lg border border-slate-700/60 bg-slate-900 px-3 py-2 text-xs text-slate-500">
            <span>المخزون الوارد الحالي</span>
            <span className="font-semibold tabular-nums text-slate-100">{formatInteger(row.stockReceived)}</span>
          </div>
        ) : null}
      </div>
    </Modal>
  );
}

interface SetupMonthDialogProps {
  open: boolean;
  month: string;
  rows: app.MonthRowDTO[];
  onClose: () => void;
  onChanged: () => void;
}

export function SetupMonthDialog({ open, month, rows, onClose, onChanged }: SetupMonthDialogProps) {
  const { notify } = useToast();
  const pending = useMemo(() => rows.filter((row) => !row.initialized && !row.archived), [rows]);
  const [values, setValues] = useState<Record<number, { opening: string; received: string; reason: string }>>({});
  const [busyId, setBusyId] = useState<number | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) {
      return;
    }
    setValues((current) => {
      const next = { ...current };
      for (const row of pending) {
        if (!next[row.vaccineId]) {
          next[row.vaccineId] = {
            opening: String(row.suggestedOpening),
            received: '0',
            reason: '',
          };
        }
      }
      return next;
    });
    setError('');
  }, [open, pending]);

  const updateValue = (id: number, field: 'opening' | 'received' | 'reason', value: string) => {
    setValues((current) => ({
      ...current,
      [id]: { ...(current[id] || { opening: '0', received: '0', reason: '' }), [field]: value },
    }));
  };

  const initialize = async (row: app.MonthRowDTO) => {
    const value = values[row.vaccineId] || { opening: String(row.suggestedOpening), received: '0', reason: '' };
    const opening = Number(value.opening);
    const received = Number(value.received);
    if (!Number.isInteger(opening) || opening < 0 || !Number.isInteger(received) || received < 0) {
      setError('يجب أن تكون كل قيم المخزون أعداداً صحيحة صفراً أو أكثر.');
      return;
    }
    setBusyId(row.vaccineId);
    setError('');
    try {
      await runAction(
        api.InitializeInventory({
          vaccineId: row.vaccineId,
          month,
          openingBalance: opening,
          stockReceived: received,
          overrideReason: value.reason,
        }),
      );
      notify(`تمت تهيئة ${row.vaccineName} لشهر ${monthLabel(month)}.`);
      onChanged();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`تهيئة المخزون الشهري · ${monthLabel(month)}`}
      description="يحتاج كل لقاح إلى صف مخزون صريح قبل تسجيل الاستخدام، حتى عندما يكون المخزون صفراً."
      size="lg"
      footer={
        <Button variant="ghost" onClick={onClose}>
          إغلاق
        </Button>
      }
    >
      {error ? <Alert tone="danger" className="mb-3">{error}</Alert> : null}
      {pending.length === 0 ? (
        <EmptyState
          icon={<Check className="h-5 w-5" />}
          title="كل اللقاحات النشطة مهيأة"
          description="هذا الشهر لديه صف مخزون صريح لكل لقاح نشط."
        />
      ) : (
        <div className="space-y-3">
          <Alert tone="info">
            تأتي قيم الافتتاحي المقترحة من الشهر السابق مباشرة. يتطلب تجاوز الاقتراح سبباً. تُسجل التهيئة في سجل
            التدقيق لكل لقاح.
          </Alert>
          {pending.map((row) => {
            const value = values[row.vaccineId] || { opening: String(row.suggestedOpening), received: '0', reason: '' };
            const overriding = row.priorExists && Number(value.opening) !== row.suggestedOpening;
            return (
              <div key={row.vaccineId} className="rounded-xl border border-slate-700/60 bg-slate-900/60 p-3">
                <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <p className="text-[13px] font-medium text-slate-100">{row.vaccineName}</p>
                    <p className="text-[11px] text-slate-500">
                      {row.vaccineCode || 'بدون رمز'} · الافتتاحي المقترح {formatInteger(row.suggestedOpening)}
                    </p>
                  </div>
                  <Badge tone="slate">{row.unit}</Badge>
                </div>
                <div className="grid grid-cols-[1fr_1fr_auto] items-end gap-2">
                  <Field label="الافتتاحي">
                    <Input
                      type="number"
                      min={0}
                      step={1}
                      value={value.opening}
                      onChange={(event) => updateValue(row.vaccineId, 'opening', event.target.value)}
                      className="tabular-nums"
                    />
                  </Field>
                  <Field label="الوارد">
                    <Input
                      type="number"
                      min={0}
                      step={1}
                      value={value.received}
                      onChange={(event) => updateValue(row.vaccineId, 'received', event.target.value)}
                      className="tabular-nums"
                    />
                  </Field>
                  <Button
                    variant="primary"
                    size="md"
                    loading={busyId === row.vaccineId}
                    icon={<Boxes className="h-4 w-4" />}
                    onClick={() => initialize(row)}
                  >
                    تهيئة
                  </Button>
                </div>
                {overriding ? (
                  <Field label="سبب التجاوز" required className="mt-2">
                    <Input
                      value={value.reason}
                      onChange={(event) => updateValue(row.vaccineId, 'reason', event.target.value)}
                      placeholder="اشرح سبب اختلاف الترحيل"
                    />
                  </Field>
                ) : null}
              </div>
            );
          })}
          <p className="flex items-center gap-1.5 text-[11px] text-slate-500">
            <SlidersHorizontal className="h-3 w-3" /> يمكن تصحيح الصفوف المهيأة ما دام الشهر مفتوحاً.
          </p>
        </div>
      )}
    </Modal>
  );
}

export function ReopenMonthDialog({
  open,
  month,
  onClose,
  onConfirmed,
}: {
  open: boolean;
  month: string;
  onClose: () => void;
  onConfirmed: () => void;
}) {
  const { notify } = useToast();
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open) {
      setReason('');
      setError('');
      setBusy(false);
    }
  }, [open]);

  const submit = async () => {
    setBusy(true);
    setError('');
    try {
      const result = await runAction(api.ReopenMonth(month, reason));
      notify(result.message);
      onConfirmed();
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={`إعادة فتح ${monthLabel(month)}`}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            إلغاء
          </Button>
          <Button variant="primary" onClick={submit} loading={busy}>
            إعادة فتح الشهر
          </Button>
        </>
      }
    >
      <div className="space-y-3">
        {error ? <Alert tone="danger">{error}</Alert> : null}
        <Alert tone="warning">
          تؤدي إعادة الفتح إلى إلغاء قفل تعديلات المخزون والاستخدام الاعتيادية لهذا الشهر. يُحفظ السبب في سجل التدقيق.
        </Alert>
        <Field label="سبب إعادة الفتح" required>
          <Textarea value={reason} onChange={(event) => setReason(event.target.value)} placeholder="لماذا يجب إعادة فتح هذا الشهر المغلق؟" />
        </Field>
      </div>
    </Modal>
  );
}
