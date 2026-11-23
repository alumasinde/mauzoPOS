<script lang="ts">
  import { cart } from '../cart.svelte';
  import { formatMoney } from '../money';

  type Method = 'cash' | 'card' | 'bank' | 'credit';
  let {
    currency = 'KSh',
    method = $bindable<Method>('cash'),
    onaction,
  }: {
    currency?: string;
    method?: Method;
    onaction: (a: string) => void;
  } = $props();

  const methods: { key: Method; label: string; icon: string }[] = [
    { key: 'cash', label: 'Cash', icon: '💵' },
    { key: 'card', label: 'Card', icon: '💳' },
    { key: 'bank', label: 'Bank', icon: '🏦' },
    { key: 'credit', label: 'Credit', icon: '📒' },
  ];
</script>

<footer class="bar">
  <div class="acts">
    <button class="act danger" onclick={() => onaction('cancel')}>
      <span class="ic">✕</span><span>Cancel</span><small>Esc</small>
    </button>
    <button class="act" onclick={() => onaction('hold')}>
      <span class="ic">⏸</span><span>Hold Sale</span><small>F5</small>
    </button>
    <button class="act" onclick={() => onaction('pricecheck')}>
      <span class="ic">🏷</span><span>Price Check</span><small>F6</small>
    </button>
    <button class="act" onclick={() => onaction('quantity')}>
      <span class="ic">▦</span><span>Quantity</span><small>F7</small>
    </button>
    <button class="act" onclick={() => onaction('discount')}>
      <span class="ic">%</span><span>Discount</span><small>F8</small>
    </button>
    <button class="act" onclick={() => onaction('more')}>
      <span class="ic">•••</span><span>More</span><small>▾</small>
    </button>
  </div>

  <div class="pay">
    <span class="plabel">Select Payment Method:</span>
    <div class="pmethods">
      {#each methods as m (m.key)}
        <button class="pm" class:on={method === m.key} onclick={() => (method = m.key)}>
          <span class="pi">{m.icon}</span>{m.label}
        </button>
      {/each}
    </div>
    <button class="paynow" disabled={cart.count === 0} onclick={() => onaction('pay')}>
      <span class="ck">✔</span>
      Pay {currency}
      {formatMoney(cart.totals.total)}
      <small>F10</small>
    </button>
  </div>
</footer>

<style>
  .bar {
    display: flex;
    gap: var(--space-3);
    align-items: stretch;
    padding: var(--space-2) var(--space-3);
    background: var(--color-surface);
    border-top: 1px solid var(--color-border);
    flex-shrink: 0;
  }
  .acts {
    display: flex;
    gap: var(--space-2);
  }
  .act {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-width: 92px;
    padding: var(--space-2) var(--space-3);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    background: var(--color-primary);
    color: #fff;
    font-size: var(--text-sm);
    font-weight: 600;
    gap: 1px;
  }
  .act:hover {
    background: var(--color-primary-hover);
  }
  .act.danger {
    background: var(--color-danger);
  }
  .act.danger:hover {
    background: var(--color-danger-hover);
  }
  .ic {
    font-size: 1.2rem;
  }
  .act small {
    font-size: 0.65rem;
    opacity: 0.75;
  }
  .pay {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-left: auto;
  }
  .plabel {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
    max-width: 70px;
    text-align: right;
  }
  .pmethods {
    display: flex;
    gap: var(--space-2);
  }
  .pm {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
    min-width: 74px;
    padding: var(--space-2);
    border: none;
    border-radius: var(--radius-md);
    background: #2f5fb3;
    color: #fff;
    font-weight: 600;
    font-size: var(--text-sm);
    cursor: pointer;
    opacity: 0.8;
  }
  .pm.on {
    opacity: 1;
    outline: 3px solid color-mix(in srgb, var(--color-primary) 40%, transparent);
  }
  .pm.on {
    opacity: 1;
  }
  .pm:nth-child(1) {
    background: var(--color-success);
  }
  .pm:nth-child(2) {
    background: #237a3d;
  }
  .pm:nth-child(3) {
    background: #2f5fb3;
  }
  .pm:nth-child(4) {
    background: #5a6270;
  }
  .pi {
    font-size: 1.3rem;
  }
  .paynow {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-width: 170px;
    padding: var(--space-2) var(--space-4);
    background: var(--color-success);
    color: #fff;
    border: none;
    border-radius: var(--radius-md);
    font-size: var(--text-lg);
    font-weight: 800;
    cursor: pointer;
  }
  .paynow:hover:not(:disabled) {
    background: var(--color-success-hover);
  }
  .paynow:disabled {
    opacity: 0.55;
    cursor: default;
  }
  .paynow small {
    font-size: 0.7rem;
    font-weight: 500;
    opacity: 0.8;
  }
  .ck {
    font-size: 1.4rem;
  }
</style>
