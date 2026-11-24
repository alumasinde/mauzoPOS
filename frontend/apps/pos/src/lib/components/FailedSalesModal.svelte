<script lang="ts">
  // Offline sales the server refused on upload. They are parked here (not
  // retried forever) so a supervisor can see why and retry once the cause is
  // fixed — e.g. a product re-activated in the back office.
  import { onMount } from 'svelte';
  import { desktop, type FailedSale } from '../desktop';
  import { formatMoney } from '../money';
  import { pulse } from '../pulse.svelte';

  let { onclose }: { onclose: () => void } = $props();

  const bridge = desktop();
  let rows = $state<FailedSale[]>([]);
  let busy = $state('');
  let message = $state('');

  async function load() {
    rows = (await bridge?.FailedSales().catch(() => [])) ?? [];
  }
  onMount(load);

  function total(s: FailedSale): string {
    try {
      const p = JSON.parse(s.payload) as { payments?: { amount: string }[] };
      return formatMoney(String((p.payments ?? []).reduce((n, x) => n + Number(x.amount), 0)));
    } catch {
      return '—';
    }
  }

  async function retry(s: FailedSale) {
    if (!bridge) return;
    busy = s.id;
    message = '';
    try {
      const n = await bridge.RetryFailedSale(s.id);
      message = n > 0 ? 'Uploaded.' : 'Still refused — see the reason.';
    } catch (e) {
      message = e instanceof Error ? e.message : 'Retry failed';
    } finally {
      busy = '';
      await load();
      void pulse.refresh();
    }
  }
</script>

<div class="overlay" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="modal">
    <h3>Sales needing attention</h3>
    <p class="muted">
      Recorded while offline but refused by the server. Stock and cash already moved — fix the cause, then
      retry.
    </p>
    {#if rows.length === 0}
      <p class="empty">Nothing waiting.</p>
    {:else}
      <ul>
        {#each rows as s (s.id)}
          <li>
            <div>
              <b>KSh {total(s)}</b> · {new Date(s.created_at).toLocaleString('en-KE')}
              <p class="why">{s.error}</p>
            </div>
            <button onclick={() => retry(s)} disabled={busy === s.id}>{busy === s.id ? '…' : 'Retry'}</button>
          </li>
        {/each}
      </ul>
    {/if}
    {#if message}<p class="msg">{message}</p>{/if}
    <button class="close" onclick={onclose}>Close</button>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: grid;
    place-items: center;
    background: rgb(0 0 0 / 0.5);
  }
  .modal {
    width: min(560px, 94vw);
    max-height: 86vh;
    overflow-y: auto;
    padding: var(--space-5);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
  }
  h3 {
    margin: 0 0 var(--space-2);
  }
  .muted,
  .empty {
    color: var(--color-text-muted);
  }
  ul {
    list-style: none;
    margin: var(--space-3) 0;
    padding: 0;
    display: grid;
    gap: var(--space-2);
  }
  li {
    display: flex;
    justify-content: space-between;
    gap: var(--space-3);
    align-items: center;
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
  }
  .why {
    margin: 4px 0 0;
    color: var(--color-danger);
    font-size: var(--text-sm);
  }
  li button,
  .close {
    padding: 10px 16px;
    font: inherit;
    font-weight: 700;
    border-radius: var(--radius-md);
    cursor: pointer;
  }
  li button {
    color: #fff;
    background: var(--color-primary);
    border: 0;
  }
  .close {
    width: 100%;
    background: none;
    border: 1px solid var(--color-border);
    color: var(--color-text);
  }
  .msg {
    font-weight: 600;
  }
</style>
