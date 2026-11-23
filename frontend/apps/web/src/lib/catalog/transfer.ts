// CSV/XLSX catalog transfer — raw fetch because the api client JSON-stringifies
// bodies, and export needs blob handling.
import type { ImportReport } from '@mauzo/types';
import { auth } from '#lib/auth.svelte.ts';

const IMPORT_URL = '/api/v1/t/catalog/products/import';
const EXPORT_URL = '/api/v1/t/catalog/products/export';
const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

/** Upload a .csv or .xlsx product file; returns the server's per-row report. */
export async function importProducts(file: File): Promise<ImportReport> {
  const isXlsx = /\.xlsx$/i.test(file.name) || file.type === XLSX_MIME;
  const res = await fetch(IMPORT_URL, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${auth.token}`,
      'Content-Type': isXlsx ? XLSX_MIME : 'text/csv',
    },
    body: file,
  });
  const payload = await res.json();
  if (!res.ok) throw new Error(payload?.error?.message ?? `HTTP ${res.status}`);
  return payload.data as ImportReport;
}

/** Download the catalog as a file. `format` is 'csv' (default) or 'xlsx'. */
export async function exportProducts(format: 'csv' | 'xlsx' = 'csv'): Promise<void> {
  const res = await fetch(`${EXPORT_URL}?format=${format}`, {
    headers: { Authorization: `Bearer ${auth.token}` },
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `products.${format}`;
  a.click();
  URL.revokeObjectURL(a.href);
}
