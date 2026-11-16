import { useState } from 'react'
import { CalendarCheck, CalendarClock, CalendarPlus, Check, Clock3, Pencil, RefreshCw, Syringe, Users, X } from 'lucide-react'
import { api, app, errorMessage, runAction } from '../lib/api'
import { formatDate, formatInteger, formatTime, initials, todayISO } from '../lib/format'
import { useAsync } from '../lib/hooks'
import { useToast } from '../components/Toast'
import {
  Badge,
  Button,
  Card,
  CardHeader,
  ConfirmDialog,
  EmptyState,
  IconButton,
  InlineError,
  PageHeader,
  Spinner,
  StatCard,
} from '../components/ui'
import { AppointmentDialog, CancelAppointmentDialog } from '../components/AppointmentDialogs'

const statusLabels: Record<string, string> = {
  scheduled: 'مجدول',
  completed: 'مكتمل',
  cancelled: 'ملغى',
}

const statusTones: Record<string, 'amber' | 'emerald' | 'rose'> = {
  scheduled: 'amber',
  completed: 'emerald',
  cancelled: 'rose',
}

export function AppointmentsPage() {
  const { notify } = useToast()
  const today = todayISO()
  const appointments = useAsync(() => api.ListAppointments({ status: '', fromDate: today, toDate: today }), [today])
  const upcoming = useAsync(() => api.ListAppointments({ status: 'scheduled', fromDate: '', toDate: '' }), [])
  const [dialog, setDialog] = useState<{ mode: 'immediate' | 'scheduled'; appointment?: app.AppointmentDTO } | null>(null)
  const [cancelTarget, setCancelTarget] = useState<app.AppointmentDTO | null>(null)
  const [completeTarget, setCompleteTarget] = useState<app.AppointmentDTO | null>(null)
  const [completing, setCompleting] = useState(false)

  const reloadAll = () => {
    appointments.reload()
    upcoming.reload()
  }

  const onSaved = (message: string) => {
    setDialog(null)
    reloadAll()
    notify(message)
  }

  const completeAppointment = async () => {
    if (!completeTarget) {
      return
    }
    setCompleting(true)
    try {
      await runAction(
        api.CompleteAppointment(
          app.CompleteAppointmentInputDTO.createFrom({ id: completeTarget.id, administeredOn: todayISO() }),
        ),
      )
      setCompleteTarget(null)
      reloadAll()
      notify('تم إكمال الموعد.')
    } catch (error) {
      notify(errorMessage(error), 'error')
    } finally {
      setCompleting(false)
    }
  }

  const allItems = [...(appointments.data ?? [])].sort((a, b) =>
    (a.scheduledTime || '99:99').localeCompare(b.scheduledTime || '99:99'),
  )
  const items = allItems.filter((appointment) => appointment.status !== 'cancelled')
  const upcomingItems = [...(upcoming.data ?? [])]
    .filter((appointment) => appointment.scheduledOn > today)
    .sort((a, b) =>
      `${a.scheduledOn} ${a.scheduledTime || '99:99'}`.localeCompare(
        `${b.scheduledOn} ${b.scheduledTime || '99:99'}`,
      ),
    )
  const doses = items.reduce(
    (total, appointment) => total + appointment.items.reduce((sum, item) => sum + item.quantity, 0),
    0,
  )
  const patients = new Set(items.map((appointment) => appointment.patientId)).size

  const rowActions = (appointment: app.AppointmentDTO) => (
    <div className="flex justify-end gap-1">
      <IconButton label="إكمال الموعد" onClick={() => setCompleteTarget(appointment)}>
        <Check className="h-4 w-4" />
      </IconButton>
      <IconButton label="تعديل الموعد" onClick={() => setDialog({ mode: 'scheduled', appointment })}>
        <Pencil className="h-4 w-4" />
      </IconButton>
      <IconButton label="إلغاء الموعد" onClick={() => setCancelTarget(appointment)}>
        <X className="h-4 w-4" />
      </IconButton>
    </div>
  )

  return (
    <>
      <PageHeader
        title="المواعيد"
        description={`${formatDate(today)} · المواعيد المسجّلة اليوم، مع خصم الجرعات من المخزون فور التسجيل.`}
      >
        <IconButton label="تحديث المواعيد" onClick={reloadAll}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        <Button
          variant="outline"
          icon={<CalendarClock className="h-4 w-4" />}
          onClick={() => setDialog({ mode: 'scheduled' })}
        >
          حجز موعد
        </Button>
        <Button
          variant="primary"
          icon={<CalendarPlus className="h-4 w-4" />}
          onClick={() => setDialog({ mode: 'immediate' })}
        >
          تسجيل موعد فوري
        </Button>
      </PageHeader>

      <InlineError message={appointments.error || upcoming.error} />

      <div className="mb-4 grid gap-3 sm:grid-cols-3">
        <StatCard
          label="مواعيد اليوم"
          value={formatInteger(items.length)}
          detail="إجمالي المواعيد غير الملغاة"
          tone="emerald"
          icon={<CalendarCheck className="h-4 w-4" />}
        />
        <StatCard
          label="المرضى"
          value={formatInteger(patients)}
          detail="عدد المرضى غير الملغاة مواعيدهم اليوم"
          tone="sky"
          icon={<Users className="h-4 w-4" />}
        />
        <StatCard
          label="الجرعات المصروفة"
          value={formatInteger(doses)}
          detail="مجموع جرعات المواعيد غير الملغاة"
          tone="amber"
          icon={<Syringe className="h-4 w-4" />}
        />
      </div>

      <Card>
        <CardHeader title={`مواعيد اليوم (${formatInteger(allItems.length)})`} />
        {appointments.loading && !appointments.data ? (
          <Spinner label="جارٍ تحميل مواعيد اليوم…" />
        ) : allItems.length === 0 ? (
          <EmptyState
            icon={<CalendarPlus className="h-5 w-5" />}
            title="لا توجد مواعيد اليوم"
            action={
              <Button
                variant="primary"
                icon={<CalendarPlus className="h-4 w-4" />}
                onClick={() => setDialog({ mode: 'immediate' })}
              >
                تسجيل أول موعد
              </Button>
            }
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>المريض</th>
                  <th>الوقت</th>
                  <th>اللقاحات</th>
                  <th className="text-end">الجرعات</th>
                  <th>الحالة</th>
                  <th className="text-end">إجراءات</th>
                </tr>
              </thead>
              <tbody>
                {allItems.map((appointment) => {
                  const appointmentDoses = appointment.items.reduce((sum, item) => sum + item.quantity, 0)
                  return (
                    <tr key={appointment.id}>
                      <td>
                        <div className="flex items-center gap-2.5">
                          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-800 text-[11px] font-semibold text-slate-300">
                            {initials(appointment.patientName) || '—'}
                          </span>
                          <div className="min-w-0">
                            <div className="truncate font-medium text-slate-100">{appointment.patientName}</div>
                            <div className="text-[11px] text-slate-500">
                              {appointment.patientCode || 'بدون رمز'}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="whitespace-nowrap">
                        <span className="inline-flex items-center gap-1.5 rounded-md border border-slate-700/70 bg-slate-900 px-2 py-1 text-xs tabular-nums text-slate-300">
                          <Clock3 className="h-3 w-3 text-slate-500" aria-hidden />
                          {formatTime(appointment.scheduledTime)}
                        </span>
                      </td>
                      <td>
                        <div className="flex flex-wrap gap-1">
                          {appointment.items.length === 0 ? (
                            <span className="text-xs text-slate-500">—</span>
                          ) : (
                            appointment.items.map((item) => (
                              <span
                                key={item.id}
                                className="rounded-md border border-slate-700/60 bg-slate-900 px-1.5 py-0.5 text-[11px] text-slate-300"
                                title={item.vaccineCode || item.vaccineName}
                              >
                                {item.vaccineName}
                                <span className="ms-1 tabular-nums text-slate-500">×{formatInteger(item.quantity)}</span>
                              </span>
                            ))
                          )}
                        </div>
                      </td>
                      <td className="text-end text-sm font-semibold tabular-nums text-slate-200">
                        {formatInteger(appointmentDoses)}
                      </td>
                      <td>
                        <Badge tone={statusTones[appointment.status] ?? 'slate'}>
                          {statusLabels[appointment.status] ?? appointment.status}
                        </Badge>
                      </td>
                      <td>{appointment.status === 'scheduled' ? rowActions(appointment) : null}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card>
        <CardHeader
          title={`مواعيد قادمة (${formatInteger(upcomingItems.length)})`}
          action={<Badge tone={upcomingItems.length > 0 ? 'amber' : 'slate'}>مجدولة</Badge>}
        />
        {upcoming.loading && !upcoming.data ? (
          <Spinner label="جارٍ تحميل المواعيد القادمة…" />
        ) : upcomingItems.length === 0 ? (
          <EmptyState
            icon={<CalendarClock className="h-5 w-5" />}
            title="لا توجد مواعيد مجدولة قادمة"
            action={
              <Button variant="outline" onClick={() => setDialog({ mode: 'scheduled' })}>
                حجز موعد
              </Button>
            }
          />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>التاريخ</th>
                  <th>الوقت</th>
                  <th>المريض</th>
                  <th>اللقاحات</th>
                  <th className="text-end">الجرعات</th>
                  <th className="text-end">إجراءات</th>
                </tr>
              </thead>
              <tbody>
                {upcomingItems.map((appointment) => (
                  <tr key={appointment.id}>
                    <td className="whitespace-nowrap text-slate-300">{formatDate(appointment.scheduledOn)}</td>
                    <td className="whitespace-nowrap text-slate-400">{formatTime(appointment.scheduledTime)}</td>
                    <td>
                      <div className="font-medium text-slate-100">{appointment.patientName}</div>
                      <div className="text-[11px] text-slate-500">{appointment.patientCode || 'بدون رمز'}</div>
                    </td>
                    <td>
                      <div className="flex flex-wrap gap-1">
                        {appointment.items.map((item) => (
                          <span
                            key={item.id}
                            className="rounded-md border border-slate-700/60 bg-slate-900 px-1.5 py-0.5 text-[11px] text-slate-300"
                            title={item.vaccineCode || item.vaccineName}
                          >
                            {item.vaccineName}
                            <span className="ms-1 tabular-nums text-slate-500">×{formatInteger(item.quantity)}</span>
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="text-end text-sm font-semibold tabular-nums text-slate-200">
                      {formatInteger(appointment.items.reduce((sum, item) => sum + item.quantity, 0))}
                    </td>
                    <td>{rowActions(appointment)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {dialog ? (
        <AppointmentDialog
          mode={dialog.mode}
          appointment={dialog.appointment}
          onClose={() => setDialog(null)}
          onSaved={onSaved}
        />
      ) : null}
      {cancelTarget ? (
        <CancelAppointmentDialog
          appointment={cancelTarget}
          onClose={() => setCancelTarget(null)}
          onSaved={(message) => {
            setCancelTarget(null)
            reloadAll()
            notify(message)
          }}
        />
      ) : null}
      <ConfirmDialog
        open={completeTarget !== null}
        title="إكمال الموعد؟"
        message={
          completeTarget
            ? `سيُسجّل موعد ${completeTarget.patientName} كمكتمل بتاريخ اليوم. الجرعات مخصومة مسبقاً ولن تُخصم مرة أخرى.`
            : ''
        }
        confirmLabel="إكمال"
        busy={completing}
        onConfirm={completeAppointment}
        onClose={() => setCompleteTarget(null)}
      />
    </>
  )
}
