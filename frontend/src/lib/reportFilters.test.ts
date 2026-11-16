import { describe, expect, it } from 'vitest';
import type { app } from './api';
import {
  defaultReportFilters,
  filterForecastRows,
  filterVaccineStats,
  isDefaultFilters,
} from './reportFilters';

function stat(overrides: Partial<app.VaccineStatDTO> & { vaccineId: number; name: string }): app.VaccineStatDTO {
  return {
    code: '',
    unit: 'dose',
    archived: false,
    initialized: true,
    opening: 0,
    received: 0,
    usageTotal: 0,
    remaining: 0,
    previousUsage: 0,
    usageChange: 0,
    lowStock: false,
    ...overrides,
  } as app.VaccineStatDTO;
}

const rows = [
  stat({ vaccineId: 1, name: 'MMR', code: 'M-01', usageTotal: 30, remaining: 20, lowStock: true }),
  stat({ vaccineId: 2, name: 'Polio', code: 'P-02', usageTotal: 10, remaining: 90 }),
  stat({ vaccineId: 3, name: 'BCG', code: 'B-03', usageTotal: 0, remaining: 0, initialized: false }),
  stat({ vaccineId: 4, name: 'Old', code: '', usageTotal: 0, remaining: 5, archived: true }),
];

describe('filterVaccineStats', () => {
  it('returns usage-descending rows by default', () => {
    const result = filterVaccineStats(rows, defaultReportFilters);
    expect(result.map((row) => row.vaccineId)).toEqual([1, 2, 3, 4]);
  });

  it('matches name and code case-insensitively', () => {
    const byName = filterVaccineStats(rows, { ...defaultReportFilters, query: 'pol' });
    expect(byName.map((row) => row.vaccineId)).toEqual([2]);
    const byCode = filterVaccineStats(rows, { ...defaultReportFilters, query: 'b-03' });
    expect(byCode.map((row) => row.vaccineId)).toEqual([3]);
  });

  it('filters low stock, uninitialized and archived rows', () => {
    expect(
      filterVaccineStats(rows, { ...defaultReportFilters, status: 'low' }).map((row) => row.vaccineId),
    ).toEqual([1]);
    expect(
      filterVaccineStats(rows, { ...defaultReportFilters, status: 'uninitialized' }).map(
        (row) => row.vaccineId,
      ),
    ).toEqual([3]);
    expect(
      filterVaccineStats(rows, { ...defaultReportFilters, status: 'archived' }).map((row) => row.vaccineId),
    ).toEqual([4]);
  });

  it('hides rows without usage', () => {
    const result = filterVaccineStats(rows, { ...defaultReportFilters, hideZeroUsage: true });
    expect(result.map((row) => row.vaccineId)).toEqual([1, 2]);
  });

  it('sorts by remaining stock or name', () => {
    expect(
      filterVaccineStats(rows, { ...defaultReportFilters, sort: 'remaining_asc' }).map(
        (row) => row.vaccineId,
      ),
    ).toEqual([3, 4, 1, 2]);
    const byName = filterVaccineStats(rows, { ...defaultReportFilters, sort: 'name' });
    expect(byName.map((row) => row.name)).toEqual(['BCG', 'MMR', 'Old', 'Polio']);
  });
});

describe('filterForecastRows', () => {
  it('keeps only rows whose vaccine survived the filter', () => {
    const forecast = [
      { vaccineId: 1, name: 'MMR' },
      { vaccineId: 2, name: 'Polio' },
      { vaccineId: 4, name: 'Old' },
    ] as unknown as app.ForecastRowDTO[];
    expect(filterForecastRows(forecast, new Set([1, 4])).map((row) => row.vaccineId)).toEqual([1, 4]);
  });
});

describe('isDefaultFilters', () => {
  it('detects active filters', () => {
    expect(isDefaultFilters(defaultReportFilters)).toBe(true);
    expect(isDefaultFilters({ ...defaultReportFilters, query: 'mmr' })).toBe(false);
    expect(isDefaultFilters({ ...defaultReportFilters, hideZeroUsage: true })).toBe(false);
  });
});
