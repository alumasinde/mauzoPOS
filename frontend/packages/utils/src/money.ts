// Display-only money formatting. Arithmetic on money happens in Go (pkg/money);
// the UI never parses amounts into floats to calculate.

const AMOUNT_RE = /^(-)?(\d+)(?:\.(\d+))?$/;

/**
 * Formats a decimal string ("1234567.5") as "KSh 1,234,567.50".
 * Rounds half-up to 2 decimals using string arithmetic (no float conversion).
 */
export function formatMoney(amount: string, symbol = 'KSh'): string {
  const m = AMOUNT_RE.exec(amount.trim());
  if (!m) return amount;
  const [, neg, intRaw = '0', fracRaw = ''] = m;

  let cents = BigInt(intRaw + (fracRaw + '00').slice(0, 2));
  if ((fracRaw[2] ?? '0') >= '5') cents += 1n;

  const int = (cents / 100n).toString();
  const frac = (cents % 100n).toString().padStart(2, '0');
  const grouped = int.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  const sign = neg && cents !== 0n ? '-' : '';
  return `${sign}${symbol ? symbol + ' ' : ''}${grouped}.${frac}`;
}
