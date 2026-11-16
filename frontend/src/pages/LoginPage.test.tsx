import { describe, expect, it, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LoginPage } from './LoginPage';
import { api, type app } from '../lib/api';

vi.mock('../lib/api', () => ({
  api: { Login: vi.fn() },
  errorMessage: (cause: unknown) => (typeof cause === 'string' ? cause : 'تعذر إكمال العملية.'),
}));

const login = vi.mocked(api.Login);

beforeEach(() => {
  login.mockReset();
});

describe('LoginPage', () => {
  it('requires both fields before calling the backend', async () => {
    const user = userEvent.setup();
    render(<LoginPage version="1.0.0" onSuccess={() => undefined} />);

    await user.click(screen.getByRole('button', { name: /دخول/ }));
    expect(screen.getByRole('alert')).toHaveTextContent('أدخل اسم المستخدم وكلمة المرور.');
    expect(login).not.toHaveBeenCalled();
  });

  it('submits credentials and reports the authenticated session', async () => {
    const onSuccess = vi.fn();
    login.mockResolvedValue({
      authenticated: true,
      user: { username: 'admin', displayName: 'مسؤول النظام', role: 'admin' },
    } as unknown as app.AuthStateDTO);
    const user = userEvent.setup();
    render(<LoginPage version="1.0.0" onSuccess={onSuccess} />);

    await user.type(screen.getByLabelText('اسم المستخدم'), 'admin');
    await user.type(screen.getByLabelText('كلمة المرور'), 'admin123');
    await user.click(screen.getByRole('button', { name: /دخول/ }));

    expect(login).toHaveBeenCalledWith('admin', 'admin123');
    expect(onSuccess).toHaveBeenCalledWith(
      expect.objectContaining({ authenticated: true, user: expect.objectContaining({ username: 'admin' }) }),
    );
  });

  it('shows the backend message when credentials are rejected', async () => {
    login.mockRejectedValue('اسم المستخدم أو كلمة المرور غير صحيحة.');
    const user = userEvent.setup();
    render(<LoginPage version="1.0.0" onSuccess={() => undefined} />);

    await user.type(screen.getByLabelText('اسم المستخدم'), 'admin');
    await user.type(screen.getByLabelText('كلمة المرور'), 'wrong');
    await user.click(screen.getByRole('button', { name: /دخول/ }));

    expect(await screen.findByRole('alert')).toHaveTextContent('اسم المستخدم أو كلمة المرور غير صحيحة.');
  });

  it('hides the password by default and toggles its visibility', async () => {
    const user = userEvent.setup();
    render(<LoginPage version="1.0.0" onSuccess={() => undefined} />);

    const password = screen.getByLabelText('كلمة المرور');
    expect(password).toHaveAttribute('type', 'password');
    await user.click(screen.getByRole('button', { name: 'إظهار كلمة المرور' }));
    expect(password).toHaveAttribute('type', 'text');
  });
});
