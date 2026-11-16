import { useId } from 'react'
import type { app } from '../lib/api'
import { formatInteger } from '../lib/format'

/** DailyUsageChart renders a compact area chart of daily administered doses. */
export function DailyUsageChart({ data }: { data: app.DailyTotalDTO[] }) {
  const gradientId = useId()
  const width = 720
  const height = 210
  const pad = { top: 14, right: 10, bottom: 26, left: 34 }
  const plotWidth = width - pad.left - pad.right
  const plotHeight = height - pad.top - pad.bottom
  const max = Math.max(1, ...data.map((point) => point.total))
  const baseline = pad.top + plotHeight
  const step = data.length > 1 ? plotWidth / (data.length - 1) : 0

  const points = data.map((point, index) => ({
    ...point,
    x: pad.left + index * step,
    y: pad.top + (1 - point.total / max) * plotHeight,
  }))
  const line = points.map((point, index) => `${index === 0 ? 'M' : 'L'}${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(' ')
  const area =
    points.length > 0
      ? `${line} L${points[points.length - 1].x.toFixed(1)},${baseline} L${points[0].x.toFixed(1)},${baseline} Z`
      : ''

  const labelIndices = Array.from(
    new Set([0, Math.floor((data.length - 1) / 4), Math.floor((data.length - 1) / 2), Math.floor(((data.length - 1) * 3) / 4), data.length - 1]),
  ).filter((index) => index >= 0)

  return (
    <div>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="h-auto w-full"
        role="img"
        aria-label="الاتجاه اليومي للجرعات المستخدمة خلال الشهر"
      >
        <defs>
          <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#1d7a52" stopOpacity="0.22" />
            <stop offset="100%" stopColor="#1d7a52" stopOpacity="0" />
          </linearGradient>
        </defs>

        {[0, 0.5, 1].map((ratio) => {
          const y = pad.top + ratio * plotHeight
          return (
            <g key={ratio}>
              <line
                x1={pad.left}
                x2={width - pad.right}
                y1={y}
                y2={y}
                className="stroke-slate-700"
                strokeDasharray={ratio === 1 ? undefined : '3 4'}
                strokeWidth="1"
              />
              <text x={pad.left - 6} y={y + 3.5} textAnchor="end" fontSize="9" className="fill-slate-500">
                {formatInteger(Math.round(max * (1 - ratio)))}
              </text>
            </g>
          )
        })}

        {area ? <path d={area} fill={`url(#${gradientId})`} /> : null}
        {line ? (
          <path d={line} fill="none" className="stroke-emerald-500" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
        ) : null}

        {points
          .filter((point) => point.total > 0)
          .map((point) => (
            <circle key={point.date} cx={point.x} cy={point.y} r="2.6" className="fill-emerald-500 stroke-slate-950" strokeWidth="1.5">
              <title>{`اليوم ${point.day}: ${formatInteger(point.total)} جرعة`}</title>
            </circle>
          ))}

        {labelIndices.map((index) => {
          const point = points[index]
          if (!point) return null
          return (
            <text key={point.date} x={point.x} y={height - 7} textAnchor="middle" fontSize="9" className="fill-slate-500">
              {point.day}
            </text>
          )
        })}
      </svg>
      <p className="mt-1 text-center text-[11px] text-slate-500">يوم الشهر</p>
    </div>
  )
}

/** UsageComparisonChart compares this month's usage with the previous month. */
export function UsageComparisonChart({ rows }: { rows: app.VaccineStatDTO[] }) {
  const ranked = [...rows]
    .filter((row) => row.usageTotal > 0 || row.previousUsage > 0)
    .sort((a, b) => b.usageTotal - a.usageTotal || b.previousUsage - a.previousUsage)
    .slice(0, 8)
  const max = Math.max(1, ...ranked.flatMap((row) => [row.usageTotal, row.previousUsage]))

  if (ranked.length === 0) {
    return <p className="px-4 py-8 text-center text-xs text-slate-500">لا توجد بيانات استخدام للمقارنة.</p>
  }

  return (
    <div className="space-y-3 px-4 py-4">
      <div className="flex flex-wrap items-center gap-4 text-[11px] text-slate-500">
        <span className="inline-flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-emerald-400" /> الشهر الحالي
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-slate-500" /> الشهر السابق
        </span>
      </div>
      {ranked.map((row) => (
        <div key={row.vaccineId} className="grid grid-cols-[minmax(0,7.5rem)_minmax(0,1fr)_auto] items-center gap-3">
          <span className="truncate text-xs text-slate-300" title={row.name}>
            {row.name}
          </span>
          <div className="space-y-1">
            <div className="h-1.5 overflow-hidden rounded-full bg-slate-800">
              <div className="h-full rounded-full bg-emerald-400" style={{ width: `${(row.usageTotal / max) * 100}%` }} />
            </div>
            <div className="h-1 overflow-hidden rounded-full bg-slate-800">
              <div className="h-full rounded-full bg-slate-500" style={{ width: `${(row.previousUsage / max) * 100}%` }} />
            </div>
          </div>
          <span className="min-w-[3rem] text-end text-[11px] tabular-nums text-slate-300">
            {formatInteger(row.usageTotal)}
          </span>
        </div>
      ))}
    </div>
  )
}

/** StockHealthChart shows the share of adequate, low, and uninitialized stock. */
export function StockHealthChart({
  healthy,
  low,
  uninitialized,
}: {
  healthy: number
  low: number
  uninitialized: number
}) {
  const total = healthy + low + uninitialized
  const segments = [
    { label: 'مخزون كافٍ', value: healthy, color: '#1d7a52' },
    { label: 'مخزون منخفض', value: low, color: '#b06b12' },
    { label: 'غير مهيأ', value: uninitialized, color: '#d4d9e3' },
  ]
  let cursor = 0
  const stops = segments
    .filter((segment) => segment.value > 0)
    .map((segment) => {
      const start = cursor
      cursor += (segment.value / Math.max(1, total)) * 100
      return `${segment.color} ${start}% ${cursor}%`
    })
    .join(', ')
  const healthyPercent = total > 0 ? Math.round((healthy / total) * 100) : 0

  return (
    <div className="flex flex-wrap items-center justify-center gap-6 px-4 py-5">
      <div
        className="relative h-32 w-32 shrink-0 rounded-full"
        style={{ background: stops ? `conic-gradient(${stops})` : '#e8ebf1' }}
        role="img"
        aria-label={`صحة المخزون: ${healthyPercent}% مخزون كافٍ`}
      >
        <div className="absolute inset-[15px] flex flex-col items-center justify-center rounded-full bg-slate-950">
          <span className="text-2xl font-semibold tabular-nums text-slate-100">{healthyPercent}%</span>
          <span className="text-[10px] text-slate-500">مخزون كافٍ</span>
        </div>
      </div>
      <ul className="space-y-2 text-xs">
        {segments.map((segment) => (
          <li key={segment.label} className="flex items-center gap-2">
            <span className="h-2.5 w-2.5 rounded-sm" style={{ backgroundColor: segment.color }} />
            <span className="text-slate-400">{segment.label}</span>
            <span className="ms-auto font-medium tabular-nums text-slate-200">{formatInteger(segment.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
