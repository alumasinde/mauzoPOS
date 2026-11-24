<script lang="ts">
  import { Icon } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import type { Status, TerminalInfo, Session } from '../desktop';
  import { mpesafeed } from '../mpesafeed.svelte';
  import { pulse } from '../pulse.svelte';

  let {
    info,
    session,
    status,
    now,
    onsync,
    onmpesa,
    onsettings,
    onlock,
  }: {
    info: TerminalInfo | null;
    session: Session;
    status: Status | null;
    now: Date;
    onsync: () => void;
    onmpesa: () => void;
    onsettings: () => void;
    onlock: () => void;
  } = $props();
</script>

<header class="topbar">
  <div class="brand">
    <span class="logo"><Icon name="cart" size={26} /></span>
    <div class="names">
      <strong>{info?.tenant ?? 'Mauzo POS'}</strong>
      <span class="tag">Sell smart.</span>
    </div>
  </div>
  <div class="meta">
    <span>Branch: <b>{info?.branch ?? '—'}</b></span>
    <span class="sep">|</span>
    <span>Till: <b>{info?.terminal ?? '—'}</b></span>
    <span class="sep">|</span>
    <span>Cashier: <b>{session.name}</b></span>
    <button
      class="dot"
      class:on={status?.online}
      title={status?.online
        ? `Online · synced ${status?.last_sync || 'never'}`
        : 'Offline — sales queue locally'}
      onclick={onsync}
    >
      <i></i>{status?.online ? 'Online' : 'Offline'}
      {#if status && status.pending_sales > 0}<em class="pend">{status.pending_sales}</em>{/if}
      {#if status && status.failed_sales > 0}
        <em class="pend bad" title="Offline sales refused by the server — open More → Needs attention">
          {status.failed_sales} need attention
        </em>
      {/if}
    </button>
    <span
      class="printer"
      class:warn={status?.printer !== 'ready'}
      title={status?.printer === 'ready' ? 'Printer ready' : 'No printer configured'}
    >
      <Icon name="printer" size={18} />
    </span>
    <button
      class="till-btn feed"
      onclick={onmpesa}
      title="M-Pesa payments — collections arriving at this till"
    >
      <Icon name="phone" size={17} />
      {#if mpesafeed.count > 0}<em class="feed-badge">{mpesafeed.count}</em>{/if}
    </button>
    <button
      class="till-btn"
      class:spin={pulse.syncing}
      disabled={pulse.syncing}
      onclick={onsync}
      title="Sync catalog & upload queued sales"
    >
      <Icon name="refresh" size={17} />
    </button>
  </div>
  <div class="right">
    <span class="clock"><Icon name="clock" size={15} /> {formatDate(now)} {formatTime(now)}</span>
    <button class="icon" onclick={onsettings} title="Terminal settings"><Icon name="cog" size={20} /></button>
    <button class="icon" onclick={onlock} title="Lock (switch cashier)">
      <Icon name="lock" size={20} />
    </button>
  </div>
</header>

<style>
  .topbar {
    display: flex;
    align-items: center;
    gap: var(--space-5);
    padding: var(--space-2) var(--space-4);
    background: var(--color-header);
    color: var(--color-header-text);
    height: 56px;
    flex-shrink: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .logo {
    display: inline-flex;
    color: var(--color-primary);
    background: var(--color-surface);
    border-radius: var(--radius-sm);
    padding: 4px;
  }
  .names {
    display: flex;
    flex-direction: column;
    line-height: 1.1;
  }
  .names strong {
    font-size: var(--text-lg);
  }
  .tag {
    font-size: var(--text-xs);
    opacity: 0.65;
  }
  .meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex: 1;
    font-size: var(--text-sm);
  }
  .sep {
    opacity: 0.4;
  }
  .dot {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: none;
    border: none;
    color: inherit;
    font: inherit;
    cursor: pointer;
  }
  .dot i {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--color-danger);
  }
  .dot.on i {
    background: #3ddc74;
  }
  .pend {
    font-style: normal;
    background: var(--color-danger);
    color: #fff;
    font-size: 0.65rem;
    border-radius: 8px;
    padding: 0 5px;
  }
  .printer {
    display: inline-flex;
    opacity: 0.9;
  }
  .printer.warn {
    opacity: 0.45;
  }
  .till-btn {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border: none;
    border-radius: var(--radius-sm);
    background: rgb(255 255 255 / 0.08);
    color: inherit;
    cursor: pointer;
    opacity: 0.9;
  }
  .till-btn:hover {
    opacity: 1;
    background: rgb(255 255 255 / 0.16);
  }
  .till-btn:disabled {
    cursor: default;
  }
  .till-btn.feed {
    color: #5ee08a;
  }
  .feed-badge {
    position: absolute;
    top: -5px;
    right: -6px;
    min-width: 15px;
    padding: 0 4px;
    border-radius: 999px;
    background: #00a651;
    color: #fff;
    font-size: 0.62rem;
    font-style: normal;
    font-weight: 700;
    line-height: 15px;
    text-align: center;
  }
  .till-btn.spin :global(svg) {
    animation: spin 0.9s linear infinite;
  }
  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
  .right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    font-size: var(--text-sm);
  }
  .clock {
    display: inline-flex;
    align-items: center;
    gap: 5px;
  }
  .icon {
    background: none;
    border: none;
    color: inherit;
    font-size: 1.2rem;
    cursor: pointer;
    opacity: 0.85;
  }
  .icon:hover {
    opacity: 1;
  }
</style>
