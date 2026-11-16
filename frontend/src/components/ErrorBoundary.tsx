import { Component, type ErrorInfo, type ReactNode } from 'react';
import { AlertTriangle, RotateCcw } from 'lucide-react';

interface Props {
  children: ReactNode;
}

interface State {
  failed: boolean;
}

/** Catches unexpected render errors so the desktop window never goes blank. */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { failed: false };

  static getDerivedStateFromError(): State {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Unhandled interface error', error, info.componentStack);
  }

  render() {
    if (!this.state.failed) {
      return this.props.children;
    }

    return (
      <div className="flex min-h-full items-center justify-center bg-slate-900 px-6 py-10" role="alert">
        <div className="w-full max-w-lg rounded-xl border border-slate-700/70 bg-slate-950 p-6 text-center shadow-panel">
          <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-rose-500/10 text-rose-300 ring-1 ring-inset ring-rose-500/25">
            <AlertTriangle className="h-6 w-6" />
          </div>
          <h1 className="mt-4 text-lg font-semibold text-slate-100">حدث خطأ غير متوقع في الواجهة</h1>
          <p className="mt-2 text-[13px] leading-6 text-slate-500">
            لم تُفقد بياناتك. أعد تحميل التطبيق للمتابعة، وإذا تكرر الخطأ فراجع ملف application.log داخل مجلد البيانات.
          </p>
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="focus-ring mt-5 inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-emerald-500 px-4 text-[13px] font-medium text-white shadow-sm transition-colors hover:bg-emerald-600"
          >
            <RotateCcw className="h-4 w-4" />
            إعادة تحميل التطبيق
          </button>
        </div>
      </div>
    );
  }
}
