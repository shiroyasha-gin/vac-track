import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfirmDialog } from './ui';
import { MonthPicker, MonthProvider } from './MonthContext';

describe('ConfirmDialog', () => {
  it('renders a dialog and invokes confirm or cancel handlers', async () => {
    const onConfirm = vi.fn();
    const onClose = vi.fn();
    const user = userEvent.setup();

    render(
      <ConfirmDialog
        open
        title="Close month?"
        message="Routine edits become read-only."
        confirmLabel="Close month"
        onConfirm={onConfirm}
        onClose={onClose}
      />,
    );

    expect(screen.getByRole('dialog', { name: 'Close month?' })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Close month' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'إلغاء' }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not render content when closed', () => {
    render(
      <ConfirmDialog
        open={false}
        title="Hidden dialog"
        message="Nothing to see"
        onConfirm={() => undefined}
        onClose={() => undefined}
      />,
    );
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

describe('MonthPicker', () => {
  it('advances and rewinds the selected month', () => {
    render(
      <MonthProvider month="2026-01">
        <MonthPicker />
      </MonthProvider>,
    );

    const input = screen.getByLabelText('شهر المخزون');
    expect(input).toHaveValue('2026-01');

    fireEvent.click(screen.getByRole('button', { name: 'الشهر السابق' }));
    expect(input).toHaveValue('2025-12');

    fireEvent.click(screen.getByRole('button', { name: 'الشهر التالي' }));
    expect(input).toHaveValue('2026-01');
  });

  it('accepts a directly selected month', () => {
    render(
      <MonthProvider month="2026-01">
        <MonthPicker />
      </MonthProvider>,
    );

    const input = screen.getByLabelText('شهر المخزون');
    fireEvent.change(input, { target: { value: '2026-08' } });
    expect(input).toHaveValue('2026-08');
  });
});
