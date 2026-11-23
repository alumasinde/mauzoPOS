<script lang="ts">
  import { ApiError } from '@mauzo/api-client';
  import type { Category, Product, Promotion } from '@mauzo/types';
  import { Button, Modal, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  interface Props {
    target: Promotion | null;
    open: boolean;
    onclose: () => void;
    ondone: () => void;
  }
  let { target, open = $bindable(), onclose, ondone }: Props = $props();

  const DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

  let name = $state('');
  let kind = $state<'percent' | 'amount' | 'bxgy'>('percent');
  let percent = $state('');
  let amount = $state('');
  let buy = $state('2');
  let get = $state('1');
  let getPercent = $state('100');
  let startsAt = $state('');
  let endsAt = $state('');
  let priority = $state('0');
  let dayMask = $state(127); // all days
  let targetProduct = $state('');
  let targetCategory = $state('');
  let products = $state<Product[]>([]);
  let categories = $state<Category[]>([]);
  let saving = $state(false);

  async function ensureData() {
    try {
      const [ps, cs] = await Promise.all([
        api.list<Product>('/api/v1/t/catalog/products', { per_page: 200 }),
        api.get<Category[]>('/api/v1/t/catalog/categories'),
      ]);
      products = ps.data;
      categories = cs;
    } catch {
      /* pickers best-effort */
    }
  }

  $effect(() => {
    if (open) {
      ensureData();
      name = target?.name ?? '';
      kind = target?.kind ?? 'percent';
      percent = target?.config.percent ?? '';
      amount = target?.config.amount ?? '';
      buy = target?.config.buy ?? '2';
      get = target?.config.get ?? '1';
      getPercent = target?.config.get_percent ?? '100';
      startsAt = target?.starts_at?.slice(0, 16) ?? '';
      endsAt = target?.ends_at?.slice(0, 16) ?? '';
      priority = String(target?.priority ?? 0);
      dayMask = target?.days_mask ?? 127;
      targetProduct = target?.targets?.[0]?.product_id ?? '';
      targetCategory = target?.targets?.[0]?.category_id ?? '';
    }
  });

  function toggleDay(i: number) {
    dayMask ^= 1 << i;
  }

  async function save() {
    if (!name.trim()) {
      toast.error('Name is required');
      return;
    }
    const config: Record<string, string> = {};
    if (kind === 'percent') config.percent = percent;
    else if (kind === 'amount') config.amount = amount;
    else {
      config.buy = buy;
      config.get = get;
      config.get_percent = getPercent;
    }
    const targets =
      targetProduct || targetCategory
        ? [
            {
              ...(targetProduct ? { product_id: targetProduct } : {}),
              ...(targetCategory ? { category_id: targetCategory } : {}),
            },
          ]
        : [];
    saving = true;
    try {
      const body = {
        name: name.trim(),
        kind,
        config,
        starts_at: startsAt ? new Date(startsAt).toISOString() : null,
        ends_at: endsAt ? new Date(endsAt).toISOString() : null,
        days_mask: dayMask === 127 ? null : dayMask,
        priority: Number(priority) || 0,
        targets,
      };
      if (target) await api.patch(`/api/v1/t/promotions/${target.id}`, body);
      else await api.post('/api/v1/t/promotions/', body);
      toast.success(target ? 'Promotion updated' : 'Promotion created');
      ondone();
      onclose();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Save failed');
    } finally {
      saving = false;
    }
  }
</script>

<Modal bind:open title={target ? 'Edit promotion' : 'New promotion'} size="lg" {onclose}>
  <div class="grid2">
    <TextField label="Name" bind:value={name} />
    <label class="f">
      <span>Kind</span>
      <select bind:value={kind}>
        <option value="percent">Percent off</option>
        <option value="amount">Amount off (per unit)</option>
        <option value="bxgy">Buy X get Y</option>
      </select>
    </label>
  </div>

  {#if kind === 'percent'}
    <TextField label="Percent off (e.g. 10)" type="number" bind:value={percent} />
  {:else if kind === 'amount'}
    <TextField label="Amount off per unit (KES)" type="number" bind:value={amount} />
  {:else}
    <div class="grid3">
      <TextField label="Buy qty" type="number" bind:value={buy} />
      <TextField label="Get qty" type="number" bind:value={get} />
      <TextField label="Get % off (100 = free)" type="number" bind:value={getPercent} />
    </div>
  {/if}

  <div class="grid2">
    <label class="f">
      <span>Starts at</span>
      <input type="datetime-local" bind:value={startsAt} />
    </label>
    <label class="f">
      <span>Ends at</span>
      <input type="datetime-local" bind:value={endsAt} />
    </label>
  </div>

  <label class="f">
    <span>Active days</span>
    <div class="days">
      {#each DAYS as d, i (d)}
        <button type="button" class:on={dayMask & (1 << i)} onclick={() => toggleDay(i)}>{d}</button>
      {/each}
    </div>
  </label>

  <div class="grid2">
    <label class="f">
      <span>Target product</span>
      <select bind:value={targetProduct}>
        <option value="">Any</option>
        {#each products as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
    </label>
    <label class="f">
      <span>Target category</span>
      <select bind:value={targetCategory}>
        <option value="">Any</option>
        {#each categories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
      </select>
    </label>
  </div>
  <p class="hint">No targets = applies to the whole catalog.</p>

  <TextField label="Priority (higher wins on ties)" type="number" bind:value={priority} />

  {#snippet footer()}
    <Button variant="ghost" onclick={onclose}>Cancel</Button>
    <Button onclick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
  {/snippet}
</Modal>

<style>
  .grid2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .grid3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 10px;
  }
  .f {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 8px;
  }
  .f span {
    font-size: 12px;
    color: var(--fg-muted, #666);
  }
  .f select,
  .f input {
    padding: 6px 8px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: var(--bg, #fff);
    color: var(--fg, #222);
  }
  .days {
    display: flex;
    gap: 4px;
  }
  .days button {
    padding: 4px 10px;
    border: 1px solid var(--border, #ddd);
    border-radius: 6px;
    background: none;
    cursor: pointer;
    font: inherit;
    color: var(--fg-muted, #666);
  }
  .days button.on {
    background: var(--accent, #2563eb);
    border-color: var(--accent, #2563eb);
    color: #fff;
  }
  .hint {
    font-size: 12px;
    color: var(--fg-muted, #666);
    margin: 0 0 8px;
  }
</style>
