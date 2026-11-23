<script lang="ts">
  // Receipt text and printing policy (tenant settings, synced to tills).
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import { Button, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let { onsaved }: { onsaved: () => void } = $props();

  interface Form {
    footer: string;
    mpesa_till: string;
    mpesa_paybill: string;
    mpesa_account: string;
    auto_print: boolean;
    copies: number;
    open_drawer: boolean;
  }
  let form = $state<Form>({
    footer: '',
    mpesa_till: '',
    mpesa_paybill: '',
    mpesa_account: '',
    auto_print: true,
    copies: 1,
    open_drawer: true,
  });
  let initial = '';
  let saving = $state(false);
  const canEdit = $derived(auth.can('settings.business.update'));

  onMount(async () => {
    try {
      const rows = await api.get<{ key: string; value: unknown }[]>('/api/v1/t/settings');
      const v = Object.fromEntries(rows.map((r) => [r.key, r.value]));
      form = {
        footer: (v['receipt.footer'] as string) ?? '',
        mpesa_till: (v['receipt.mpesa_till'] as string) ?? '',
        mpesa_paybill: (v['receipt.mpesa_paybill'] as string) ?? '',
        mpesa_account: (v['receipt.mpesa_account'] as string) ?? '',
        auto_print: (v['receipt.auto_print'] as boolean) ?? true,
        copies: (v['receipt.copies'] as number) ?? 1,
        open_drawer: (v['receipt.open_drawer'] as boolean) ?? true,
      };
      initial = JSON.stringify(form);
    } catch {
      /* settings may be hidden for this role */
    }
  });

  async function save() {
    saving = true;
    try {
      const before = JSON.parse(initial || '{}') as Partial<Form>;
      const keys: [keyof Form, string][] = [
        ['footer', 'receipt.footer'],
        ['mpesa_till', 'receipt.mpesa_till'],
        ['mpesa_paybill', 'receipt.mpesa_paybill'],
        ['mpesa_account', 'receipt.mpesa_account'],
        ['auto_print', 'receipt.auto_print'],
        ['copies', 'receipt.copies'],
        ['open_drawer', 'receipt.open_drawer'],
      ];
      for (const [field, key] of keys) {
        const value = typeof form[field] === 'string' ? (form[field] as string).trim() : form[field];
        if (before[field] === value) continue;
        await api.put(`/api/v1/t/settings/${key}`, { value });
      }
      initial = JSON.stringify(form);
      toast.success('Receipt settings saved');
      onsaved();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }
</script>

<div class="grid">
  <section>
    <h3>Printed details</h3>
    <label class="lbl" for="rs-footer">Footer message</label>
    <textarea
      id="rs-footer"
      rows="3"
      maxlength="500"
      bind:value={form.footer}
      disabled={!canEdit}
      placeholder="e.g. Prices include VAT. Goods once sold are not returnable."></textarea>
    <div class="three">
      <TextField
        label="M-Pesa till"
        bind:value={form.mpesa_till}
        disabled={!canEdit}
        maxlength={20}
        inputmode="numeric"
      />
      <TextField
        label="Paybill"
        bind:value={form.mpesa_paybill}
        disabled={!canEdit}
        maxlength={20}
        inputmode="numeric"
      />
      <TextField label="Account number" bind:value={form.mpesa_account} disabled={!canEdit} maxlength={40} />
    </div>
    <p class="hint">Layouts can print these with {'{{extra.mpesa_till}}'} etc. The mini-mart layout does.</p>
  </section>
  <section>
    <h3>Printing at the till</h3>
    <label class="chk">
      <input type="checkbox" bind:checked={form.auto_print} disabled={!canEdit} />
      Print a receipt automatically after every sale
    </label>
    <label class="chk">
      <input type="checkbox" bind:checked={form.open_drawer} disabled={!canEdit} />
      Open the cash drawer for cash payments
    </label>
    <label class="lbl" for="rs-copies">Copies per sale</label>
    <select id="rs-copies" bind:value={form.copies} disabled={!canEdit}>
      <option value={1}>1 — customer copy</option>
      <option value={2}>2 — customer + shop copy</option>
      <option value={3}>3</option>
    </select>
    <p class="hint">
      A till can override automatic printing in its own settings (e.g. a till with no printer).
    </p>
  </section>
</div>
{#if canEdit}
  <div class="actions"><Button onclick={save} loading={saving}>Save receipt settings</Button></div>
{/if}

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: var(--space-5);
  }
  h3 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-md);
  }
  .lbl {
    display: block;
    font-size: var(--text-sm);
    font-weight: 600;
    margin: var(--space-3) 0 var(--space-1);
  }
  textarea,
  select {
    width: 100%;
    padding: var(--space-2);
    font: inherit;
    color: var(--color-text);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
  }
  .three {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
  .chk {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    margin-bottom: var(--space-2);
  }
  .hint {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    margin: var(--space-2) 0 0;
  }
  .actions {
    margin-top: var(--space-5);
  }
</style>
