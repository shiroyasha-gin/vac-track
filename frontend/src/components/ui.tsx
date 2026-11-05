import {
  useEffect,
  useId,
  type ButtonHTMLAttributes,
  type InputHTMLAttributes,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react';
import { AlertTriangle, Loader2, X } from 'lucide-react';

export function cn(...values: Array<string | false | null | undefined>): string {
  return values.filter(Boolean).join(' ');
}

/* ------------------------------------------------------------------ buttons */

type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline';
type ButtonSize = 'sm' | 'md' | 'lg';

const buttonVariants: Record<ButtonVariant, string> = {
  primary:
    'bg-emerald-500 text-emerald-950 hover:bg-emerald-400 active:bg-emerald-300 shadow-sm shadow-emerald-950/40',
  secondary: 'bg-slate-800 text-slate-100 hover:bg-slate-700 active:bg-slate-600',
  ghost: 'bg-transparent text-slate-300 hover:bg-slate-800/70 hover:text-slate-100',
  danger: 'bg-rose-500/90 text-white hover:bg-rose-500 active:bg-rose-400',
  outline: 'border border-slate-700 bg-transparent text-slate-200 hover:border-slate-500 hover:bg-slate-800/60',
};

const buttonSizes: Record<ButtonSize, string> = {
  sm: 'h-8 px-2.5 text-xs gap-1.5',
  md: 'h-9 px-3.5 text-[13px] gap-2',
  lg: 'h-10 px-5 text-sm gap-2',
};

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
  icon?: ReactNode;
}

export function Button({
  variant = 'secondary',
  size = 'md',
  loading = false,
  icon,
  className,
  children,
  disabled,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      className={cn(
        'focus-ring inline-flex shrink-0 items-center justify-center rounded-lg font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50',
        buttonVariants[variant],
        buttonSizes[size],
        className,
      )}
      {...rest}
    >
      {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden /> : icon}
      {children}
    </button>
  );
}

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  variant?: ButtonVariant;
  size?: 'sm' | 'md';
}

export function IconButton({ label, variant = 'ghost', size = 'sm', className, children, ...rest }: IconButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={cn(
        'focus-ring inline-flex items-center justify-center rounded-lg transition-colors disabled:opacity-40',
        variant === 'ghost' ? 'text-slate-400 hover:bg-slate-800 hover:text-slate-100' : buttonVariants[variant],
        size === 'sm' ? 'h-7 w-7' : 'h-9 w-9',
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

/* ------------------------------------------------------------------- forms */

const controlClass =
  'w-full rounded-lg border border-slate-700/80 bg-slate-950/60 px-3 text-[13px] text-slate-100 placeholder:text-slate-500 transition-colors focus:border-emerald-400/70 focus:outline-none focus:ring-1 focus:ring-emerald-400/40 disabled:cursor-not-allowed disabled:opacity-50';

export function Input({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(controlClass, 'h-9', className)} {...rest} />;
}

export function Select({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={cn(controlClass, 'h-9 appearance-none bg-slate-950/80', className)} {...rest}>
      {children}
    </select>
  );
}

export function Textarea({ className, ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(controlClass, 'min-h-[76px] py-2 leading-5', className)} {...rest} />;
}

export interface FieldProps {
  label: string;
  htmlFor?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
  className?: string;
}

export function Field({ label, htmlFor, hint, error, required, children, className }: FieldProps) {
  return (
    <div className={cn('space-y-1.5', className)}>
      <label htmlFor={htmlFor} className="flex items-center gap-1 text-xs font-medium text-slate-300">
        {label}
        {required && <span className="text-emerald-400">*</span>}
      </label>
      {children}
      {error ? (
        <p className="text-xs text-rose-300">{error}</p>
      ) : hint ? (
        <p className="text-xs text-slate-500">{hint}</p>
      ) : null}
    </div>
  );
}

/* ------------------------------------------------------------------ layout */

export function Card({ className, children }: { className?: string; children: ReactNode }) {
  return <section className={cn('card', className)}>{children}</section>;
}

export function CardHeader({
  title,
  description,
  action,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn('flex flex-wrap items-start justify-between gap-3 border-b border-slate-800/70 px-4 py-3.5', className)}>
      <div>
        <h2 className="text-sm font-semibold text-slate-100">{title}</h2>
        {description ? <p className="mt-0.5 text-xs text-slate-500">{description}</p> : null}
      </div>
      {action}
    </div>
  );
}

export function PageHeader({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children?: ReactNode;
}) {
  return (
    <header className="mb-5 flex flex-wrap items-end justify-between gap-4">
      <div>
        <h1 className="text-xl font-semibold text-slate-100">{title}</h1>
        {description ? <p className="mt-1 max-w-3xl text-[13px] text-slate-400">{description}</p> : null}
      </div>
      {children ? <div className="flex flex-wrap items-center gap-2">{children}</div> : null}
    </header>
  );
}

export function StatCard({
  label,
  value,
  detail,
  tone = 'default',
  icon,
}: {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
  tone?: 'default' | 'emerald' | 'amber' | 'rose' | 'sky';
  icon?: ReactNode;
}) {
  const toneClass = {
    default: 'text-slate-100',
    emerald: 'text-emerald-300',
    amber: 'text-amber-300',
    rose: 'text-rose-300',
    sky: 'text-sky-300',
  }[tone];
  return (
    <div className="card flex items-start justify-between gap-3 px-4 py-3.5">
      <div>
        <p className="text-[11px] font-medium text-slate-500">{label}</p>
        <p className={cn('mt-1 text-2xl font-semibold tabular-nums', toneClass)}>{value}</p>
        {detail ? <p className="mt-0.5 text-xs text-slate-500">{detail}</p> : null}
      </div>
      {icon ? <div className="rounded-lg bg-slate-800/70 p-2 text-slate-400">{icon}</div> : null}
    </div>
  );
}

/* ------------------------------------------------------------------ badges */

type BadgeTone = 'slate' | 'emerald' | 'amber' | 'rose' | 'sky' | 'violet';

const badgeTones: Record<BadgeTone, string> = {
  slate: 'bg-slate-800 text-slate-300 ring-slate-700',
  emerald: 'bg-emerald-500/10 text-emerald-300 ring-emerald-500/30',
  amber: 'bg-amber-500/10 text-amber-300 ring-amber-500/30',
  rose: 'bg-rose-500/10 text-rose-300 ring-rose-500/30',
  sky: 'bg-sky-500/10 text-sky-300 ring-sky-500/30',
  violet: 'bg-violet-500/10 text-violet-300 ring-violet-500/30',
};

export function Badge({ tone = 'slate', children, className }: { tone?: BadgeTone; children: ReactNode; className?: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ring-1 ring-inset',
        badgeTones[tone],
        className,
      )}
    >
      {children}
    </span>
  );
}

export function StatusBadge({ state }: { state: string }) {
  switch (state) {
    case 'closed':
      return <Badge tone="violet">مغلق</Badge>;
    case 'open':
      return <Badge tone="emerald">مفتوح</Badge>;
    case 'not_initialized':
      return <Badge tone="slate">غير مهيأ</Badge>;
    default:
      return <Badge tone="slate">{state || 'غير معروف'}</Badge>;
  }
}

/* ------------------------------------------------------------------ alerts */

export function Alert({
  tone = 'info',
  title,
  children,
  className,
}: {
  tone?: 'info' | 'warning' | 'danger' | 'success';
  title?: ReactNode;
  children?: ReactNode;
  className?: string;
}) {
  const tones = {
    info: 'border-sky-500/30 bg-sky-500/5 text-sky-200',
    warning: 'border-amber-500/30 bg-amber-500/5 text-amber-100',
    danger: 'border-rose-500/30 bg-rose-500/5 text-rose-100',
    success: 'border-emerald-500/30 bg-emerald-500/5 text-emerald-100',
  }[tone];
  return (
    <div className={cn('rounded-xl border px-3.5 py-3 text-[13px]', tones, className)} role={tone === 'danger' ? 'alert' : undefined}>
      {title ? <p className="mb-0.5 flex items-center gap-1.5 text-xs font-semibold">{title}</p> : null}
      {children}
    </div>
  );
}

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 px-6 py-14 text-center">
      {icon ? <div className="mb-1 rounded-xl bg-slate-800/60 p-3 text-slate-400">{icon}</div> : null}
      <p className="text-sm font-medium text-slate-200">{title}</p>
      {description ? <p className="max-w-md text-xs text-slate-500">{description}</p> : null}
      {action ? <div className="mt-2">{action}</div> : null}
    </div>
  );
}

export function Spinner({ label = 'جارٍ التحميل' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-12 text-xs text-slate-500">
      <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
      {label}
    </div>
  );
}

export function InlineError({ message }: { message: string }) {
  if (!message) {
    return null;
  }
  return (
    <Alert tone="danger" className="flex items-start gap-2">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden />
      <span>{message}</span>
    </Alert>
  );
}

/* ------------------------------------------------------------------- modal */


export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'md',
  dismissable = true,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  dismissable?: boolean;
}) {
  const titleId = useId();
  const descriptionId = useId();
  useEffect(() => {
    if (!open) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && dismissable) {
        onClose();
      }
    };
    document.addEventListener('keydown', onKeyDown);
    return () => document.removeEventListener('keydown', onKeyDown);
  }, [open, dismissable, onClose]);

  if (!open) {
    return null;
  }
  const widths = {
    sm: 'max-w-md',
    md: 'max-w-xl',
    lg: 'max-w-3xl',
    xl: 'max-w-5xl',
  }[size];

  return (
      <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-500/30 p-4 backdrop-blur-sm sm:items-center">
        <button
          type="button"
          aria-label="إغلاق النافذة"
          className="absolute inset-0 cursor-default"
          onClick={dismissable ? onClose : undefined}
          tabIndex={-1}
        />
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          aria-describedby={description ? descriptionId : undefined}
          className={cn(
            'relative z-10 my-auto w-full animate-fade-in rounded-2xl border border-slate-800 bg-slate-900 shadow-2xl shadow-slate-500/20',
            widths,
          )}
        >
          <div className="flex items-start justify-between gap-4 border-b border-slate-800 px-5 py-4">
            <div>
              <h2 id={titleId} className="text-base font-semibold text-slate-100">
                {title}
              </h2>
              {description ? (
                <div id={descriptionId} className="mt-1 text-xs text-slate-400">
                  {description}
                </div>
              ) : null}
            </div>
            {dismissable ? (
              <IconButton label="إغلاق" onClick={onClose}>
                <X className="h-4 w-4" />
              </IconButton>
            ) : null}
          </div>
          <div className="max-h-[calc(100vh-14rem)] overflow-y-auto px-5 py-4">{children}</div>
          {footer ? <div className="flex flex-wrap items-center justify-end gap-2 border-t border-slate-800 px-5 py-3.5">{footer}</div> : null}
        </div>
      </div>
  );
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = 'تأكيد',
  tone = 'primary',
  busy = false,
  onConfirm,
  onClose,
  children,
}: {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  tone?: ButtonVariant;
  busy?: boolean;
  onConfirm: () => void;
  onClose: () => void;
  children?: ReactNode;
}) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={busy}>
            إلغاء
          </Button>
          <Button variant={tone} onClick={onConfirm} loading={busy}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      <div className="space-y-3 text-[13px] text-slate-300">
        <div>{message}</div>
        {children}
      </div>
    </Modal>
  );
}

/* ------------------------------------------------------------------ misc */

export function SectionTitle({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="mb-2 flex items-center justify-between">
      <h3 className="text-[11px] font-semibold text-slate-500">{children}</h3>
      {action}
    </div>
  );
}

export function KeyValue({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-slate-800/60 py-2 text-[13px] last:border-b-0">
      <span className="text-slate-500">{label}</span>
      <span className="max-w-[70%] break-words text-end text-slate-200">{children}</span>
    </div>
  );
}
