import { resolve } from '$app/paths';
import type { NavItem } from '@mauzo/types';

/**
 * Pages this app ships. The server may send nav entries for screens that are
 * not built yet (so the sidebar keeps its final shape); those render as
 * "coming soon" instead of links. A module plugs in by adding its page and
 * listing the path here.
 */
export const ROUTES = [
  '/',
  '/onboarding',
  '/modules',
  '/settings',
  '/audit',
  '/employees',
  '/roles',
  '/catalog',
  '/catalog/taxonomy',
  '/catalog/labels',
  '/inventory',
  '/inventory/documents',
  '/pricing',
  '/pricing/promotions',
  '/pricing/history',
] as const;

export type AppRoute = (typeof ROUTES)[number];

/** Splits "/inventory?tab=movements" into its route and query. */
export function splitPath(path: string): { base: string; query: URLSearchParams } {
  const [base = '/', q = ''] = path.split('?');
  return { base, query: new URLSearchParams(q) };
}

export function isBuilt(path: string): boolean {
  return (ROUTES as readonly string[]).includes(splitPath(path).base);
}

/**
 * The key of the nav item for the current URL. An item whose query also
 * matches wins over a bare route ("/inventory?tab=movements" over "/inventory").
 */
export function activeKey(
  items: NavItem[],
  url: { pathname: string; searchParams: { get(key: string): string | null } },
): string | null {
  let fallback: string | null = null;
  for (const item of items) {
    const { base, query } = splitPath(item.path);
    if (base !== url.pathname) continue;
    const keys = [...query.keys()];
    if (keys.length && keys.every((k) => url.searchParams.get(k) === query.get(k))) return item.key;
    if (!keys.length && fallback === null) fallback = item.key;
  }
  return fallback;
}

/** A link for a built nav path, base-path aware, keeping its query. */
export function hrefFor(path: string): string {
  const { base, query } = splitPath(path);
  const q = query.toString();
  return resolve(base as AppRoute) + (q ? `?${q}` : '');
}
