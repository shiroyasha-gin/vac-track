import type { app } from './api';

export type ReportStatusFilter = 'all' | 'low' | 'uninitialized' | 'archived';
export type ReportSort = 'usage_desc' | 'usage_asc' | 'remaining_asc' | 'name';

export interface ReportFilters {
  query: string;
  status: ReportStatusFilter;
  hideZeroUsage: boolean;
  sort: ReportSort;
}

export const defaultReportFilters: ReportFilters = {
  query: '',
  status: 'all',
  hideZeroUsage: false,
  sort: 'usage_desc',
};

export const reportStatusLabels: Record<ReportStatusFilter, string> = {
  all: 'كل اللقاحات',
  low: 'مخزون منخفض',
  uninitialized: 'غير مهيأة',
  archived: 'مؤرشفة',
};

export const reportSortLabels: Record<ReportSort, string> = {
  usage_desc: 'الأكثر استخداماً',
  usage_asc: 'الأقل استخداماً',
  remaining_asc: 'الأقل مخزوناً',
  name: 'الاسم أبجدياً',
};

function matchesQuery(row: app.VaccineStatDTO, query: string): boolean {
  const needle = query.trim().toLocaleLowerCase();
  if (!needle) {
    return true;
  }
  return (
    row.name.toLocaleLowerCase().includes(needle) ||
    row.code.toLocaleLowerCase().includes(needle)
  );
}

function matchesStatus(row: app.VaccineStatDTO, status: ReportStatusFilter): boolean {
  switch (status) {
    case 'low':
      return row.lowStock;
    case 'uninitialized':
      return !row.initialized;
    case 'archived':
      return row.archived;
    default:
      return true;
  }
}

/** filterVaccineStats applies the report filter bar to one month of stats. */
export function filterVaccineStats(
  rows: app.VaccineStatDTO[],
  filters: ReportFilters,
): app.VaccineStatDTO[] {
  const filtered = rows.filter(
    (row) =>
      matchesQuery(row, filters.query) &&
      matchesStatus(row, filters.status) &&
      (!filters.hideZeroUsage || row.usageTotal > 0),
  );
  const sorted = [...filtered];
  switch (filters.sort) {
    case 'usage_asc':
      sorted.sort((a, b) => a.usageTotal - b.usageTotal || a.name.localeCompare(b.name, 'ar'));
      break;
    case 'remaining_asc':
      sorted.sort((a, b) => a.remaining - b.remaining || a.name.localeCompare(b.name, 'ar'));
      break;
    case 'name':
      sorted.sort((a, b) => a.name.localeCompare(b.name, 'ar'));
      break;
    default:
      sorted.sort((a, b) => b.usageTotal - a.usageTotal || a.name.localeCompare(b.name, 'ar'));
  }
  return sorted;
}

/** filterForecastRows keeps the forecast aligned with the filtered vaccine table. */
export function filterForecastRows(
  rows: app.ForecastRowDTO[],
  allowedIds: Set<number>,
): app.ForecastRowDTO[] {
  return rows.filter((row) => allowedIds.has(row.vaccineId));
}

export function isDefaultFilters(filters: ReportFilters): boolean {
  return (
    !filters.query.trim() &&
    filters.status === 'all' &&
    !filters.hideZeroUsage &&
    filters.sort === 'usage_desc'
  );
}
