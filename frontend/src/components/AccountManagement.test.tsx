import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AccountManagement } from './AccountManagement';
import { ToastProvider } from './Toast';
import { api, type app } from '../lib/api';

vi.mock('../lib/api', async () => {
  const actual = await vi.importActual<typeof import('../lib/api')>('../lib/api');
  return {
    ...actual,
    api: {
      Users: vi.fn(),
      CreateUser: vi.fn(),
      UpdateUser: vi.fn(),
      SetUserActive: vi.fn(),
      ResetUserPassword: vi.fn(),
      AuthStatus: vi.fn(),
    },
  };
});

const users = [
  {
    id: 1,
    username: 'admin',
    displayName: 'مسؤول النظام',
    role: 'admin',
    active: true,
    createdAt: '',
    updatedAt: '',
  },
  {
    id: 2,
    username: 'nurse',
    displayName: 'ممرضة',
    role: 'staff',
    active: true,
    createdAt: '',
    updatedAt: '',
  },
] as unknown as app.UserDTO[];

const success = { ok: true, code: '', field: '', message: 'تم.', entityId: 0 } as unknown as app.ActionResult;

beforeEach(() => {
  vi.mocked(api.Users).mockResolvedValue(users);
  vi.mocked(api.AuthStatus).mockResolvedValue({
    authenticated: true,
    usingDefaultPassword: false,
    user: { id: 1, username: 'admin', displayName: 'مسؤول النظام', role: 'admin' },
  } as unknown as app.AuthStateDTO);
  vi.mocked(api.CreateUser).mockResolvedValue(success);
  vi.mocked(api.UpdateUser).mockResolvedValue(success);
  vi.mocked(api.SetUserActive).mockResolvedValue(success);
  vi.mocked(api.ResetUserPassword).mockResolvedValue(success);
});

function renderAccounts() {
  const onAuthUpdated = vi.fn();
  render(
    <ToastProvider>
      <AccountManagement currentUserId={1} onAuthUpdated={onAuthUpdated} />
    </ToastProvider>,
  );
  return onAuthUpdated;
}

describe('AccountManagement', () => {
  it('lists accounts and marks the signed-in user', async () => {
    renderAccounts();
    expect(await screen.findByText('ممرضة')).toBeInTheDocument();
    expect(screen.getByText('أنت')).toBeInTheDocument();
    expect(screen.getByText('مسؤول')).toBeInTheDocument();
    expect(screen.getByText('موظف')).toBeInTheDocument();
  });

  it('creates a local account', async () => {
    const user = userEvent.setup();
    renderAccounts();
    await screen.findByText('ممرضة');

    await user.click(screen.getByRole('button', { name: 'مستخدم جديد' }));
    await user.type(screen.getByLabelText('اسم المستخدم'), 'doctor');
    await user.type(screen.getByLabelText('الاسم المعروض'), 'طبيب');
    await user.selectOptions(screen.getByLabelText('الدور'), 'admin');
    await user.type(screen.getByLabelText('كلمة المرور'), 'clinic-secret-2026');
    await user.type(screen.getByLabelText('تأكيد كلمة المرور'), 'clinic-secret-2026');
    await user.click(screen.getByRole('button', { name: 'إضافة المستخدم' }));

    expect(api.CreateUser).toHaveBeenCalledWith({
      username: 'doctor',
      displayName: 'طبيب',
      password: 'clinic-secret-2026',
      role: 'admin',
    });
  });

  it('resets another account password', async () => {
    const user = userEvent.setup();
    renderAccounts();
    await screen.findByText('ممرضة');

    await user.click(screen.getAllByRole('button', { name: 'إعادة تعيين' })[1]);
    await user.type(screen.getByLabelText('كلمة المرور الجديدة'), 'another-secret-2026');
    await user.type(screen.getByLabelText('تأكيد كلمة المرور'), 'another-secret-2026');
    await user.click(screen.getByRole('button', { name: 'تعيين كلمة المرور' }));

    expect(api.ResetUserPassword).toHaveBeenCalledWith(2, 'another-secret-2026');
  });

  it('deactivates an account after confirmation', async () => {
    const user = userEvent.setup();
    renderAccounts();
    await screen.findByText('ممرضة');

    await user.click(screen.getByRole('button', { name: 'تعطيل' }));
    const confirm = screen.getAllByRole('button', { name: 'تعطيل' });
    await user.click(confirm[confirm.length - 1]);

    expect(api.SetUserActive).toHaveBeenCalledWith(2, false);
  });
});
