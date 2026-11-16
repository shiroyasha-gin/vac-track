import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SettingsPage } from './SettingsPage';
import { ToastProvider } from '../components/Toast';
import { api, type app } from '../lib/api';

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api');
  return {
    ...actual,
    api: {
      Settings: vi.fn(),
      ChangePassword: vi.fn(),
      CreateBackup: vi.fn(),
      RestoreBackup: vi.fn(),
      SetLowStockThreshold: vi.fn(),
    },
  };
});

const changePassword = vi.mocked(api.ChangePassword);

const settings = {
  lowStockThreshold: 10,
  publisherEndpoint: '',
  publisherTokenConfigured: false,
  publisherTokenStorage: '',
  dataDir: '/tmp/data',
  databasePath: '/tmp/data/app.db',
  backupDir: '/tmp/data/backups',
};

beforeEach(() => {
  changePassword.mockReset();
  vi.mocked(api.Settings).mockResolvedValue(settings as unknown as app.SettingsDTO);
});

function renderPage(onAuthUpdated = vi.fn(), role = 'staff') {
  render(
    <ToastProvider>
      <SettingsPage role={role} currentUserId={1} onAuthUpdated={onAuthUpdated} />
    </ToastProvider>,
  );
  return onAuthUpdated;
}

describe('SettingsPage password change', () => {
  it('hides admin-only sections from staff accounts', async () => {
    renderPage(vi.fn(), 'staff');
    await screen.findByText('أمان الحساب');
    expect(screen.queryByText('إدارة المستخدمين')).not.toBeInTheDocument();
    expect(screen.queryByText('النسخ الاحتياطي والاحتفاظ')).not.toBeInTheDocument();
  });

  it('rejects mismatched confirmation without calling the backend', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByText('أمان الحساب');

    await user.type(screen.getByLabelText('كلمة المرور الحالية'), 'admin123');
    await user.type(screen.getByLabelText('كلمة المرور الجديدة'), 'clinic-secret-2026');
    await user.type(screen.getByLabelText('تأكيد كلمة المرور'), 'different-secret');
    await user.click(screen.getByRole('button', { name: 'تغيير كلمة المرور' }));

    expect(await screen.findByText('كلمتا المرور الجديدتان غير متطابقتين.')).toBeInTheDocument();
    expect(changePassword).not.toHaveBeenCalled();
  });

  it('updates the session after a successful change', async () => {
    const user = userEvent.setup();
    const onAuthUpdated = renderPage();
    changePassword.mockResolvedValue({
      authenticated: true,
      usingDefaultPassword: false,
      user: { username: 'admin', displayName: 'مسؤول النظام', role: 'admin' },
    } as unknown as app.AuthStateDTO);
    await screen.findByText('أمان الحساب');

    await user.type(screen.getByLabelText('كلمة المرور الحالية'), 'admin123');
    await user.type(screen.getByLabelText('كلمة المرور الجديدة'), 'clinic-secret-2026');
    await user.type(screen.getByLabelText('تأكيد كلمة المرور'), 'clinic-secret-2026');
    await user.click(screen.getByRole('button', { name: 'تغيير كلمة المرور' }));

    expect(changePassword).toHaveBeenCalledWith('admin123', 'clinic-secret-2026');
    expect(onAuthUpdated).toHaveBeenCalledWith(
      expect.objectContaining({ authenticated: true, usingDefaultPassword: false }),
    );
    expect(screen.getByLabelText('كلمة المرور الحالية')).toHaveValue('');
  });
});
