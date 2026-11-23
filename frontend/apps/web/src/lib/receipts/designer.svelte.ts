import { ApiError } from '@mauzo/api-client';
import type {
  ReceiptMeta,
  ReceiptOutput,
  ReceiptPreview,
  ReceiptTemplate,
  ReceiptTemplateBody,
  ReceiptTemplateVersion,
  Sale,
} from '@mauzo/types';
import { toast } from '@mauzo/ui';
import { api } from '#lib/api.ts';
import { branch } from '#lib/branch.svelte.ts';
import { loadLogoURL } from './logo.ts';

const BASE = '/api/v1/t/receipts';

/** State and actions of the receipt designer page. */
export class Designer {
  meta = $state<ReceiptMeta | null>(null);
  templates = $state<ReceiptTemplate[]>([]);
  loading = $state(true);

  // The layout being edited.
  currentId = $state<string | null>(null);
  name = $state('');
  version = $state(0);
  body = $state<ReceiptTemplateBody | null>(null);
  saved = $state(''); // name+body as last saved
  selected = $state(0);
  saving = $state(false);
  dirty = $derived(this.body !== null && JSON.stringify({ name: this.name, body: this.body }) !== this.saved);

  // Preview.
  preview = $state<ReceiptOutput | null>(null);
  problems = $state<string[]>([]);
  source = $state(''); // sale public id, "" = sample
  recent = $state<Sale[]>([]);
  logoUrl = $state<string | null>(null);
  private timer: ReturnType<typeof setTimeout> | undefined;

  versions = $state<ReceiptTemplateVersion[]>([]);

  async load(): Promise<void> {
    try {
      [this.meta, this.templates] = await Promise.all([
        api.get<ReceiptMeta>(`${BASE}/meta`),
        api.get<ReceiptTemplate[]>(`${BASE}/templates`),
      ]);
      if (this.templates[0]) this.open(this.templates[0], true);
      api
        .list<Sale>('/api/v1/t/sales', { per_page: 10 })
        .then((r) => (this.recent = r.data))
        .catch(() => (this.recent = []));
      await this.refreshLogo();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load the designer');
    } finally {
      this.loading = false;
    }
  }

  dispose(): void {
    clearTimeout(this.timer);
    if (this.logoUrl) URL.revokeObjectURL(this.logoUrl);
  }

  async refreshLogo(): Promise<void> {
    if (this.logoUrl) URL.revokeObjectURL(this.logoUrl);
    this.logoUrl = await loadLogoURL();
    this.schedulePreview();
  }

  /** Asks before throwing away unsaved edits. */
  canLeave(): boolean {
    return !this.dirty || confirm('Discard your unsaved changes to this layout?');
  }

  open(t: ReceiptTemplate, force = false): void {
    if (!force && t.id !== this.currentId && !this.canLeave()) return;
    this.currentId = t.id;
    this.name = t.name;
    this.version = t.version;
    this.body = structuredClone(t.body);
    this.saved = JSON.stringify({ name: this.name, body: this.body });
    this.selected = 0;
    this.schedulePreview(0);
  }

  created(t: ReceiptTemplate): void {
    this.templates = [t, ...this.templates];
    this.open(t, true);
  }

  schedulePreview(delay = 300): void {
    clearTimeout(this.timer);
    this.timer = setTimeout(() => void this.runPreview(), delay);
  }

  setSource(id: string): void {
    this.source = id;
    this.schedulePreview(0);
  }

  private async runPreview(): Promise<void> {
    if (!this.body) return;
    try {
      const res = await api.post<ReceiptPreview>(`${BASE}/preview`, {
        body: $state.snapshot(this.body),
        branch_id: branch.id ?? '',
        sale_id: this.source,
      });
      this.preview = res.output;
      this.problems = [];
    } catch (e) {
      const details = e instanceof ApiError ? (e.details?.body as string[] | undefined) : undefined;
      this.problems = details ?? [e instanceof ApiError ? e.message : 'Preview failed'];
    }
  }

  async save(): Promise<void> {
    if (!this.body || !this.currentId) return;
    this.saving = true;
    try {
      const t = await api.put<ReceiptTemplate>(`${BASE}/templates/${this.currentId}`, {
        name: this.name.trim(),
        body: $state.snapshot(this.body),
        version: this.version,
      });
      this.templates = this.templates.map((x) => (x.id === t.id ? t : x));
      this.version = t.version;
      this.saved = JSON.stringify({ name: this.name, body: this.body });
      toast.success(`Saved as version ${t.version}. Tills update on their next sync.`);
    } catch (e) {
      if (e instanceof ApiError && e.code === 'validation_failed') {
        this.problems = (e.details?.body as string[] | undefined) ?? [e.message];
        toast.error('Fix the problems listed above the preview');
      } else {
        toast.error(e instanceof ApiError ? e.message : 'Save failed');
      }
    } finally {
      this.saving = false;
    }
  }

  async archive(): Promise<void> {
    const id = this.currentId;
    if (!id || !confirm(`Delete the layout "${this.name}"? Receipts already printed keep their copy.`))
      return;
    try {
      await api.del(`${BASE}/templates/${id}`);
      this.templates = this.templates.filter((t) => t.id !== id);
      this.currentId = null;
      this.body = null;
      this.preview = null;
      if (this.templates[0]) this.open(this.templates[0], true);
      toast.success('Layout deleted');
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not delete');
    }
  }

  async loadHistory(): Promise<boolean> {
    if (!this.currentId) return false;
    try {
      this.versions = await api.get<ReceiptTemplateVersion[]>(`${BASE}/templates/${this.currentId}/versions`);
      return true;
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not load history');
      return false;
    }
  }

  restore(v: ReceiptTemplateVersion): void {
    this.body = structuredClone(v.body);
    this.selected = 0;
    this.schedulePreview(0);
    toast.info(`Version ${v.version} loaded — save to make it current`);
  }
}
