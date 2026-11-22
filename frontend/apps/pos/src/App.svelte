<script lang="ts">
  import { onMount } from 'svelte';
  import { Button, StatusDot, Toaster, toast } from '@mauzo/ui';
  import { formatDate, formatTime } from '@mauzo/utils';
  import { desktop, type ServerStatus, type TerminalInfo } from './lib/desktop';

  const bridge = desktop();

  let now = $state(new Date());
  let info = $state<TerminalInfo | null>(null);
  let server = $state<ServerStatus | null>(null);
  let checking = $state(false);

  async function checkServer() {
    if (!bridge) return;
    checking = true;
    try {
      server = await bridge.CheckServer();
      if (!server.reachable) toast.error(server.error ?? 'Server unreachable');
    } finally {
      checking = false;
    }
  }

  onMount(() => {
    const tick = setInterval(() => (now = new Date()), 1000);
    if (bridge) {
      bridge.Info().then((i) => (info = i));
      checkServer();
    }
    return () => clearInterval(tick);
  });
</script>

<header class="topbar">
  <div class="brand">
    <strong>Mauzo POS</strong>
    {#if info}<span class="muted">v{info.version}</span>{/if}
  </div>
  <div class="right">
    {#if server}
      <StatusDot status={server.reachable ? 'ok' : 'error'} label={server.reachable ? 'Online' : 'Offline'} />
    {/if}
    <span>{formatDate(now)} {formatTime(now)}</span>
  </div>
</header>

<main>
  <section class="card">
    {#if !bridge}
      <h1>Desktop runtime not detected</h1>
      <p>
        This UI is running in a plain browser. Start it with <code>wails dev</code> from
        <code>backend/desktop</code> to use terminal features.
      </p>
    {:else}
      <h1>{info?.activated ? 'Terminal ready' : 'Terminal not activated'}</h1>
      <p class="muted">
        Terminal activation and cashier PIN login arrive in Phase 6. This screen verifies the desktop shell
        can reach the central API.
      </p>
      <dl>
        <dt>Server</dt>
        <dd>{info?.api_base_url ?? '—'}</dd>
        <dt>Status</dt>
        <dd>
          {#if server}
            {server.reachable
              ? `${server.status} (v${server.version}, ${server.latency_ms} ms)`
              : server.error}
          {:else}
            checking…
          {/if}
        </dd>
      </dl>
      <Button size="lg" variant="secondary" loading={checking} onclick={checkServer}>Check connection</Button>
    {/if}
  </section>
</main>

<Toaster />

<style>
  :global(body) {
    height: 100vh;
    overflow: hidden;
  }
  .topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-3) var(--space-5);
    background: var(--color-header);
    color: var(--color-header-text);
    font-size: var(--text-lg);
  }
  .brand {
    display: flex;
    gap: var(--space-3);
    align-items: baseline;
  }
  .brand strong {
    font-size: var(--text-2xl);
  }
  .right {
    display: flex;
    gap: var(--space-5);
    align-items: center;
  }
  .muted {
    color: var(--color-text-muted);
  }
  .topbar .muted {
    color: var(--color-header-text);
    opacity: 0.7;
  }
  main {
    display: grid;
    place-items: center;
    height: calc(100vh - 64px);
  }
  .card {
    width: min(560px, 92vw);
    padding: var(--space-6);
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md);
  }
  h1 {
    margin-top: 0;
  }
  dl {
    display: grid;
    grid-template-columns: 100px 1fr;
    gap: var(--space-2);
    margin: var(--space-5) 0;
  }
  dt {
    font-weight: 600;
  }
  dd {
    margin: 0;
    word-break: break-all;
  }
</style>
