<script lang="ts">
  import { Icon } from '@mauzo/ui';
  import { desktop } from '../desktop';
  import { mpesafeed } from '../mpesafeed.svelte';
  import { pulse } from '../pulse.svelte';
  import FailedSalesModal from './FailedSalesModal.svelte';
  import { shift, refreshShift, shiftDrawer } from '../shift.svelte';
  import { formatMoney } from '../money';
  // The "More" action sheet — switch cashier, lock, held sales, M-Pesa feed,
  // find sale, shift actions, exit. Sync lives on the top bar now.
  let {
    onclose,
    onlock,
    onswitch,
    onheld,
    onlookup,
    onmpesa,
    onshift,
  }: {
    onclose: () => void;
    onlock: () => void;
    onswitch: () => void;
    onheld: () => void;
    onlookup: () => void;
    onmpesa: () => void;
    onshift: (m: 'shiftopen' | 'shiftcash' | 'shiftclose' | 'shiftreport') => void;
  } = $props();

  let failedOpen = $state(false);
  let drawerBusy = $state(false);
  $effect(() => void refreshShift());

  async function drawer() {
    drawerBusy = true;
    await shiftDrawer().catch(() => {});
    drawerBusy = false;
  }

  function run(fn: () => void) {
    return () => {
      fn();
      onclose();
    };
  }

  function quit() {
    if (confirm('Exit Mauzo POS?')) void desktop()?.Quit();
  }
</script>

<div class="modal">
  <h3>More</h3>
  {#if shift.checked}
    <div class="shiftrow" class:off={!shift.current}>
      {#if shift.current}
        <Icon name="clock" size={16} />
        <span>Shift: <b>{shift.current.opened_by_name}</b> · float {formatMoney(shift.current.float)}</span>
      {:else}
        <Icon name="alert" size={16} />
        <span>No shift open — sales still work, cash is untracked.</span>
      {/if}
    </div>
  {/if}
  <div class="grid">
    <button class="btn" onclick={run(onswitch)}>
      <Icon name="user" size={26} /><span>Switch cashier</span>
    </button>
    <button class="btn" onclick={run(onlock)}><Icon name="lock" size={26} /><span>Lock till</span></button>
    <button class="btn" onclick={onheld}><Icon name="pause" size={26} /><span>Held sales</span></button>
    <button class="btn" onclick={onlookup}><Icon name="search" size={26} /><span>Find sale</span></button>
    <button class="btn" onclick={onmpesa}>
      <Icon name="phone" size={26} /><span>M-Pesa</span>
      {#if mpesafeed.count > 0}<em class="badge">{mpesafeed.count}</em>{/if}
    </button>
    {#if !shift.current}
      <button class="btn" onclick={() => onshift('shiftopen')}>
        <Icon name="clock" size={26} /><span>Open shift</span>
      </button>
    {:else}
      <button class="btn" onclick={() => onshift('shiftcash')}>
        <Icon name="banknote" size={26} /><span>Cash in / out</span>
      </button>
      <button class="btn" onclick={() => onshift('shiftreport')}>
        <Icon name="file" size={26} /><span>X report</span>
      </button>
      <button class="btn" onclick={() => onshift('shiftclose')}>
        <Icon name="logout" size={26} /><span>Close shift</span>
      </button>
    {/if}
    <button class="btn" onclick={drawer} disabled={drawerBusy}>
      <Icon name="inbox" size={26} /><span>Open drawer</span>
    </button>
    <button class="btn" onclick={quit}><Icon name="power" size={26} /><span>Exit app</span></button>
    {#if (pulse.status?.failed_sales ?? 0) > 0}
      <button class="btn" onclick={() => (failedOpen = true)}>
        <Icon name="alert" size={26} /><span>Needs attention</span>
        <em class="badge">{pulse.status?.failed_sales}</em>
      </button>
    {/if}
  </div>
  <button class="ghost" onclick={onclose}>Close</button>
</div>

{#if failedOpen}<FailedSalesModal onclose={() => (failedOpen = false)} />{/if}

<style>
  .modal {
    width: min(420px, 92vw);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: 0 12px 40px rgb(0 0 0 / 0.3);
  }
  h3 {
    margin: 0 0 var(--space-3);
  }
  .shiftrow {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    margin-bottom: var(--space-3);
    border-radius: var(--radius-md);
    background: var(--color-accent-soft, #e8f0ff);
    font-size: var(--text-sm);
  }
  .shiftrow.off {
    background: var(--color-danger-soft, #fde8e8);
    color: var(--color-danger, #c0392b);
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-2);
    margin-bottom: var(--space-3);
  }
  .btn {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    padding: var(--space-4);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    cursor: pointer;
    color: var(--color-primary);
  }
  .badge {
    position: absolute;
    top: 6px;
    right: 10px;
    background: #00a651;
    color: #fff;
    font-size: 0.7rem;
    font-style: normal;
    font-weight: 700;
    border-radius: 999px;
    padding: 1px 7px;
  }
  .btn span {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  .ghost {
    margin-top: var(--space-3);
    padding: var(--space-2) var(--space-4);
    font: inherit;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    cursor: pointer;
  }
</style>
