import type { NavItem, NavResponse, NavSection } from '@mauzo/types';
import { api } from './api';

export interface NavGroup extends NavSection {
  items: NavItem[];
}

/** The sidebar, built server-side from enabled modules × permissions. */
class Nav {
  items = $state<NavItem[]>([]);
  sections = $state<NavSection[]>([]);
  loaded = $state(false);

  /** Items grouped under their headings, in server order; empty sections dropped. */
  groups = $derived.by<NavGroup[]>(() =>
    this.sections
      .map((s) => ({ ...s, items: this.items.filter((i) => i.section === s.key) }))
      .filter((g) => g.items.length > 0),
  );

  async load(): Promise<void> {
    try {
      const res = await api.get<NavResponse>('/api/v1/t/navigation');
      this.items = res.items;
      this.sections = res.sections ?? [];
    } catch {
      this.items = [];
      this.sections = [];
    } finally {
      this.loaded = true;
    }
  }

  clear(): void {
    this.items = [];
    this.sections = [];
    this.loaded = false;
  }
}

export const nav = new Nav();
