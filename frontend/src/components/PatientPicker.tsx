import { useEffect, useState } from 'react'
import { Search, UserRound } from 'lucide-react'
import { api, errorMessage } from '../lib/api'
import { initials } from '../lib/format'
import { useDebouncedValue } from '../lib/hooks'
import { Input } from './ui'

export type PatientChoice = {
  id: number
  fullName: string
  patientCode: string
  archived?: boolean
}

const MAX_RESULTS = 6

/**
 * PatientPicker searches active patients in bounded pages. It never loads the
 * whole directory, so large clinics stay responsive.
 */
export function PatientPicker({
  value,
  onChange,
  autoFocus = false,
  disabled = false,
  onError,
}: {
  value: PatientChoice | null
  onChange: (patient: PatientChoice | null) => void
  autoFocus?: boolean
  disabled?: boolean
  onError?: (message: string) => void
}) {
  const [query, setQuery] = useState('')
  const debouncedQuery = useDebouncedValue(query, 250)
  const [results, setResults] = useState<PatientChoice[]>([])
  const [searching, setSearching] = useState(false)

  useEffect(() => {
    if (debouncedQuery.trim().length < 2) {
      setResults([])
      setSearching(false)
      return
    }
    let active = true
    setSearching(true)
    api
      .SearchPatientsLimited(debouncedQuery, MAX_RESULTS)
      .then((items) => {
        if (active) {
          setResults(items ?? [])
        }
      })
      .catch((cause: unknown) => {
        if (active) {
          onError?.(errorMessage(cause))
        }
      })
      .finally(() => {
        if (active) {
          setSearching(false)
        }
      })
    return () => {
      active = false
    }
  }, [debouncedQuery, onError])

  if (value) {
    return (
      <div className="flex items-center gap-3 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2.5">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-emerald-500/15 text-xs font-semibold text-emerald-200">
          {initials(value.fullName) || <UserRound className="h-4 w-4" />}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium text-slate-100">{value.fullName}</p>
          <p className="text-[11px] text-slate-400">
            {value.patientCode || 'بدون رمز'}
            {value.archived ? ' · مؤرشف' : ''}
          </p>
        </div>
        {disabled ? null : (
          <button
            type="button"
            className="text-xs text-emerald-300 hover:text-emerald-200"
            onClick={() => {
              onChange(null)
              setQuery('')
            }}
          >
            تغيير المريض
          </button>
        )}
      </div>
    )
  }

  return (
    <div className="relative">
      <Search className="pointer-events-none absolute start-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-slate-500" />
      <Input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder="ابحث بالاسم أو رمز المريض…"
        className="ps-8"
        autoFocus={autoFocus}
        disabled={disabled}
        aria-label="البحث عن مريض"
      />
      {query.trim().length >= 2 ? (
        <div className="mt-1.5 overflow-hidden rounded-lg border border-slate-700/70 bg-slate-950 shadow-panel">
          {searching ? (
            <p className="px-3 py-2 text-xs text-slate-400">جارٍ البحث…</p>
          ) : results.length === 0 ? (
            <p className="px-3 py-2 text-xs text-slate-400">لا يوجد مريض مطابق.</p>
          ) : (
            results.map((result) => (
              <button
                key={result.id}
                type="button"
                onClick={() => {
                  onChange(result)
                  setResults([])
                  setQuery('')
                }}
                className="flex w-full items-center gap-2.5 border-b border-slate-700/50 px-3 py-2 text-start last:border-b-0 hover:bg-slate-900"
              >
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-slate-800 text-[10px] font-semibold text-slate-300">
                  {initials(result.fullName) || <UserRound className="h-3.5 w-3.5" />}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm text-slate-100">{result.fullName}</span>
                  <span className="block text-[11px] text-slate-500">
                    {result.patientCode || 'بدون رمز'}
                  </span>
                </span>
              </button>
            ))
          )}
        </div>
      ) : (
        <p className="mt-1.5 text-[11px] text-slate-500">اكتب حرفين على الأقل للبحث في المرضى.</p>
      )}
    </div>
  )
}
