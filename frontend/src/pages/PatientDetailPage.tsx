import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import {
  ArrowRight,
  CalendarClock,
  CalendarPlus,
  CheckCircle2,
  History,
  RefreshCw,
  ShieldAlert,
  Syringe,
  UserRound,
} from 'lucide-react'
import { api } from '../lib/api'
import { formatDate, formatInteger, formatTimestamp, initials } from '../lib/format'
import { useAsync } from '../lib/hooks'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardHeader,
  EmptyState,
  IconButton,
  InlineError,
  PageHeader,
  Spinner,
  StatCard,
} from '../components/ui'

interface VaccineCoverage {
  vaccineId: number
  name: string
  code: string
  unit: string
  doses: number
  entries: number
  latest: string
}

export function PatientDetailPage() {
  const { patientId: rawPatientId } = useParams<{ patientId: string }>()
  const navigate = useNavigate()
  const patientId = Number(rawPatientId)
  const validPatientId = /^\d+$/.test(rawPatientId ?? '') && patientId > 0
  const detail = useAsync(
    () => {
      if (!validPatientId) {
        return Promise.reject(new Error('ملف المريض غير صالح.'))
      }
      return api.PatientDetail(patientId)
    },
    [patientId, validPatientId],
  )
  const patient = detail.data?.patient
  const summary = detail.data?.summary
  const usage = useMemo(() => detail.data?.usage ?? [], [detail.data])
  const appointments = useMemo(() => detail.data?.appointments ?? [], [detail.data])

  const coverage = useMemo(() => {
    const byVaccine = new Map<number, VaccineCoverage>()
    for (const entry of usage) {
      if (entry.voided) {
        continue
      }
      const current = byVaccine.get(entry.vaccineId)
      if (current) {
        current.doses += entry.quantity
        current.entries += 1
        if (entry.administeredOn > current.latest) {
          current.latest = entry.administeredOn
        }
      } else {
        byVaccine.set(entry.vaccineId, {
          vaccineId: entry.vaccineId,
          name: entry.vaccineName,
          code: entry.vaccineCode,
          unit: entry.vaccineUnit,
          doses: entry.quantity,
          entries: 1,
          latest: entry.administeredOn,
        })
      }
    }
    return Array.from(byVaccine.values()).sort((left, right) => right.latest.localeCompare(left.latest))
  }, [usage])

  if (detail.loading && !detail.data) {
    return (
      <>
        <PageHeader title="ملف المريض" />
        <Card>
          <Spinner label="جارٍ تحميل ملف المريض…" />
        </Card>
      </>
    )
  }

  if (!detail.data || !patient || !summary) {
    return (
      <>
        <PageHeader title="ملف المريض">
          <Button variant="ghost" icon={<ArrowRight className="h-4 w-4" />} onClick={() => navigate('/patients')}>
            العودة إلى المرضى
          </Button>
        </PageHeader>
        <InlineError message={detail.error || 'لم يتم العثور على هذا المريض.'} />
        <Card>
          <EmptyState icon={<UserRound className="h-5 w-5" />} title="لا يمكن عرض هذا الملف" />
        </Card>
      </>
    )
  }

  return (
    <>
      <PageHeader title={patient.fullName}>
        <Button variant="ghost" icon={<ArrowRight className="h-4 w-4" />} onClick={() => navigate('/patients')}>
          المرضى
        </Button>
        <IconButton label="تحديث ملف المريض" onClick={detail.reload}>
          <RefreshCw className="h-4 w-4" />
        </IconButton>
        {!patient.archived ? (
          <Button variant="primary" icon={<CalendarPlus className="h-4 w-4" />} onClick={() => navigate('/appointments')}>
            موعد جديد
          </Button>
        ) : null}
      </PageHeader>

      {patient.archived ? (
        <Alert tone="warning" className="mb-4" title="ملف مؤرشف">
          هذا المريض مؤرشف. يبقى سجل الجرعات والمواعيد متاحاً للقراءة، لكن لا يمكن حجز موعد جديد.
        </Alert>
      ) : null}

      <div className="mb-4 flex items-start gap-3 rounded-xl border border-emerald-500/20 bg-emerald-500/[0.06] px-4 py-3.5">
        <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-emerald-500/10 text-sm font-semibold text-emerald-300 ring-1 ring-inset ring-emerald-500/20">
          {initials(patient.fullName) || <UserRound className="h-4 w-4" />}
        </span>
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-semibold text-slate-100">{patient.fullName}</span>
            {patient.archived ? <Badge tone="slate">مؤرشف</Badge> : <Badge tone="emerald">نشط</Badge>}
          </div>
          <p className="mt-0.5 text-xs text-slate-500">
            رمز المريض: {patient.patientCode || 'بدون رمز'} · آخر تحديث: {formatTimestamp(patient.updatedAt)}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label="إجمالي الجرعات" value={formatInteger(summary.totalDoses)} detail={summary.lastDoseOn ? `آخر جرعة: ${formatDate(summary.lastDoseOn)}` : 'لا توجد جرعات مسجلة'} tone="emerald" icon={<Syringe className="h-4 w-4" />} />
        <StatCard label="اللقاحات المستلمة" value={formatInteger(summary.vaccineCount)} detail="لقاحات مختلفة في السجل" tone="sky" icon={<ShieldAlert className="h-4 w-4" />} />
        <StatCard label="المواعيد المجدولة" value={formatInteger(summary.scheduledAppointments)} detail={summary.nextAppointmentOn ? `التالي: ${formatDate(summary.nextAppointmentOn)}` : 'لا يوجد موعد قادم'} tone="amber" icon={<CalendarClock className="h-4 w-4" />} />
        <StatCard label="المواعيد المكتملة" value={formatInteger(summary.completedAppointments)} detail="مواعيد ارتبطت بجرعات" icon={<CheckCircle2 className="h-4 w-4" />} />
      </div>

      <div className="mt-4 grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader title="التغطية باللقاح" />
          {coverage.length === 0 ? (
            <EmptyState icon={<Syringe className="h-5 w-5" />} title="لا توجد جرعات مسجلة" />
          ) : (
            <div className="table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>اللقاح</th>
                    <th className="text-end">الجرعات</th>
                    <th>آخر جرعة</th>
                  </tr>
                </thead>
                <tbody>
                  {coverage.map((item) => (
                    <tr key={item.vaccineId}>
                      <td>
                        {item.name}
                        {item.code ? <span className="ms-1.5 text-[11px] text-slate-500">{item.code}</span> : null}
                      </td>
                      <td className="text-end font-medium tabular-nums">{formatInteger(item.doses)}</td>
                      <td className="whitespace-nowrap text-xs text-slate-400">{formatDate(item.latest)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>

        <Card>
          <CardHeader title="المواعيد" />
          {appointments.length === 0 ? (
            <EmptyState icon={<CalendarClock className="h-5 w-5" />} title="لا توجد مواعيد" />
          ) : (
            <div className="table-scroll">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>التاريخ</th>
                    <th>الحالة</th>
                    <th>اللقاحات</th>
                  </tr>
                </thead>
                <tbody>
                  {appointments.map((appt) => (
                    <tr key={appt.id}>
                      <td className="whitespace-nowrap">{formatDate(appt.scheduledOn)}</td>
                      <td>
                        <Badge tone={appt.status === 'completed' ? 'emerald' : appt.status === 'cancelled' ? 'rose' : 'amber'}>
                          {appt.status === 'completed' ? 'مكتمل' : appt.status === 'cancelled' ? 'ملغى' : 'مجدول'}
                        </Badge>
                      </td>
                      <td className="text-xs text-slate-400">
                        {appt.items.map((it) => `${it.vaccineName} ×${it.quantity}`).join('، ') || '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </div>

      <Card className="mt-4">
        <CardHeader title="سجل الجرعات" />
        {usage.length === 0 ? (
          <EmptyState icon={<History className="h-5 w-5" />} title="لا يوجد استخدام مسجل" />
        ) : (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>التاريخ</th>
                  <th>اللقاح</th>
                  <th className="text-end">الجرعات</th>
                  <th>الحالة</th>
                </tr>
              </thead>
              <tbody>
                {usage.map((entry) => (
                  <tr key={entry.id} className={entry.voided ? 'opacity-60' : undefined}>
                    <td className="whitespace-nowrap">{formatDate(entry.administeredOn)}</td>
                    <td>
                      {entry.vaccineName}
                      {entry.vaccineCode ? <span className="ms-1.5 text-[11px] text-slate-500">{entry.vaccineCode}</span> : null}
                    </td>
                    <td className="text-end font-medium tabular-nums">{formatInteger(entry.quantity)}</td>
                    <td>
                      {entry.voided ? (
                        <Badge tone="rose">ملغى</Badge>
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
      </Card>
    </>
  )
}
