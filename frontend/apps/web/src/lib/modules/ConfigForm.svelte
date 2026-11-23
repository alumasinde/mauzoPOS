<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { ConfigSchema, ModuleInfo } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';

  interface Props {
    open: boolean;
    module: ModuleInfo | null;
    onsave: (config: Record<string, unknown>) => Promise<void>;
  }

  let { open = $bindable(), module, onsave }: Props = $props();

  // Form values are strings/booleans; they are converted per schema on save.
  let values = $state<Record<string, string | boolean>>({});
  let saving = $state(false);
  let error = $state<string | null>(null);

  const fields = $derived(Object.entries(module?.config_schema?.properties ?? {}));

  function label(key: string): string {
    return key.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase());
  }

  $effect(() => {
    if (!open || !module) return;
    error = null;
    const cur = module.config ?? {};
    const next: Record<string, string | boolean> = {};
    for (const [key, sch] of fields) {
      const v = cur[key];
      if (sch.type === 'boolean') next[key] = v === true;
      else if (sch.type === 'array') next[key] = Array.isArray(v) ? v.join(', ') : '';
      else next[key] = v === undefined || v === null ? '' : String(v);
    }
    values = next;
  });

  function convert(sch: ConfigSchema, raw: string | boolean): unknown {
    if (sch.type === 'boolean') return raw === true;
    const s = String(raw).trim();
    if (s === '') return undefined;
    if (sch.type === 'number' || sch.type === 'integer') return Number(s);
    if (sch.type === 'array') {
      return s
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean);
    }
    return s;
  }

  async function save() {
    saving = true;
    error = null;
    const out: Record<string, unknown> = {};
    for (const [key, sch] of fields) {
      const v = convert(sch, values[key] ?? '');
      if (v !== undefined) out[key] = v;
    }
    try {
      await onsave(out);
      toast.success('Settings saved');
      open = false;
    } catch (e) {
      error =
        e instanceof ApiError ? ((e.details?.config as string | undefined) ?? e.message) : 'Save failed';
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title={`${module?.name ?? ''} settings`} locked={saving}>
  <div class="grid">
    {#each fields as [key, sch] (key)}
      {#if sch.type === 'boolean'}
        <label class="check">
          <input type="checkbox" bind:checked={values[key] as boolean} />
          {label(key)}
        </label>
      {:else if sch.enum}
        <label class="field">
          <span>{label(key)}</span>
          <select bind:value={values[key]}>
            <option value="">—</option>
            {#each sch.enum as opt (opt)}<option value={String(opt)}>{opt}</option>{/each}
          </select>
        </label>
      {:else}
        <TextField
          label={label(key)}
          inputmode={sch.type === 'number' || sch.type === 'integer' ? 'decimal' : undefined}
          bind:value={values[key] as string}
          hint={sch.type === 'array'
            ? 'Comma-separated'
            : sch.minimum !== undefined || sch.maximum !== undefined
              ? `Between ${sch.minimum ?? '…'} and ${sch.maximum ?? '…'}`
              : undefined}
        />
      {/if}
    {/each}
    {#if error}<p class="error">{error}</p>{/if}
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={save} loading={saving}>Save</Button>
  {/snippet}
</Modal>

<style>
  .grid {
    display: grid;
    gap: var(--space-4);
  }
  .check {
    display: flex;
    gap: var(--space-2);
    align-items: center;
  }
  .field {
    display: grid;
    gap: var(--space-1);
    font-size: var(--text-sm);
    font-weight: 600;
  }
  select {
    min-height: 40px;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-text);
    font: inherit;
  }
  .error {
    color: var(--color-danger);
    margin: 0;
  }
</style>
