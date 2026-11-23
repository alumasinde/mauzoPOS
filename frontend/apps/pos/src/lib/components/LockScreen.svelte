<script lang="ts">
  import { Icon } from '@mauzo/ui';
  import PinPad from './PinPad.svelte';
  import type { Session } from '../desktop';

  let {
    session,
    error,
    busy,
    onunlock,
    onswitch,
  }: {
    session: Session;
    error: string;
    busy: boolean;
    onunlock: (pin: string) => void;
    onswitch: () => void;
  } = $props();
</script>

<div class="lock">
  <div class="card">
    <div class="badge"><Icon name="lock" size={30} /></div>
    <PinPad title="Till locked" subtitle="Sign back in, {session.name}" {error} {busy} onsubmit={onunlock} />
    <button class="switch" onclick={onswitch}>Switch cashier</button>
  </div>
</div>

<style>
  .lock {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: var(--color-header);
    display: grid;
    place-items: center;
  }
  .card {
    background: var(--color-surface);
    border-radius: var(--radius-lg);
    padding: var(--space-6);
    text-align: center;
  }
  .badge {
    display: inline-flex;
    color: var(--color-primary);
    background: var(--color-surface-muted);
    border-radius: 50%;
    padding: var(--space-3);
    margin-bottom: var(--space-2);
  }
  .switch {
    margin-top: var(--space-4);
    background: none;
    border: none;
    color: var(--color-primary);
    font: inherit;
    cursor: pointer;
  }
</style>
