<script lang="ts">
  import { onMount } from 'svelte';
  import { Toaster } from '@mauzo/ui';
  import { desktop, type ActivateResult, type Cashier, type Session } from './lib/desktop';
  import { cart } from './lib/cart.svelte';
  import { pulse } from './lib/pulse.svelte';
  import { createScanner } from './lib/scanner';
  import { posKeydown } from './lib/keys';
  import TopBar from './lib/components/TopBar.svelte';
  import ActivationScreen from './lib/components/ActivationScreen.svelte';
  import CashierSelect from './lib/components/CashierSelect.svelte';
  import PinPad from './lib/components/PinPad.svelte';
  import CartPanel from './lib/components/CartPanel.svelte';
  import ProductArea from './lib/components/ProductArea.svelte';
  import ActionBar from './lib/components/ActionBar.svelte';
  import Modals from './lib/components/Modals.svelte';
  import LockScreen from './lib/components/LockScreen.svelte';
  import SettingsScreen from './lib/components/SettingsScreen.svelte';

  type Screen = 'boot' | 'browser' | 'activate' | 'login' | 'pin' | 'checkout' | 'settings';

  const bridge = desktop();
  let screen = $state<Screen>('boot');
  let session = $state<Session | null>(null);
  let picking = $state<Cashier | null>(null);
  let pinError = $state('');
  let pinBusy = $state(false);
  let locked = $state(false);
  let method = $state<'cash' | 'card' | 'bank' | 'credit'>('cash');
  let now = $state(new Date());
  let modals = $state<Modals | null>(null);
  let productArea = $state<ProductArea | null>(null);
  let currency = $state('KSh');

  // ---- boot ----------------------------------------------------------------

  onMount(() => {
    const clock = setInterval(() => (now = new Date()), 1000);
    if (!bridge) {
      screen = 'browser';
      return () => clearInterval(clock);
    }
    void boot();
    const poll = setInterval(() => void pulse.refresh(), 15_000);
    const idle = setInterval(checkIdle, 5_000);
    // Heartbeat keeps the back office's online flag fresh; a successful beat
    // is also the trigger to flush queued offline sales.
    const beat = setInterval(() => void pulse.heartbeat(), 60_000);
    const reset = () => (lastActivity = Date.now());
    window.addEventListener('pointerdown', reset);
    window.addEventListener('keydown', reset);
    return () => {
      clearInterval(clock);
      clearInterval(poll);
      clearInterval(idle);
      clearInterval(beat);
      window.removeEventListener('pointerdown', reset);
      window.removeEventListener('keydown', reset);
    };
  });

  async function boot() {
    pulse.info = await bridge!.Info();
    if (!pulse.info.activated) {
      screen = 'activate';
      return;
    }
    await pulse.refresh();
    await pulse.loadLayout();
    session = await bridge!.CurrentSession();
    if (session) {
      screen = 'checkout';
      startScanner();
    } else {
      screen = 'login';
    }
  }

  // ---- auto-lock -------------------------------------------------------------

  let lastActivity = Date.now();
  let lockAfter = 120;
  void bridge?.GetSettings().then((s) => (lockAfter = s.auto_lock_seconds));

  function checkIdle() {
    if (screen === 'checkout' && !locked && Date.now() - lastActivity > lockAfter * 1000) {
      locked = true;
    }
  }

  // ---- activation → login → checkout ------------------------------------------

  function onActivated(r: ActivateResult) {
    pulse.info = {
      version: pulse.info?.version ?? '',
      api_base_url: pulse.info?.api_base_url ?? '',
      activated: true,
      terminal: r.terminal.name,
      branch: r.terminal.branch,
      tenant: r.terminal.tenant,
      business_type: r.terminal.business_type,
    };
    screen = 'login';
  }

  function pickCashier(c: Cashier) {
    picking = c;
    pinError = '';
    screen = 'pin';
  }

  async function login(pin: string) {
    if (!bridge || !picking) return;
    pinBusy = true;
    try {
      session = await bridge.Login(picking.id, pin);
      locked = false;
      screen = 'checkout';
      startScanner();
    } catch (e) {
      pinError = e instanceof Error ? e.message : 'Wrong PIN';
    } finally {
      pinBusy = false;
    }
  }

  async function unlock(pin: string) {
    await login(pin);
  }

  function switchCashier() {
    locked = false;
    session = null;
    picking = null;
    screen = 'login';
  }

  function lock() {
    locked = true;
  }

  // ---- scanner + keys -----------------------------------------------------------

  let scannerOn = false;
  function startScanner() {
    if (scannerOn) return;
    scannerOn = true;
    const s = createScanner((code) => void onScan(code));
    s.attach();
  }

  async function onScan(code: string) {
    if (screen !== 'checkout' || locked) return;
    await productArea?.handleCode(code);
  }

  window.addEventListener('pos:unknown', (e: Event) => {
    modals?.open('unknown', (e as CustomEvent<string>).detail);
  });
  window.addEventListener('pos:held', () => modals?.open('held'));

  function onKeydown(e: KeyboardEvent) {
    if (screen !== 'checkout' || locked) return;
    posKeydown(e, action);
  }

  // ---- cart actions ---------------------------------------------------------------

  const can = (p: string) => session?.permissions.includes(p) ?? false;
  const need = (perm: string, fn: () => void) => (can(perm) ? fn() : modals?.withSupervisor(fn));

  function action(a: string) {
    switch (a) {
      case 'cancel':
        if (cart.count > 0) need('pos.sale.cancel', () => void cart.clear());
        break;
      case 'hold':
        modals?.open('held');
        break;
      case 'pricecheck':
        modals?.open('pricecheck');
        break;
      case 'quantity':
        if (cart.count > 0) modals?.open('quantity');
        break;
      case 'discount':
        if (cart.count > 0) need('pos.discount.apply', () => modals?.open('discount'));
        break;
      case 'more':
        modals?.open('more');
        break;
      case 'pay':
        if (cart.count > 0) modals?.open('pay');
        break;
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if screen === 'boot'}
  <div class="boot">Starting…</div>
{:else if screen === 'browser'}
  <div class="boot">
    <h1>Desktop runtime not detected</h1>
    <p>Run <code>wails dev</code> in <code>backend/desktop</code>, or build the terminal .exe.</p>
  </div>
{:else if screen === 'activate'}
  <ActivationScreen onactivated={onActivated} />
{:else if screen === 'login'}
  <CashierSelect onpick={pickCashier} />
{:else if screen === 'pin' && picking}
  <div class="pinScreen">
    <PinPad
      title="Enter your PIN"
      subtitle={picking.name}
      error={pinError}
      busy={pinBusy}
      onsubmit={login}
      oncancel={switchCashier}
    />
  </div>
{:else if screen === 'settings'}
  <SettingsScreen onback={() => (screen = 'checkout')} />
{:else if screen === 'checkout' && session}
  <div class="pos">
    <TopBar
      info={pulse.info}
      {session}
      status={pulse.status}
      {now}
      onsync={() => void pulse.sync()}
      onsettings={() => (screen = 'settings')}
      onlock={lock}
    />
    <div class="body">
      <CartPanel {currency} />
      <ProductArea bind:this={productArea} {currency} layout={pulse.layout} />
    </div>
    <ActionBar {currency} bind:method onaction={action} />
  </div>
  <Modals
    bind:this={modals}
    {currency}
    {method}
    onpaid={(r) => void pulse.paid(r)}
    onhold={(l) => void pulse.hold(l)}
    onlock={lock}
    onsync={() => void pulse.sync()}
    onswitch={switchCashier}
  />
  {#if locked}
    <LockScreen {session} error={pinError} busy={pinBusy} onunlock={unlock} onswitch={switchCashier} />
  {/if}
{/if}

<Toaster />

<style>
  :global(body) {
    margin: 0;
    height: 100vh;
    overflow: hidden;
    font-family: var(--font-sans, system-ui, sans-serif);
    -webkit-user-select: none;
    user-select: none;
  }
  .boot {
    display: grid;
    place-items: center;
    height: 100vh;
    text-align: center;
    color: var(--color-text-muted, #666);
  }
  .pinScreen {
    display: grid;
    place-items: center;
    height: 100vh;
    background: var(--color-surface-muted, #f4f6fa);
  }
  .pos {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }
  .body {
    flex: 1;
    display: grid;
    grid-template-columns: 400px 1fr;
    min-height: 0;
  }
  @media (max-width: 1100px) {
    .body {
      grid-template-columns: 340px 1fr;
    }
  }
</style>
