<script lang="ts">
  import '@mauzo/ui/tokens.css';
  import { Toaster } from '@mauzo/ui';

  let { children } = $props();
</script>

<header class="topbar">
  <strong>Mauzo</strong>
  <span>Business Back-office</span>
</header>

<main>
  {@render children()}
</main>

<Toaster />

<style>
  .topbar {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
    padding: var(--space-4) var(--space-5);
    background: var(--color-header);
    color: var(--color-header-text);
  }
  .topbar strong {
    font-size: var(--text-xl);
  }
  main {
    padding: var(--space-5);
    max-width: 1200px;
    margin: 0 auto;
  }
</style>
