import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react';
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react';
import { cn } from './ui';

type ToastTone = 'success' | 'error' | 'info';

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

interface ToastApi {
  notify: (message: string, tone?: ToastTone) => void;
}

const ToastContext = createContext<ToastApi | null>(null);

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const nextId = useRef(1);

  const dismiss = useCallback((id: number) => {
    setItems((current) => current.filter((item) => item.id !== id));
  }, []);

  const notify = useCallback(
    (message: string, tone: ToastTone = 'success') => {
      const id = nextId.current++;
      setItems((current) => [...current.slice(-3), { id, message, tone }]);
      window.setTimeout(() => dismiss(id), tone === 'error' ? 8000 : 4500);
    },
    [dismiss],
  );

  const api = useMemo(() => ({ notify }), [notify]);

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed bottom-4 end-4 z-[60] flex w-[min(380px,calc(100vw-2rem))] flex-col gap-2" aria-live="polite">
        {items.map((item) => {
          const Icon = item.tone === 'success' ? CheckCircle2 : item.tone === 'error' ? AlertTriangle : Info;
          return (
            <div
              key={item.id}
              className={cn(
                'pointer-events-auto flex items-start gap-2.5 rounded-xl border bg-slate-950 px-3.5 py-3 text-[13px] text-slate-200 shadow-floating animate-slide-in-up',
                item.tone === 'success' && 'border-emerald-500/35',
                item.tone === 'error' && 'border-rose-500/35',
                item.tone === 'info' && 'border-sky-500/35',
              )}
              role={item.tone === 'error' ? 'alert' : 'status'}
            >
              <Icon
                className={cn(
                  'mt-0.5 h-4 w-4 shrink-0',
                  item.tone === 'success' && 'text-emerald-400',
                  item.tone === 'error' && 'text-rose-400',
                  item.tone === 'info' && 'text-sky-400',
                )}
                aria-hidden
              />
              <span className="flex-1 leading-5">{item.message}</span>
              <button
                type="button"
                onClick={() => dismiss(item.id)}
                className="focus-ring rounded p-0.5 text-slate-500 hover:text-slate-200"
                aria-label="إغلاق الإشعار"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          );
        })}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const value = useContext(ToastContext);
  if (!value) {
    throw new Error('يجب استخدام useToast داخل ToastProvider');
  }
  return value;
}
