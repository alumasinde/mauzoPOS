/** Formatting shared by the A4 invoice components. */

export const METHODS: Record<string, string> = {
  cash: 'Cash',
  card: 'Card',
  mpesa: 'M-Pesa',
  bank: 'Bank transfer',
  credit: 'On account',
};

export const money = (s?: string | null): string =>
  s == null || s === ''
    ? ''
    : Number(s).toLocaleString('en-KE', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export const qty = (s: string): string => Number(s).toLocaleString('en-KE', { maximumFractionDigits: 3 });

export const when = (s: string): string =>
  new Date(s).toLocaleString('en-KE', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Africa/Nairobi',
  });
