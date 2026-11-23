<script lang="ts">
  import { Icon } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import type { Status, TerminalInfo, Session } from '../desktop';

  let {
    info,
    session,
    status,
    now,
    onsync,
    onsettings,
    onlock,
  }: {
    info: TerminalInfo | null;
    session: Session;
    status: Status | null;
    now: Date;
    onsync: () => void;
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
    </button>
    <span
      class="printer"
      class:warn={status?.printer !== 'ready'}
      title={status?.printer === 'ready' ? 'Printer ready' : 'No printer configured'}
    >
      <Icon name="printer" size={18} />
    </span>
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
