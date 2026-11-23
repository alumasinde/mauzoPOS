<script lang="ts">
  import '@mauzo/ui/tokens.css';
  import { Toaster } from '@mauzo/ui';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '#lib/auth.ts';

  let { children } = $props();

  onMount(() => auth.init());

  $effect(() => {
    if (!auth.ready) return;
    const path = page.url.pathname;
    if (!auth.token && !auth.isPublic(path)) goto(resolve('/login'));
    else if (auth.token && path === '/login') goto(resolve('/'));
  });

  const nav = [
    { href: '/', label: 'Dashboard' },
    { href: '/tenants', label: 'Tenants' },
    { href: '/users', label: 'Platform users' },
    { href: '/business-types', label: 'Business types' },
    { href: '/plans', label: 'Plans' },
    { href: '/audit', label: 'Audit log' },
    { href: '/account', label: 'Account' },
  ] as const;
</script>

{#if !auth.ready}
  <div class="boot">Loading…</div>
{:else if auth.isPublic(page.url.pathname)}
  {@render children()}
{:else}
  <div class="shell">
    <aside class="side">
      <div class="brand">
        <strong>Mauzo</strong>
        <span>Platform</span>
      </div>
      <nav>
        {#each nav as item (item.href)}
          <a href={resolve(item.href)} class:active={page.url.pathname === item.href}>{item.label}</a>
        {/each}
      </nav>
    </aside>
    <div class="main">
      <header class="topbar">
        <span></span>
        <div class="who">
          {#if auth.user}
            <span class="name">{auth.user.name}</span>
            <span class="role">{auth.user.role.replace('_', ' ')}</span>
          {/if}
          <button class="out" onclick={() => auth.logout()}>Sign out</button>
        </div>
      </header>
      <main>{@render children()}</main>
    </div>
  </div>
{/if}

<Toaster />

<style>
  .boot {
    display: grid;
    place-items: center;
    min-height: 100vh;
    color: var(--color-text-muted);
  }
  .shell {
    display: flex;
    min-height: 100vh;
  }
  .side {
    width: 220px;
    background: var(--color-header);
    color: var(--color-header-text);
    display: flex;
    flex-direction: column;
    padding: var(--space-4) 0;
  }
  .brand {
    display: flex;
    flex-direction: column;
    padding: 0 var(--space-5) var(--space-4);
  }
  .brand strong {
    font-size: var(--text-xl);
  }
  .brand span {
    font-size: var(--text-xs);
    opacity: 0.7;
  }
  nav {
    display: flex;
    flex-direction: column;
  }
  nav a {
    padding: var(--space-3) var(--space-5);
    color: inherit;
    text-decoration: none;
    opacity: 0.85;
  }
  nav a:hover {
    background: rgb(255 255 255 / 0.08);
  }
  nav a.active {
    background: rgb(255 255 255 / 0.14);
    opacity: 1;
    font-weight: 600;
  }
  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-3) var(--space-5);
    background: var(--color-surface);
    border-bottom: 1px solid var(--color-border);
  }
  .who {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .who .name {
    font-weight: 600;
  }
  .who .role {
    font-size: var(--text-xs);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--color-text-muted);
  }
  .out {
    border: 1px solid var(--color-border);
    background: none;
    border-radius: var(--radius-sm);
    padding: var(--space-1) var(--space-3);
    font: inherit;
    font-size: var(--text-sm);
    cursor: pointer;
    color: var(--color-text);
  }
  .out:hover {
    background: var(--color-surface-muted);
  }
  main {
    padding: var(--space-5);
    max-width: 1100px;
    width: 100%;
    margin: 0 auto;
  }
</style>
