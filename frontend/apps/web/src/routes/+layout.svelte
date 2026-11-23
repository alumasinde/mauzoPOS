<script lang="ts">
  import '@mauzo/ui/tokens.css';
  import { Toaster } from '@mauzo/ui';
  import { onMount, untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { resolve } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '#lib/auth.svelte.ts';
  import { branch } from '#lib/branch.svelte.ts';
  import { nav } from '#lib/nav.svelte.ts';
  import Sidebar from '#lib/shell/Sidebar.svelte';
  import Topbar from '#lib/shell/Topbar.svelte';

  let { children } = $props();
  let drawer = $state(false);

  // The sidebar follows the user's permissions and the tenant's modules.
  $effect(() => {
    if (auth.user) nav.load();
    else nav.clear();
  });

  // The branch picker follows who is signed in (not every /me refresh).
  const userId = $derived(auth.user?.id ?? null);
  $effect(() => {
    if (!userId) return untrack(() => branch.clear());
    untrack(() => branch.load(auth.user?.all_branches ?? false));
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
    <Sidebar open={drawer} onnavigate={() => (drawer = false)} />
    {#if drawer}
      <button class="scrim" aria-label="Close navigation" onclick={() => (drawer = false)}></button>
    {/if}
    <div class="main">
      <Topbar onmenu={() => (drawer = true)} />
      <main>
        {#if branch.loaded}
          <!-- Re-mount the page when the branch changes so it refetches. -->
          {#key branch.id}
            {@render children()}
          {/key}
        {/if}
      </main>
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
  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  main {
    padding: var(--space-5);
    max-width: 1480px;
    width: 100%;
    margin: 0 auto;
  }
  .scrim {
    display: none;
  }
  @media (max-width: 960px) {
    .scrim {
      display: block;
      position: fixed;
      inset: 0;
      z-index: 35;
      border: 0;
      background: rgb(0 0 0 / 0.45);
    }
  }
  @media (max-width: 640px) {
    main {
      padding: var(--space-3);
    }
  }
</style>
