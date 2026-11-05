import type { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import {
  BarChart3,
  CalendarDays,
  Database,
  ScrollText,
  Settings,
  ShieldCheck,
  Syringe,
  Upload,
  Users,
} from 'lucide-react';
import type { app } from '../lib/api';
import { MonthPicker } from './MonthContext';
import { cn } from './ui';

const navigation = [
  { to: '/', label: 'الشهري', icon: CalendarDays, end: true },
  { to: '/vaccines', label: 'اللقاحات', icon: Syringe },
  { to: '/patients', label: 'المرضى', icon: Users },
  { to: '/reports', label: 'التقارير', icon: BarChart3 },
  { to: '/publish', label: 'النشر', icon: Upload },
  { to: '/audit', label: 'سجل التدقيق', icon: ScrollText },
  { to: '/settings', label: 'الإعدادات', icon: Settings },
];

export function Shell({ bootstrap, children }: { bootstrap: app.BootstrapInfo; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 bg-slate-950">
      <aside className="flex w-[236px] shrink-0 flex-col border-e border-slate-800/80 bg-slate-950/90">
        <div className="flex items-center gap-3 px-5 py-5">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-500 text-emerald-950 shadow-lg shadow-emerald-500/20">
            <Syringe className="h-5 w-5" strokeWidth={2.4} />
          </div>
          <div className="leading-tight">
            <p className="text-sm font-semibold text-slate-100">مخزون اللقاحات</p>
            <p className="text-[11px] text-slate-500">نسخة سطح المكتب المحلية</p>
          </div>
        </div>

        <nav className="mt-2 flex-1 space-y-0.5 px-3" aria-label="التنقل الرئيسي">
          {navigation.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  'focus-ring flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors',
                  isActive
                    ? 'bg-slate-800/80 text-slate-100 shadow-sm'
                    : 'text-slate-400 hover:bg-slate-900 hover:text-slate-200',
                )
              }
            >
              <Icon className="h-4 w-4 shrink-0" aria-hidden />
              {label}
            </NavLink>
          ))}
        </nav>

        <div className="space-y-2 border-t border-slate-800/70 px-4 py-4">
          <div className="flex items-start gap-2 rounded-lg bg-emerald-500/5 px-2.5 py-2 text-[11px] leading-4 text-emerald-200/80 ring-1 ring-inset ring-emerald-500/15">
            <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
            <span>بيانات محلية خاصة. لا يتم رفع سجلات المرضى أو نشرها أبداً.</span>
          </div>
          <div className="flex items-center gap-1.5 px-1 text-[11px] text-slate-600" title={bootstrap.databasePath}>
            <Database className="h-3 w-3" aria-hidden />
            <span className="truncate">الإصدار {bootstrap.version} · محطة عمل واحدة</span>
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-[58px] shrink-0 items-center justify-between gap-4 border-b border-slate-800/80 bg-slate-950/80 px-6 backdrop-blur">
          <div className="flex items-center gap-2 text-xs text-slate-500">
            <span className="relative flex h-1.5 w-1.5">
              <span className="absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-60" />
              <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-400" />
            </span>
            قاعدة البيانات المحلية متصلة
          </div>
          <MonthPicker />
        </header>
        <main className="min-h-0 flex-1 overflow-y-auto px-6 py-5">
          <div className="mx-auto w-full max-w-[1600px]">{children}</div>
        </main>
      </div>
    </div>
  );
}
