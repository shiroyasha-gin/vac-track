import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import {
  BarChart3,
  CalendarCheck,
  CalendarDays,
  Database,
  LogOut,
  Menu,
  Settings,
  ShieldAlert,
  ShieldCheck,
  Syringe,
  UserRound,
  Users,
  X,
} from 'lucide-react';
import type { app } from '../lib/api';
import { MonthPicker } from './MonthContext';
import { cn, IconButton } from './ui';

const navigation = [
  { to: '/', label: 'الشهري', icon: CalendarDays, end: true },
  { to: '/appointments', label: 'المواعيد', icon: CalendarCheck },
  { to: '/patients', label: 'المرضى', icon: Users },
  { to: '/vaccines', label: 'اللقاحات', icon: Syringe },
  { to: '/reports', label: 'التقارير', icon: BarChart3 },
];

function Navigation({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <nav className="flex-1 space-y-0.5 px-3" aria-label="التنقل الرئيسي">
      {navigation.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          onClick={onNavigate}
          className={({ isActive }) =>
            cn(
              'focus-ring relative flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors duration-150',
              isActive
                ? 'bg-emerald-500/10 text-emerald-300'
                : 'text-slate-500 hover:bg-slate-900 hover:text-slate-200',
            )
          }
        >
          {({ isActive }) => (
            <>
              <span
                className={cn(
                  'absolute inset-y-2 start-0 w-0.5 rounded-full bg-emerald-500 transition-opacity duration-150',
                  isActive ? 'opacity-100' : 'opacity-0',
                )}
                aria-hidden
              />
              <Icon className="h-4 w-4 shrink-0" aria-hidden />
              {label}
            </>
          )}
        </NavLink>
      ))}
    </nav>
  );
}

export function Shell({
  bootstrap,
  user,
  usingDefaultPassword,
  onLogout,
  children,
}: {
  bootstrap: app.BootstrapInfo;
  user: app.AuthUserDTO;
  usingDefaultPassword: boolean;
  onLogout: () => void;
  children: ReactNode;
}) {
  const [navOpen, setNavOpen] = useState(false);
  const location = useLocation();
  const mainRef = useRef<HTMLElement>(null);

  useEffect(() => {
    mainRef.current?.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }, [location.pathname]);

  return (
    <div className="flex h-full min-h-0 bg-slate-900">
      <a
        href="#main-content"
        className="focus-ring sr-only focus:not-sr-only focus:absolute focus:start-3 focus:top-3 focus:z-[100] focus:rounded-lg focus:border focus:border-slate-700 focus:bg-slate-950 focus:px-3 focus:py-2 focus:text-sm focus:text-slate-100"
      >
        تجاوز إلى المحتوى
      </a>

      {navOpen ? (
        <button
          type="button"
          aria-label="إغلاق القائمة"
          className="fixed inset-0 z-40 bg-[rgba(37,42,61,0.4)] backdrop-blur-sm lg:hidden"
          onClick={() => setNavOpen(false)}
        />
      ) : null}

      <aside
        className={cn(
          'fixed inset-y-0 start-0 z-50 w-[252px] shrink-0 flex-col border-e border-slate-700/60 bg-slate-950',
          navOpen ? 'flex' : 'hidden',
          'lg:static lg:flex',
        )}
      >
        <div className="flex items-center gap-3 px-5 py-5">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-500 text-white shadow-sm">
            <Syringe className="h-5 w-5" strokeWidth={2.3} />
          </div>
          <div className="min-w-0 leading-tight">
            <p className="text-[15px] font-semibold text-slate-100">VacTrack</p>
            <p className="mt-0.5 text-[11px] text-slate-500">إدارة مخزون اللقاحات</p>
          </div>
          <IconButton label="إغلاق القائمة" className="ms-auto lg:hidden" onClick={() => setNavOpen(false)}>
            <X className="h-4 w-4" />
          </IconButton>
        </div>

        <div className="mt-1 px-3">
          <p className="px-3 pb-1.5 pt-2 text-[11px] font-medium text-slate-600">العمل اليومي</p>
        </div>
        <Navigation onNavigate={() => setNavOpen(false)} />

        <div className="mt-2 border-t border-slate-700/60 px-3 pt-2">
          <NavLink
            to="/settings"
            onClick={() => setNavOpen(false)}
            className={({ isActive }) =>
              cn(
                'focus-ring relative flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors duration-150',
                isActive
                  ? 'bg-emerald-500/10 text-emerald-300'
                  : 'text-slate-500 hover:bg-slate-900 hover:text-slate-200',
              )
            }
          >
            <Settings className="h-4 w-4 shrink-0" aria-hidden />
            الإعدادات
          </NavLink>
        </div>

        <div className="space-y-3 border-t border-slate-700/60 px-4 py-4">
          <div className="flex items-center gap-2.5 rounded-lg border border-slate-700/70 bg-slate-900 px-3 py-2.5">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-300 ring-1 ring-inset ring-emerald-500/20">
              <UserRound className="h-4 w-4" aria-hidden />
            </div>
            <div className="min-w-0 flex-1 leading-tight">
              <p className="truncate text-xs font-medium text-slate-100">{user.displayName}</p>
              <p className="mt-0.5 text-[10.5px] text-slate-500">{user.role === 'admin' ? 'مسؤول' : 'موظف'}</p>
            </div>
            <IconButton label="تسجيل الخروج" onClick={onLogout}>
              <LogOut className="h-4 w-4" />
            </IconButton>
          </div>
          <div className="flex items-start gap-2 rounded-lg bg-slate-900 px-3 py-2.5 text-[11px] leading-5 text-slate-500">
            <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" aria-hidden />
            <span>بيانات محلية خاصة. لا يتم رفع سجلات المرضى أو نشرها أبداً.</span>
          </div>
          <div className="flex items-center justify-between gap-2 px-1 text-[11px] text-slate-600">
            <span className="flex min-w-0 items-center gap-1.5" title={bootstrap.databasePath}>
              <Database className="h-3 w-3 shrink-0" aria-hidden />
              <span className="truncate">الإصدار {bootstrap.version}</span>
            </span>
            <span className="shrink-0">محطة واحدة</span>
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="app-chrome flex h-[60px] shrink-0 items-center justify-between gap-3 border-b border-slate-700/60 bg-slate-950/85 px-4 backdrop-blur sm:px-6">
          <div className="flex min-w-0 items-center gap-2">
            <IconButton label="فتح القائمة" className="lg:hidden" onClick={() => setNavOpen(true)}>
              <Menu className="h-4 w-4" />
            </IconButton>
            <span className="hidden items-center gap-2 text-xs text-slate-500 sm:flex">
              <span className="relative flex h-1.5 w-1.5">
                <span className="absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-40" />
                <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
              </span>
              قاعدة البيانات المحلية متصلة
            </span>
          </div>
          <div className="flex items-center gap-2">
            <span className="hidden text-[11px] text-slate-500 md:inline">شهر المخزون</span>
            <MonthPicker />
          </div>
        </header>
        <main
          ref={mainRef}
          id="main-content"
          className="app-scroll min-h-0 flex-1 overflow-y-auto px-4 py-5 sm:px-6 lg:px-8 lg:py-6"
        >
          <div className="mx-auto w-full max-w-[1560px]">
            {usingDefaultPassword ? (
              <div className="no-print mb-4 flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-xl border border-amber-500/25 bg-amber-500/[0.07] px-3.5 py-2.5 text-[12px] leading-5 text-slate-200">
                <ShieldAlert className="h-4 w-4 shrink-0 text-amber-300" aria-hidden />
                <span className="min-w-0 flex-1">
                  ما زلت تستخدم كلمة المرور الافتراضية. غيّرها من إعدادات الأمان قبل إدخال بيانات المرضى.
                </span>
                <Link
                  to="/settings"
                  className="focus-ring rounded-md px-2 py-0.5 text-[11.5px] font-medium text-amber-300 hover:bg-amber-500/10"
                >
                  فتح إعدادات الأمان
                </Link>
              </div>
            ) : null}
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
