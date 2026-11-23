// Money display helpers. Values arrive as decimal strings; format only —
// arithmetic stays server/binding side.

export function formatMoney(value: string, currency = ''): string {
  const n = Number(value);
  if (Number.isNaN(n)) return value;
  const s = n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  return currency ? s : s;
}
