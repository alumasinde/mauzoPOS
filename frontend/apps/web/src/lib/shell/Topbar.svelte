<script lang="ts">
  import { resolve } from '$app/paths';
  import { Icon } from '@mauzo/ui';
  import { auth } from '#lib/auth.svelte.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import { onClickOutside } from './outside';

  interface Props {
    onmenu: () => void;
  }
  let { onmenu }: Props = $props();

  let branchOpen = $state(false);
  let userOpen = $state(false);
  let online = $state(typeof navigator === 'undefined' ? true : navigator.onLine);

  const initials = $derived(
    (auth.user?.name ?? '?')
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((w) => w[0]?.toUpperCase())
      .join(''),
  );
  const roleLabel = $derived(auth.user?.role_name || auth.user?.role || '');
  const canSwitch = $derived(branch.list.length > 1);

  function pick(id: string | null) {
    branch.select(id);
    branchOpen = false;
  }
</script>

<svelte:window onoffline={() => (online = false)} ononline={() => (online = true)} />

<header class="topbar">
  <button class="burger" aria-label="Open navigation" onclick={onmenu}><Icon name="menu" size={22} /></button>

  <div class="ctx" use:onClickOutside={() => (branchOpen = false)}>
    <button
      class="store"
      aria-haspopup={canSwitch ? 'listbox' : undefined}
      aria-expanded={canSwitch ? branchOpen : undefined}
      disabled={!canSwitch}
      onclick={() => (branchOpen = !branchOpen)}
    >
      <span class="store-icon"><Icon name="store" size={22} /></span>
      <span class="store-text">
        <strong>{auth.user?.tenant?.name ?? 'Back office'}</strong>
        <span>Branch: {branch.label}</span>
      </span>
      {#if canSwitch}<Icon name="chevron-down" size={18} />{/if}
    </button>
    {#if branchOpen}
      <ul class="menu" role="listbox" aria-label="Branch">
        {#if branch.allowAll}
          <li>
            <button role="option" aria-selected={branch.id === null} onclick={() => pick(null)}>
              All branches
              {#if branch.id === null}<Icon name="check" size={16} />{/if}
            </button>
          </li>
        {/if}
        {#each branch.list as b (b.id)}
          <li>
            <button role="option" aria-selected={branch.id === b.id} onclick={() => pick(b.id)}>
              {b.name}
              {#if b.is_main}<small>Main</small>{/if}
              {#if branch.id === b.id}<Icon name="check" size={16} />{/if}
            </button>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  <div class="right">
    <span class="net" class:off={!online} role="status">
      <span class="dot" aria-hidden="true"></span>{online ? 'Online' : 'Offline'}
    </span>

    <div class="user" use:onClickOutside={() => (userOpen = false)}>
      <button
        class="who"
        aria-haspopup="menu"
        aria-expanded={userOpen}
        onclick={() => (userOpen = !userOpen)}
      >
        <span class="avatar" aria-hidden="true">{initials}</span>
        <span class="who-text">
          <strong>{auth.user?.name}</strong>
          <span>{roleLabel}</span>
        </span>
        <Icon name="chevron-down" size={18} />
      </button>
      {#if userOpen}
        <ul class="menu end" role="menu">
          <li>
            <a role="menuitem" href={resolve('/account')} onclick={() => (userOpen = false)}>
              <Icon name="user" size={16} /> My account
            </a>
          </li>
          <li>
            <button role="menuitem" onclick={() => auth.logout()}>
              <Icon name="logout" size={16} /> Sign out
            </button>
          </li>
        </ul>
      {/if}
    </div>
  </div>
</header>

<style>
  .topbar {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-5);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
    position: sticky;
    top: 0;
    z-index: 20;
  }
  .burger {
    display: none;
    background: none;
    border: 0;
    color: var(--color-text);
    padding: var(--space-1);
    cursor: pointer;
  }
  .ctx,
  .user {
    position: relative;
  }
  .store,
  .who {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
    color: var(--color-text);
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .store {
    border-color: var(--color-border);
    background: var(--color-surface-muted);
    min-width: 0;
  }
  .store:disabled {
    cursor: default;
  }
  .store:not(:disabled):hover,
  .who:hover {
    border-color: var(--color-border);
  }
  .store-icon {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    border-radius: var(--radius-sm);
    background: var(--color-surface);
    color: var(--color-primary);
  }
  .store-text,
  .who-text {
    display: flex;
    flex-direction: column;
    line-height: 1.25;
    min-width: 0;
  }
  .store-text strong,
  .who-text strong {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .store-text span,
  .who-text span {
    font-size: var(--text-xs);
    color: var(--color-text-muted);
  }
  .right {
    margin-left: auto;
    display: flex;
    align-items: center;
    gap: var(--space-4);
  }
  .net {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
    font-weight: 600;
    color: var(--color-success);
  }
  .net .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: currentColor;
  }
  .net.off {
    color: var(--color-danger);
  }
  .avatar {
    display: grid;
    place-items: center;
    width: 38px;
    height: 38px;
    border-radius: 50%;
    background: var(--color-primary);
    color: #fff;
    font-weight: 700;
    font-size: var(--text-sm);
  }
  .menu {
    position: absolute;
    top: calc(100% + 6px);
    left: 0;
    min-width: 220px;
    list-style: none;
    margin: 0;
    padding: var(--space-1);
    background: var(--color-surface-raised);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    z-index: 30;
  }
  .menu.end {
    left: auto;
    right: 0;
  }
  .menu button,
  .menu a {
    width: 100%;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    background: none;
    border: 0;
    border-radius: var(--radius-sm);
    color: var(--color-text);
    font: inherit;
    font-size: var(--text-sm);
    text-decoration: none;
    text-align: left;
    cursor: pointer;
  }
  .menu button:hover,
  .menu a:hover {
    background: var(--color-surface-muted);
  }
  .menu small {
    color: var(--color-text-muted);
  }
  .menu [aria-selected='true'] {
    font-weight: 600;
  }
  .menu [role='option'] :global(svg) {
    margin-left: auto;
    color: var(--color-primary);
  }

  @media (max-width: 960px) {
    .burger {
      display: block;
    }
  }
  @media (max-width: 640px) {
    .topbar {
      padding: var(--space-2) var(--space-3);
      gap: var(--space-2);
    }
    .who-text,
    .store-icon,
    .net {
      display: none;
    }
  }
</style>
