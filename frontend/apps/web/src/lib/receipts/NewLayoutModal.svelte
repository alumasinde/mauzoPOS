<script lang="ts">
  // Start a layout from a shipped layout or a copy of an existing one.
  import { ApiError } from '@mauzo/api-client';
  import type { ReceiptBuiltin, ReceiptTemplate } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  interface Props {
    open: boolean;
    builtins: ReceiptBuiltin[];
    templates: ReceiptTemplate[];
    oncreated: (t: ReceiptTemplate) => void;
  }
  let { open = $bindable(), builtins, templates, oncreated }: Props = $props();

  let name = $state('');
  let from = $state('');
  let busy = $state(false);

  $effect(() => {
    if (!open) return;
    name = '';
    from = builtins[0] ? `builtin:${builtins[0].key}` : '';
  });

  async function create() {
    if (!name.trim() || !from) return;
    busy = true;
    try {
      const [kind, ref] = from.split(':');
      const t = await api.post<ReceiptTemplate>('/api/v1/t/receipts/templates', {
        name: name.trim(),
        ...(kind === 'builtin' ? { builtin: ref } : { copy_from: ref }),
      });
      open = false;
      oncreated(t);
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not create the layout');
    } finally {
      busy = false;
    }
  }
</script>

<Modal bind:open title="New receipt layout" locked={busy}>
  <div class="form">
    <TextField label="Name" bind:value={name} placeholder="e.g. Main counter" maxlength={120} />
    <span class="lbl">Start from</span>
    {#each builtins as b (b.key)}
      <label class="opt">
        <input type="radio" name="from" value={`builtin:${b.key}`} bind:group={from} />
        <span><strong>{b.name}</strong><small>{b.description}</small></span>
      </label>
    {/each}
    {#each templates as t (t.id)}
      <label class="opt">
        <input type="radio" name="from" value={`copy:${t.id}`} bind:group={from} />
        <span><strong>Copy of “{t.name}”</strong><small>{t.paper_width} mm, version {t.version}</small></span>
      </label>
    {/each}
  </div>
  {#snippet footer()}
    <Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
    <Button onclick={create} loading={busy} disabled={!name.trim() || !from}>Create</Button>
  {/snippet}
</Modal>

<style>
  .form {
    display: grid;
    gap: var(--space-2);
  }
  .lbl {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  .opt {
    display: flex;
    gap: var(--space-3);
    align-items: flex-start;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    cursor: pointer;
  }
  .opt span {
    display: flex;
    flex-direction: column;
  }
  .opt small {
    color: var(--color-text-muted);
  }
</style>
