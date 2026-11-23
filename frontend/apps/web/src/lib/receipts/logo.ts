import { auth } from '#lib/auth.svelte.ts';

/**
 * Loads the tenant's receipt logo (already 1-bit, as printed) as an object
 * URL. <img src> cannot send the bearer token, so it is fetched here.
 * Returns null when there is no logo.
 */
export async function loadLogoURL(): Promise<string | null> {
  const base = import.meta.env.VITE_API_URL ?? '';
  const res = await fetch(`${base}/api/v1/t/receipts/logo`, {
    headers: auth.token ? { Authorization: `Bearer ${auth.token}` } : {},
  }).catch(() => null);
  if (!res || !res.ok) return null;
  return URL.createObjectURL(await res.blob());
}
