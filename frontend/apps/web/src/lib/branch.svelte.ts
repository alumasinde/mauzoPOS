import type { Branch } from '@mauzo/types';
import { api } from './api';

const KEY = 'mauzo_branch';

/**
 * The branch the back office is working in, picked in the top bar and sent
 * as X-Branch-Id on every API call. null = all branches (only offered to
 * users with access to every branch).
 */
class BranchStore {
  list = $state<Branch[]>([]);
  id = $state<string | null>(null);
  allowAll = $state(false);
  loaded = $state(false);

  current = $derived(this.list.find((b) => b.id === this.id) ?? null);
  label = $derived(this.current?.name ?? (this.allowAll ? 'All branches' : '—'));

  /** Suppresses the header while the list loads (see header()). */
  private loading = false;

  /** The X-Branch-Id header for API calls. */
  header(): Record<string, string> {
    return this.id && !this.loading ? { 'X-Branch-Id': this.id } : {};
  }

  async load(allBranches: boolean): Promise<void> {
    // Fetch without the header: a branch the user lost access to would
    // otherwise make this very call fail.
    this.loading = true;
    try {
      this.list = await api.get<Branch[]>('/api/v1/t/branches/accessible');
    } catch {
      this.list = [];
    } finally {
      this.loading = false;
    }
    this.allowAll = allBranches && this.list.length > 1;
    const saved = read();
    const next =
      saved && this.list.some((b) => b.id === saved)
        ? saved
        : this.allowAll
          ? null
          : ((this.list.find((b) => b.is_main) ?? this.list[0])?.id ?? null);
    if (next !== this.id) this.id = next;
    this.loaded = true;
  }

  select(id: string | null): void {
    if (id === null && !this.allowAll) return;
    this.id = id;
    try {
      if (id) localStorage.setItem(KEY, id);
      else localStorage.removeItem(KEY);
    } catch {
      /* storage unavailable: selection lasts for this tab */
    }
  }

  clear(): void {
    this.list = [];
    this.id = null;
    this.loaded = false;
  }
}

function read(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

export const branch = new BranchStore();
