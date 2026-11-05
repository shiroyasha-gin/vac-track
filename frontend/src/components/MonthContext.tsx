import { createContext, useContext, useState, type ReactNode } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { currentMonth, monthLabel, shiftMonth } from '../lib/format';
import { Button, IconButton } from './ui';

interface MonthApi {
  month: string;
  setMonth: (month: string) => void;
}

const MonthContext = createContext<MonthApi | null>(null);

export function MonthProvider({ month: initial, children }: { month: string; children: ReactNode }) {
  const [month, setMonthState] = useState(initial || currentMonth());

  const setMonth = (value: string) => {
    if (/^\d{4}-\d{2}$/.test(value)) {
      setMonthState(value);
    }
  };

  return <MonthContext.Provider value={{ month, setMonth }}>{children}</MonthContext.Provider>;
}

export function useMonth(): MonthApi {
  const value = useContext(MonthContext);
  if (!value) {
    throw new Error('يجب استخدام useMonth داخل MonthProvider');
  }
  return value;
}

export function MonthPicker({ compact = false }: { compact?: boolean }) {
  const { month, setMonth } = useMonth();
  return (
    <div className="flex items-center gap-1.5">
      {!compact ? (
        <IconButton label="الشهر السابق" onClick={() => setMonth(shiftMonth(month, -1))}>
          <ChevronRight className="h-4 w-4" />
        </IconButton>
      ) : null}
      <label className="sr-only" htmlFor="global-month">
        شهر المخزون
      </label>
      <input
        id="global-month"
        type="month"
        value={month}
        onChange={(event) => setMonth(event.target.value)}
        className="focus-ring h-8 rounded-lg border border-slate-700/80 bg-slate-950/70 px-2.5 text-[13px] text-slate-100"
      />
      {!compact ? (
        <IconButton label="الشهر التالي" onClick={() => setMonth(shiftMonth(month, 1))}>
          <ChevronLeft className="h-4 w-4" />
        </IconButton>
      ) : null}
      <Button size="sm" variant="ghost" className="hidden xl:inline-flex" onClick={() => setMonth(currentMonth())}>
        هذا الشهر
      </Button>
      <span className="sr-only">{monthLabel(month)}</span>
    </div>
  );
}
