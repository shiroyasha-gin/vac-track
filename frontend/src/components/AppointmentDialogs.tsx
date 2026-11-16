import { useEffect, useMemo, useState } from 'react'
import { Minus, Plus, Search, Syringe, X } from 'lucide-react'
import { api, runAction, app } from '../lib/api'
import { formatInteger, todayISO } from '../lib/format'
import { Alert, Button, Field, Input, Modal, Spinner, Textarea } from './ui'
import { PatientPicker, type PatientChoice } from './PatientPicker'

function currentTime(): string {
  const now = new Date()
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
}

const MAX_RESULTS = 6

export function AppointmentDialog({
  mode,
  appointment,
  onClose,
  onSaved,
}: {
  mode: 'immediate' | 'scheduled'
  appointment?: app.AppointmentDTO | null
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const editing = Boolean(appointment)
  const [vaccines, setVaccines] = useState<app.VaccineDTO[]>([])
  const [loadingVaccines, setLoadingVaccines] = useState(true)
  const [patient, setPatient] = useState<PatientChoice | null>(
    appointment
      ? { id: appointment.patientId, fullName: appointment.patientName, patientCode: appointment.patientCode }
      : null,
  )
  const [vaccineQuery, setVaccineQuery] = useState('')
  const [scheduledOn, setScheduledOn] = useState(appointment?.scheduledOn || todayISO())
  const [scheduledTime, setScheduledTime] = useState(appointment?.scheduledTime || currentTime())
  const [note, setNote] = useState(appointment?.note || '')
  const [reason, setReason] = useState('')
  const [selected, setSelected] = useState<Record<number, number>>(() => {
    const initial: Record<number, number> = {}
    for (const item of appointment?.items ?? []) {
      initial[item.vaccineId] = item.quantity
    }
    return initial
  })
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    let active = true
    api
      .ListVaccines(false)
      .then((items) => {
        if (active) {
          setVaccines((items ?? []).filter((vaccine) => !vaccine.archived))
        }
      })
      .catch((cause) => {
        if (active) setError(cause instanceof Error ? cause.message : 'تعذر تحميل اللقاحات.')
      })
      .finally(() => {
        if (active) setLoadingVaccines(false)
      })
    return () => {
      active = false
    }
  }, [])

  const vaccineMatches = useMemo(() => {
    const query = vaccineQuery.trim().toLowerCase()
    return vaccines
      .filter((vaccine) => !selected[vaccine.id])
      .filter((vaccine) => {
        if (!query) return true
        return [vaccine.name, vaccine.code, vaccine.category].some((value) =>
          (value || '').toLowerCase().includes(query),
        )
      })
      .slice(0, MAX_RESULTS)
  }, [vaccines, vaccineQuery, selected])

  const selectedVaccines = useMemo(
    () =>
      Object.entries(selected)
        .map(([id, quantity]) => ({
          vaccine: vaccines.find((item) => item.id === Number(id)),
          quantity,
        }))
        .filter((item): item is { vaccine: app.VaccineDTO; quantity: number } => Boolean(item.vaccine)),
    [selected, vaccines],
  )

  const items = selectedVaccines.map(({ vaccine, quantity }) => ({
    vaccineId: vaccine.id,
    quantity,
  }))
  const total = items.reduce((sum, item) => sum + item.quantity, 0)

  const addVaccine = (id: number) => {
    setSelected((prev) => ({ ...prev, [id]: 1 }))
    setVaccineQuery('')
  }

  const removeVaccine = (id: number) => {
    setSelected((prev) => {
      const next = { ...prev }
      delete next[id]
      return next
    })
  }

  const setQty = (id: number, qty: number) => {
    setSelected((prev) => ({ ...prev, [id]: Math.max(1, qty || 1) }))
  }

  const handleSave = async () => {
    if (!patient) {
      setError('اختر مريضاً أولاً.')
      return
    }
    if (items.length === 0) {
      setError('أضف لقاحاً واحداً على الأقل.')
      return
    }
    setError('')
    setSaving(true)
    try {
      if (editing && appointment) {
        const dto = app.UpdateAppointmentInputDTO.createFrom({
          patientId: patient.id,
          scheduledOn,
          scheduledTime,
          note,
          items,
          reason,
        })
        await runAction(api.UpdateAppointment(appointment.id, dto))
        onSaved('تم تحديث الموعد وتسوية الجرعات المرتبطة به.')
        return
      }
      const dto = app.AppointmentInputDTO.createFrom({
        patientId: patient.id,
        scheduledOn,
        scheduledTime,
        note,
        items,
      })
      if (mode === 'immediate') {
        await runAction(api.CreateTakenAppointment(dto))
        onSaved('تم تسجيل الموعد كمكتمل وخصم الجرعات من المخزون.')
      } else {
        await runAction(api.CreateAppointment(dto))
        onSaved('تم حجز الموعد وخصم الجرعات من مخزون الشهر المحدد.')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'تعذر حفظ الموعد.')
    } finally {
      setSaving(false)
    }
  }

  const title = editing ? 'تعديل الموعد' : mode === 'immediate' ? 'تسجيل موعد فوري' : 'حجز موعد'
  const saveLabel = editing
    ? 'حفظ التعديلات'
    : mode === 'immediate'
      ? `تسجيل وخصم ${total > 0 ? `${formatInteger(total)} جرعة` : 'الجرعات'}`
      : `حجز وخصم ${total > 0 ? `${formatInteger(total)} جرعة` : 'الجرعات'}`

  return (
    <Modal
      open
      onClose={onClose}
      title={title}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={saving}>
            إلغاء
          </Button>
          <Button variant="primary" onClick={handleSave} loading={saving}>
            {saveLabel}
          </Button>
        </>
      }
    >
      <div className="space-y-5">
        {error ? <Alert tone="danger">{error}</Alert> : null}
        <Alert tone="info">
          {editing
            ? 'تعديل الموعد يحدّث الجرعات المرتبطة به أو يلغيها إذا أزلت اللقاح، دون تكرار الخصم.'
            : mode === 'immediate'
              ? 'سيُسجّل الموعد كمكتمل فوراً وتُخصم الجرعات المحددة من مخزون الشهر الحالي.'
              : 'سيُحجز الموعد وتُخصم الجرعات المحددة من مخزون شهر الموعد فوراً. إلغاء الموعد يعيدها.'}
        </Alert>

        <section>
          <div className="mb-2 flex items-center justify-between">
            <h4 className="text-sm font-medium text-slate-200">المريض</h4>
          </div>
          <PatientPicker value={patient} onChange={setPatient} autoFocus onError={setError} />
        </section>

        <section>
          <div className="mb-2 flex items-center justify-between">
            <h4 className="text-sm font-medium text-slate-200">اللقاحات</h4>
            <span className="text-[11px] text-slate-500">{formatInteger(selectedVaccines.length)} محدد</span>
          </div>

          {selectedVaccines.length > 0 ? (
            <div className="mb-2 space-y-1.5">
              {selectedVaccines.map(({ vaccine, quantity }) => (
                <div
                  key={vaccine.id}
                  className="flex items-center gap-2 rounded-lg border border-emerald-500/25 bg-emerald-500/5 px-2.5 py-2"
                >
                  <Syringe className="h-3.5 w-3.5 shrink-0 text-emerald-300" aria-hidden />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm text-slate-100">{vaccine.name}</p>
                    <p className="text-[11px] text-slate-500">{vaccine.code || vaccine.unit}</p>
                  </div>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => setQty(vaccine.id, quantity - 1)}
                      className="flex h-6 w-6 items-center justify-center rounded-md bg-slate-800 text-slate-300 hover:bg-slate-700"
                      aria-label={`تقليل كمية ${vaccine.name}`}
                    >
                      <Minus className="h-3 w-3" />
                    </button>
                    <Input
                      type="number"
                      min={1}
                      value={quantity}
                      onChange={(event) => setQty(vaccine.id, Number(event.target.value))}
                      className="h-7 w-14 px-1 text-center text-xs"
                      aria-label={`كمية ${vaccine.name}`}
                    />
                    <button
                      type="button"
                      onClick={() => setQty(vaccine.id, quantity + 1)}
                      className="flex h-6 w-6 items-center justify-center rounded-md bg-slate-800 text-slate-300 hover:bg-slate-700"
                      aria-label={`زيادة كمية ${vaccine.name}`}
                    >
                      <Plus className="h-3 w-3" />
                    </button>
                    <button
                      type="button"
                      onClick={() => removeVaccine(vaccine.id)}
                      className="ms-1 flex h-6 w-6 items-center justify-center rounded-md text-slate-500 hover:bg-rose-500/10 hover:text-rose-300"
                      aria-label={`إزالة ${vaccine.name}`}
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : null}

          {loadingVaccines ? (
            <Spinner label="جارٍ تحميل اللقاحات…" />
          ) : (
            <>
              <div className="relative">
                <Search className="pointer-events-none absolute start-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
                <Input
                  value={vaccineQuery}
                  onChange={(event) => setVaccineQuery(event.target.value)}
                  placeholder="ابحث عن لقاح لإضافته…"
                  className="ps-8"
                  aria-label="البحث عن لقاح"
                />
              </div>
              {vaccines.length === 0 ? (
                <p className="mt-2 text-xs text-slate-400">لا توجد لقاحات نشطة.</p>
              ) : vaccineMatches.length === 0 ? (
                <p className="mt-2 text-xs text-slate-400">
                  {vaccineQuery.trim() ? 'لا يوجد لقاح مطابق.' : 'كل اللقاحات مضافة.'}
                </p>
              ) : (
                <div className="mt-1.5 overflow-hidden rounded-lg border border-slate-700/70 bg-slate-950 shadow-panel">
                  {vaccineMatches.map((vaccine) => (
                    <button
                      key={vaccine.id}
                      type="button"
                      onClick={() => addVaccine(vaccine.id)}
                      className="flex w-full items-center gap-2.5 border-b border-slate-700/50 px-3 py-2 text-start last:border-b-0 hover:bg-slate-900"
                    >
                      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-slate-800 text-slate-400">
                        <Syringe className="h-3.5 w-3.5" />
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-sm text-slate-100">{vaccine.name}</span>
                        <span className="block text-[11px] text-slate-500">
                          {[vaccine.code, vaccine.category, vaccine.unit].filter(Boolean).join(' · ')}
                        </span>
                      </span>
                      <Plus className="h-4 w-4 text-emerald-300" aria-hidden />
                    </button>
                  ))}
                </div>
              )}
            </>
          )}
        </section>

        <div className="grid grid-cols-2 gap-3">
          <Field label="التاريخ">
            <Input
              type="date"
              value={scheduledOn}
              readOnly={mode === 'immediate' && !editing}
              onChange={(event) => setScheduledOn(event.target.value)}
              dir="ltr"
            />
          </Field>
          <Field label="الوقت">
            <Input
              type="time"
              value={scheduledTime}
              onChange={(event) => setScheduledTime(event.target.value)}
            />
          </Field>
        </div>

        <Field label="ملاحظة">
          <Textarea
            value={note}
            maxLength={300}
            onChange={(event) => setNote(event.target.value)}
            placeholder="ملاحظة اختيارية"
          />
        </Field>

        {editing ? (
          <Field label="سبب التعديل" hint="اختياري، يُسجّل في سجل التدقيق.">
            <Input
              value={reason}
              maxLength={500}
              onChange={(event) => setReason(event.target.value)}
              placeholder="مثال: طلب المريض تأجيل الموعد"
            />
          </Field>
        ) : null}

        {total > 0 ? (
          <div className="flex items-center justify-between rounded-md border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-sm">
            <span className="text-emerald-100">إجمالي الجرعات</span>
            <span className="font-semibold tabular-nums text-emerald-200">{formatInteger(total)}</span>
          </div>
        ) : null}
      </div>
    </Modal>
  )
}

export function CancelAppointmentDialog({
  appointment,
  onClose,
  onSaved,
}: {
  appointment: app.AppointmentDTO
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const [reason, setReason] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)

  const handleCancel = async () => {
    if (!reason.trim()) {
      setError('سبب الإلغاء مطلوب.')
      return
    }
    setError('')
    setSaving(true)
    try {
      const dto = app.CancelAppointmentInputDTO.createFrom({ id: appointment.id, reason })
      await runAction(api.CancelAppointment(dto))
      onSaved('تم إلغاء الموعد وإرجاع الجرعات المحجوزة إلى المخزون.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'تعذر إلغاء الموعد.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title="إلغاء الموعد"
      size="md"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={saving}>
            رجوع
          </Button>
          <Button variant="danger" onClick={handleCancel} loading={saving}>
            إلغاء الموعد
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error ? <Alert tone="danger">{error}</Alert> : null}
        <Alert tone="warning">
          سيُلغى موعد {appointment.patientName} وتُعاد الجرعات المحجوزة إلى المخزون. يبقى السجل للتدقيق.
        </Alert>
        <Field label="سبب الإلغاء" required>
          <Textarea
            value={reason}
            maxLength={500}
            onChange={(event) => setReason(event.target.value)}
            placeholder="مثال: طلب المريض إعادة الجدولة"
          />
        </Field>
      </div>
    </Modal>
  )
}
