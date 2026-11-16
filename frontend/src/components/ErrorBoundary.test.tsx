import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

function BrokenPage(): never {
  throw new Error('render failed');
}

describe('ErrorBoundary', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows a recoverable fallback instead of a blank window', () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    // React rethrows caught render errors in development; keep jsdom from
    // printing the expected error as an uncaught exception.
    const preventDefault = (event: ErrorEvent) => event.preventDefault();
    window.addEventListener('error', preventDefault);

    try {
      render(
        <ErrorBoundary>
          <BrokenPage />
        </ErrorBoundary>,
      );
    } finally {
      window.removeEventListener('error', preventDefault);
    }

    expect(screen.getByRole('alert')).toHaveTextContent('حدث خطأ غير متوقع في الواجهة');
    expect(screen.getByRole('button', { name: 'إعادة تحميل التطبيق' })).toBeInTheDocument();
    expect(screen.queryByText('render failed')).not.toBeInTheDocument();
  });
});
