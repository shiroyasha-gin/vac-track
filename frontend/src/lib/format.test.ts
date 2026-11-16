import { describe, expect, it, vi } from 'vitest';
import {
  currentMonth,
  dateInMonth,
  formatInteger,
  formatPercent,
  formatSigned,
  formatTime,
  initials,
  monthDays,
  monthLabel,
  parseMonth,
  pluralize,
  shiftMonth,
  todayISO,
} from './format';

describe('month helpers', () => {
  it('counts days correctly including leap years', () => {
    expect(monthDays('2024-02')).toBe(29);
    expect(monthDays('2023-02')).toBe(28);
    expect(monthDays('2026-11')).toBe(30);
    expect(monthDays('2026-12')).toBe(31);
  });

  it('shifts months across year boundaries', () => {
    expect(shiftMonth('2026-01', -1)).toBe('2025-12');
    expect(shiftMonth('2026-12', 1)).toBe('2027-01');
    expect(shiftMonth('2026-06', 0)).toBe('2026-06');
  });

  it('parses and labels months', () => {
    expect(parseMonth('2026-11')).toEqual({ year: 2026, month: 11 });
    expect(monthLabel('2026-11')).toContain('2026');
    expect(monthLabel('not-a-month')).toBe('not-a-month');
  });

  it('builds ISO dates within a month', () => {
    expect(dateInMonth('2026-03', 1)).toBe('2026-03-01');
    expect(dateInMonth('2026-03', 31)).toBe('2026-03-31');
  });

  it('produces current ISO values in the expected shape', () => {
    expect(todayISO()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(currentMonth()).toMatch(/^\d{4}-\d{2}$/);
  });
});

describe('local calendar dates', () => {
  it('keeps ISO dates on their own day west of UTC', async () => {
    const proc = (globalThis as { process?: { env: Record<string, string | undefined> } }).process;
    if (!proc) {
      throw new Error('vitest must run in a Node environment');
    }
    const originalTZ = proc.env.TZ;
    proc.env.TZ = 'America/New_York';
    vi.resetModules();
    try {
      const local = await import('./format');

      // A UTC-midnight parse would show the previous day (28 February) here.
      const label = local.formatDate('2026-03-01');
      expect(label).toContain('1');
      expect(label).toContain('مارس');
      expect(local.monthLabel('2026-03')).toContain('مارس');
    } finally {
      if (originalTZ === undefined) {
        delete proc.env.TZ;
      } else {
        proc.env.TZ = originalTZ;
      }
      vi.resetModules();
    }
  });
});

describe('formatters', () => {
  it('formats whole numbers and missing values', () => {
    expect(formatInteger(0)).toBe('0');
    expect(formatInteger(42)).toBe('42');
    expect(formatInteger(null)).toBe('—');
    expect(formatInteger(undefined)).toBe('—');
    expect(formatInteger(Number.NaN)).toBe('—');
  });

  it('formats signed adjustments', () => {
    expect(formatSigned(5)).toBe('+5');
    expect(formatSigned(-3)).toBe('\u200e-3');
    expect(formatSigned(0)).toBe('0');
  });

  it('formats appointment times and missing values', () => {
    expect(formatTime('')).toBe('—');
    expect(formatTime('09:30')).toMatch(/9:30/);
  });

  it('formats percentages and plural labels', () => {
    expect(formatPercent(12.345)).toBe('12.3٪');
    expect(pluralize(1, 'dose')).toBe('dose');
    expect(pluralize(2, 'dose')).toBe('doses');
  });

  it('builds initials from a name', () => {
    expect(initials('Ada Lovelace')).toBe('AL');
    expect(initials('  Grace   Brewster Hopper ')).toBe('GB');
    expect(initials('')).toBe('');
  });
});
