const locale = 'ar';
const numberingSystem = 'latn';

const integerFormat = new Intl.NumberFormat(locale, {
  maximumFractionDigits: 0,
  numberingSystem,
});

const percentFormat = new Intl.NumberFormat(locale, {
  maximumFractionDigits: 1,
  minimumFractionDigits: 1,
  numberingSystem,
});

const monthFormat = new Intl.DateTimeFormat(locale, {
  month: 'long',
  year: 'numeric',
  calendar: 'gregory',
  numberingSystem,
});

const shortMonthFormat = new Intl.DateTimeFormat(locale, {
  month: 'short',
  year: 'numeric',
  calendar: 'gregory',
  numberingSystem,
});

const dateFormat = new Intl.DateTimeFormat(locale, {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  calendar: 'gregory',
  numberingSystem,
});

const timestampFormat = new Intl.DateTimeFormat(locale, {
  dateStyle: 'medium',
  timeStyle: 'short',
  calendar: 'gregory',
  numberingSystem,
});

const timeFormat = new Intl.DateTimeFormat(locale, {
  hour: 'numeric',
  minute: '2-digit',
  hour12: true,
  calendar: 'gregory',
  numberingSystem,
});

export function formatInteger(value: number | null | undefined): string {
  if (value === null || value === undefined || Number.isNaN(value)) {
    return '—';
  }
  return integerFormat.format(value);
}

export function formatSigned(value: number): string {
  if (value > 0) {
    return `+${formatInteger(value)}`;
  }
  return formatInteger(value);
}

export function formatPercent(value: number): string {
  return `${percentFormat.format(value)}٪`;
}

function pad2(value: number): string {
  return String(value).padStart(2, '0');
}

export function todayISO(): string {
  const now = new Date();
  return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}-${pad2(now.getDate())}`;
}

export function currentMonth(): string {
  const now = new Date();
  return `${now.getFullYear()}-${pad2(now.getMonth() + 1)}`;
}

export function parseMonth(month: string): { year: number; month: number } {
  const [year, monthNumber] = month.split('-').map(Number);
  return { year, month: monthNumber };
}

export function monthLabel(month: string): string {
  if (!/^\d{4}-\d{2}$/.test(month)) {
    return month;
  }
  const { year, month: monthNumber } = parseMonth(month);
  return monthFormat.format(new Date(year, monthNumber - 1, 1));
}

export function monthLabelShort(month: string): string {
  if (!/^\d{4}-\d{2}$/.test(month)) {
    return month;
  }
  const { year, month: monthNumber } = parseMonth(month);
  return shortMonthFormat.format(new Date(year, monthNumber - 1, 1));
}

export function shiftMonth(month: string, delta: number): string {
  const { year, month: monthNumber } = parseMonth(month);
  const date = new Date(Date.UTC(year, monthNumber - 1 + delta, 1));
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, '0')}`;
}

export function monthDays(month: string): number {
  const { year, month: monthNumber } = parseMonth(month);
  return new Date(Date.UTC(year, monthNumber, 0)).getUTCDate();
}

export function dateInMonth(month: string, day: number): string {
  return `${month}-${String(day).padStart(2, '0')}`;
}

export function formatDate(value: string): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return value || '—';
  }
  const [year, monthNumber, day] = value.split('-').map(Number);
  return dateFormat.format(new Date(year, monthNumber - 1, day));
}

export function formatTime(value: string): string {
  if (!/^\d{2}:\d{2}$/.test(value)) {
    return value || '—';
  }
  const [hour, minute] = value.split(':').map(Number);
  return timeFormat.format(new Date(2000, 0, 1, hour, minute));
}

export function formatTimestamp(value: string): string {
  if (!value) {
    return '—';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return timestampFormat.format(parsed);
}

export function initials(name: string): string {
  return name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('');
}

export function pluralize(count: number, singular: string, plural = `${singular}s`): string {
  return count === 1 ? singular : plural;
}
