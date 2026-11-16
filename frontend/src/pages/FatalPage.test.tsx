import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { FatalPage } from './FatalPage';
import { api, type app } from '../lib/api';

vi.mock('../lib/api', () => ({
  api: {
    RestoreBackup: vi.fn(),
    InitializeDatabase: vi.fn(),
  },
  errorMessage: (cause: unknown) => (typeof cause === 'string' ? cause : 'تعذر إكمال العملية.'),
}));

const initialize = vi.mocked(api.InitializeDatabase);

const bootstrap = {
  ready: false,
  fatalError: 'file is not a database',
  dataDir: '/tmp/vactrack',
  databasePath: '/tmp/vactrack/vaccineshit.db',
  version: '1.0.0',
} as unknown as app.BootstrapInfo;

beforeEach(() => {
  initialize.mockReset();
});

describe('FatalPage database recovery', () => {
  it('asks for confirmation before creating a new database', async () => {
    const user = userEvent.setup();
    render(<FatalPage bootstrap={bootstrap} onReload={() => undefined} />);

    await user.click(screen.getByRole('button', { name: /إنشاء قاعدة بيانات جديدة/ }));
    expect(screen.getByRole('dialog', { name: 'إنشاء قاعدة بيانات جديدة؟' })).toBeInTheDocument();
    expect(initialize).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'إنشاء قاعدة جديدة' }));
    expect(initialize).toHaveBeenCalledTimes(1);
  });

  it('keeps the old database when the confirmation is cancelled', async () => {
    const user = userEvent.setup();
    render(<FatalPage bootstrap={bootstrap} onReload={() => undefined} />);

    await user.click(screen.getByRole('button', { name: /إنشاء قاعدة بيانات جديدة/ }));
    await user.click(screen.getByRole('button', { name: 'إلغاء' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(initialize).not.toHaveBeenCalled();
  });

  it('reports the preserved location when initialization succeeds', async () => {
    initialize.mockResolvedValue({
      ok: true,
      code: '',
      message: 'تم إنشاء قاعدة بيانات جديدة فارغة.',
      preservedPath: '/tmp/vactrack/backups/pre-initialize-20261115T120000Z',
    } as unknown as app.InitializeDatabaseResultDTO);
    const user = userEvent.setup();
    render(<FatalPage bootstrap={bootstrap} onReload={() => undefined} />);

    await user.click(screen.getByRole('button', { name: /إنشاء قاعدة بيانات جديدة/ }));
    await user.click(screen.getByRole('button', { name: 'إنشاء قاعدة جديدة' }));

    expect(await screen.findByRole('status')).toHaveTextContent('تم إنشاء قاعدة بيانات جديدة فارغة.');
  });

  it('surfaces a failure without pretending the recovery worked', async () => {
    initialize.mockResolvedValue({
      ok: false,
      code: 'initialize_failed',
      message: 'تعذر إنشاء قاعدة بيانات جديدة.',
    } as unknown as app.InitializeDatabaseResultDTO);
    const user = userEvent.setup();
    render(<FatalPage bootstrap={bootstrap} onReload={() => undefined} />);

    await user.click(screen.getByRole('button', { name: /إنشاء قاعدة بيانات جديدة/ }));
    await user.click(screen.getByRole('button', { name: 'إنشاء قاعدة جديدة' }));

    expect(await screen.findByRole('status')).toHaveTextContent('تعذر إنشاء قاعدة بيانات جديدة.');
  });
});
