import { useState, type FormEvent } from 'react';
import {
  ArrowLeft,
  Database,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  LockKeyhole,
  ShieldCheck,
  Syringe,
  UserRound,
  WifiOff,
} from 'lucide-react';
import { api, errorMessage, type app } from '../lib/api';
import { Input } from '../components/ui';

const highlights = [
  {
    icon: Database,
    title: 'بيانات محلية',
    detail: 'سجلات المرضى تبقى على محطة العمل هذه.',
  },
  {
    icon: WifiOff,
    title: 'عمل دون اتصال',
    detail: 'لا يعتمد العمل اليومي على الشبكة.',
  },
  {
    icon: ShieldCheck,
    title: 'نسخ احتياطي موثّق',
    detail: 'يُتحقق من النسخة كاملة قبل الاستعادة.',
  },
];

export function LoginPage({
  version,
  onSuccess,
}: {
  version: string;
  onSuccess: (state: app.AuthStateDTO) => void;
}) {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (busy) {
      return;
    }
    if (!username.trim() || !password) {
      setError('أدخل اسم المستخدم وكلمة المرور.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const state = await api.Login(username, password);
      if (!state.authenticated) {
        setError('تعذر بدء الجلسة. حاول مرة أخرى.');
        return;
      }
      onSuccess(state);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-full bg-slate-900">
      <aside className="relative hidden w-[46%] flex-col justify-between overflow-hidden bg-emerald-950 p-10 text-white lg:flex xl:p-14">
        <div
          className="pointer-events-none absolute inset-0 opacity-[0.14]"
          style={{
            backgroundImage:
              'linear-gradient(to bottom, rgba(255,255,255,0.55) 1px, transparent 1px), linear-gradient(to right, rgba(255,255,255,0.3) 1px, transparent 1px)',
            backgroundSize: '100% 46px, 46px 100%',
          }}
          aria-hidden
        />
        <svg
          className="pointer-events-none absolute -bottom-24 -start-20 h-[420px] w-[420px] text-white opacity-[0.07]"
          viewBox="0 0 200 200"
          fill="none"
          aria-hidden
        >
          <circle cx="100" cy="100" r="94" stroke="currentColor" strokeWidth="0.8" />
          <circle cx="100" cy="100" r="70" stroke="currentColor" strokeWidth="0.8" />
          <circle cx="100" cy="100" r="46" stroke="currentColor" strokeWidth="0.8" />
          <circle cx="100" cy="100" r="22" stroke="currentColor" strokeWidth="0.8" />
          <path d="M100 6v22M100 172v22M6 100h22M172 100h22" stroke="currentColor" strokeWidth="0.8" />
          <circle cx="100" cy="100" r="4" fill="currentColor" />
        </svg>

        <div className="relative flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/10 ring-1 ring-inset ring-white/15">
            <Syringe className="h-5 w-5" strokeWidth={2.2} />
          </div>
          <div className="leading-tight">
            <p className="text-[15px] font-semibold">VacTrack</p>
            <p className="mt-0.5 text-[11px] text-white/55">إدارة مخزون اللقاحات</p>
          </div>
        </div>

        <div className="relative max-w-md">
          <p className="text-[11px] font-medium text-white/55">سجل تحصين لمحطة عمل واحدة</p>
          <h2 className="mt-4 text-[40px] font-semibold leading-[1.15] tracking-tight xl:text-[46px]">
            كل جرعة في مكانها.
          </h2>
          <p className="mt-5 text-[13px] leading-7 text-white/65">
            مخزون شهري، استخدامات المرضى، وتقارير دقيقة — في سجل واحد خاص لا يغادر هذا الجهاز إلا عندما تختار
            مشاركته.
          </p>
        </div>

        <ul className="relative space-y-4 border-t border-white/10 pt-6">
          {highlights.map(({ icon: Icon, title, detail }) => (
            <li key={title} className="flex items-start gap-3">
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-white/[0.06] text-white/70 ring-1 ring-inset ring-white/10">
                <Icon className="h-4 w-4" aria-hidden />
              </span>
              <span className="min-w-0">
                <span className="block text-xs font-medium text-white/85">{title}</span>
                <span className="mt-0.5 block text-[11.5px] leading-5 text-white/50">{detail}</span>
              </span>
            </li>
          ))}
        </ul>
      </aside>

      <main className="flex min-w-0 flex-1 items-center justify-center px-6 py-10 sm:px-10">
        <div className="w-full max-w-[380px] animate-slide-in-up">
          <div className="mb-9 flex items-center gap-3 lg:hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500 text-white shadow-sm">
              <Syringe className="h-5 w-5" strokeWidth={2.2} />
            </div>
            <div className="leading-tight">
              <p className="text-[15px] font-semibold text-slate-100">VacTrack</p>
              <p className="mt-0.5 text-[11px] text-slate-500">إدارة مخزون اللقاحات</p>
            </div>
          </div>

          <h1 className="text-[30px] font-semibold leading-tight tracking-tight text-slate-100">تسجيل الدخول</h1>
          <p className="mt-2 text-[13px] leading-6 text-slate-500">
            استخدم حساب المشغّل المحلي للوصول إلى سجل المحطة.
          </p>

          <form className="mt-8 space-y-4" onSubmit={submit} noValidate>
            {error ? (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-xl border border-rose-500/25 bg-rose-500/[0.06] px-3.5 py-3 text-[13px] leading-6 text-slate-200"
              >
                <LockKeyhole className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
                <span>{error}</span>
              </div>
            ) : null}

            <div className="space-y-1.5">
              <label htmlFor="login-username" className="block text-xs font-medium text-slate-200">
                اسم المستخدم
              </label>
              <div className="relative">
                <UserRound
                  className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500"
                  aria-hidden
                />
                <Input
                  id="login-username"
                  name="username"
                  autoComplete="username"
                  autoFocus
                  className="h-11 ps-10 text-sm"
                  value={username}
                  onChange={(event) => {
                    setUsername(event.target.value);
                    setError('');
                  }}
                  disabled={busy}
                  placeholder="اسم المستخدم"
                  spellCheck={false}
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label htmlFor="login-password" className="block text-xs font-medium text-slate-200">
                كلمة المرور
              </label>
              <div className="relative">
                <KeyRound
                  className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500"
                  aria-hidden
                />
                <Input
                  id="login-password"
                  name="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  className="h-11 ps-10 pe-11 text-sm"
                  value={password}
                  onChange={(event) => {
                    setPassword(event.target.value);
                    setError('');
                  }}
                  disabled={busy}
                  placeholder="كلمة المرور"
                />
                <button
                  type="button"
                  className="focus-ring absolute end-1.5 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-900 hover:text-slate-300"
                  onClick={() => setShowPassword((value) => !value)}
                  aria-label={showPassword ? 'إخفاء كلمة المرور' : 'إظهار كلمة المرور'}
                  aria-pressed={showPassword}
                  disabled={busy}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            <button
              type="submit"
              disabled={busy}
              className="focus-ring inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-emerald-500 text-sm font-medium text-white shadow-sm transition-all duration-150 hover:bg-emerald-600 active:scale-[0.99] active:bg-emerald-600 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              {busy ? 'جارٍ التحقق…' : 'دخول'}
              {!busy ? <ArrowLeft className="h-4 w-4" aria-hidden /> : null}
            </button>
          </form>

          <div className="mt-8 flex items-start gap-2 rounded-xl border border-slate-700/70 bg-slate-950 px-3.5 py-3 text-[11.5px] leading-5 text-slate-500">
            <ShieldCheck className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-500" aria-hidden />
            <p>تُتحقق بيانات الدخول على هذا الجهاز فقط. لا تُرسل كلمة المرور ولا سجلات المرضى إلى أي خادم.</p>
          </div>

          <div className="mt-8 flex items-center justify-between border-t border-slate-700/60 pt-4 text-[11px] text-slate-500">
            <span>الإصدار {version}</span>
            <span>محطة عمل واحدة</span>
          </div>
        </div>
      </main>
    </div>
  );
}
