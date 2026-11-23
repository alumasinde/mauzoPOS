/** The dashboard's single time control; every widget reads it. */
export type Period = 'today' | '7d' | '30d' | '90d';

export const PERIODS: { key: Period; label: string; compare: string }[] = [
  { key: 'today', label: 'Today', compare: 'vs yesterday' },
  { key: '7d', label: 'Last 7 days', compare: 'vs previous 7 days' },
  { key: '30d', label: 'Last 30 days', compare: 'vs previous 30 days' },
  { key: '90d', label: 'Last 90 days', compare: 'vs previous 90 days' },
];

export function periodLabel(p: Period): string {
  return PERIODS.find((x) => x.key === p)?.label ?? p;
}
