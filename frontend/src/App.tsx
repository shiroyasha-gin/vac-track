import { useCallback, useEffect, useState } from 'react';
import { HashRouter, Navigate, Route, Routes } from 'react-router-dom';
import { Loader2, Syringe } from 'lucide-react';
import { api, errorMessage, type app } from './lib/api';
import { MonthProvider } from './components/MonthContext';
import { Shell } from './components/Shell';
import { ToastProvider } from './components/Toast';
import { FatalPage } from './pages/FatalPage';
import { MonthlyPage } from './pages/MonthlyPage';
import { VaccinesPage } from './pages/VaccinesPage';
import { PatientsPage } from './pages/PatientsPage';
import { ReportsPage } from './pages/ReportsPage';
import { PublishPage } from './pages/PublishPage';
import { AuditPage } from './pages/AuditPage';
import { SettingsPage } from './pages/SettingsPage';

function StartupScreen({ message }: { message?: string }) {
  return (
    <div className="flex min-h-full items-center justify-center bg-slate-950">
      <div className="flex flex-col items-center gap-3 text-slate-400">
        <div className="flex h-11 w-11 animate-pulse items-center justify-center rounded-xl bg-emerald-500 text-emerald-950">
          <Syringe className="h-6 w-6" />
        </div>
        <div className="flex items-center gap-2 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {message || 'جارٍ فتح قاعدة البيانات المحلية…'}
        </div>
      </div>
    </div>
  );
}

export default function App() {
  const [bootstrap, setBootstrap] = useState<app.BootstrapInfo | null>(null);
  const [startupError, setStartupError] = useState('');
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    setStartupError('');
    api
      .Bootstrap()
      .then((info) => {
        if (active) {
          setBootstrap(info);
        }
      })
      .catch((cause: unknown) => {
        if (active) {
          setStartupError(errorMessage(cause));
        }
      });
    return () => {
      active = false;
    };
  }, [attempt]);

  const reload = useCallback(() => {
    setBootstrap(null);
    setAttempt((value) => value + 1);
  }, []);

  return (
    <ToastProvider>
      {startupError ? (
        <FatalPage
          bootstrap={{
            ready: false,
            fatalError: startupError,
            dataDir: '',
            databasePath: '',
            version: '',
            today: '',
            currentMonth: '',
            lowStockThreshold: 0,
            publisherEndpoint: '',
            publisherTokenConfigured: false,
            publisherTokenStorage: '',
          }}
          onReload={reload}
        />
      ) : !bootstrap ? (
        <StartupScreen />
      ) : !bootstrap.ready ? (
        <FatalPage bootstrap={bootstrap} onReload={reload} />
      ) : (
        <HashRouter>
          <MonthProvider month={bootstrap.currentMonth}>
            <Shell bootstrap={bootstrap}>
              <Routes>
                <Route path="/" element={<MonthlyPage />} />
                <Route path="/vaccines" element={<VaccinesPage />} />
                <Route path="/patients" element={<PatientsPage />} />
                <Route path="/reports" element={<ReportsPage />} />
                <Route path="/publish" element={<PublishPage />} />
                <Route path="/audit" element={<AuditPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </Shell>
          </MonthProvider>
        </HashRouter>
      )}
    </ToastProvider>
  );
}
