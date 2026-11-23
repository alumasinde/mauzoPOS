import type { NavItem } from '@mauzo/types';
import { api } from './api';

/** The sidebar, built server-side from enabled modules × permissions. */
class Nav {
  items = $state<NavItem[]>([]);
  loaded = $state(false);

  async load(): Promise<void> {
    try {
      const res = await api.get<{ items: NavItem[] }>('/api/v1/t/navigation');
      this.items = res.items;
    } catch {
      this.items = [];
    } finally {
      this.loaded = true;
    }
  }

  clear(): void {
    this.items = [];
    this.loaded = false;
  }
}

export const nav = new Nav();
