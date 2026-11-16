import { afterEach, describe, expect, it, vi } from 'vitest';
import { printReportNode } from './print';

afterEach(() => {
  document.body.className = '';
  document.body.innerHTML = '';
  document.head.querySelectorAll('style[data-test-page]').forEach((element) => element.remove());
  document.title = '';
  vi.restoreAllMocks();
});

describe('printReportNode', () => {
  it('clones the report, hides no-print content and cleans up after printing', () => {
    const source = document.createElement('div');
    source.innerHTML = '<p>تقرير</p><button class="no-print">طباعة</button>';
    document.body.appendChild(source);

    const print = vi.spyOn(window, 'print').mockImplementation(() => {
      const sheet = document.querySelector('.print-sheet');
      expect(sheet).not.toBeNull();
      expect(sheet?.textContent).toContain('تقرير');
      expect(sheet?.querySelector('.no-print')).toBeNull();
      expect(document.body.classList.contains('printing-sheet')).toBe(true);
      expect(document.title).toBe('VacTrack-test-report');
    });

    printReportNode(source, { title: 'VacTrack-test-report', orientation: 'portrait' });

    expect(print).toHaveBeenCalledOnce();
    expect(document.querySelector('.print-sheet')).toBeNull();
    expect(document.body.classList.contains('printing-sheet')).toBe(false);
    expect(document.title).toBe('');
  });

  it('does nothing without a node', () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => {});
    printReportNode(null, { title: 'VacTrack-test-report' });
    expect(print).not.toHaveBeenCalled();
  });
});
