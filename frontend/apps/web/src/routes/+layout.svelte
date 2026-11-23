<script lang="ts">
  import '@mauzo/ui/tokens.css';
  import { Toaster } from '@mauzo/ui';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '#lib/auth.svelte.ts';
  import { nav } from '#lib/nav.svelte.ts';

  let { children } = $props();

  // Pages this app ships. Server nav entries for anything else are ignored,
  // so a module can never link to a screen that does not exist yet.
  const ROUTES = ['/', '/onboarding', '/modules', '/settings', '/audit', '/employees', '/roles'] as const;
  type AppRoute = (typeof ROUTES)[number];
  const isRoute = (p: string): p is AppRoute => (ROUTES as readonly string[]).includes(p);

  // The sidebar follows the user's permissions and the tenant's modules.
  $effect(() => {
    if (auth.user) nav.load();
    else nav.clear();
  });

  onMount(() => auth.init());

  $effect(() => {
    if (!auth.ready) return;
    const path = page.url.pathname;
    if (!auth.token && !auth.isPublic(path)) goto(resolve('/login'));
    else if (auth.token && path === '/login') goto(resolve('/'));
  });
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
        <span>{auth.user?.tenant?.name ?? 'Back office'}</span>
      </div>
      <nav>
        {#each nav.items as item (item.key)}
          {#if isRoute(item.path)}
            <a href={resolve(item.path)} class:active={page.url.pathname === item.path}>{item.label}</a>
          {/if}
        {/each}
        <a href={resolve('/account')} class:active={page.url.pathname === '/account'}>Account</a>
      </nav>
    </aside>
    <div class="main">
      <header class="topbar">
        <span></span>
        <div class="who">
          {#if auth.user}
            <span class="name">{auth.user.name}</span>
            <span class="role">{auth.user.role}</span>
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
