<script lang="ts">
  import { page } from '$app/state';
  import { Icon } from '@mauzo/ui';
  import { nav } from '#lib/nav.svelte.ts';
  import { activeKey, hrefFor, isBuilt } from './routes';

  interface Props {
    /** Mobile drawer state. */
    open: boolean;
    onnavigate: () => void;
  }
  let { open, onnavigate }: Props = $props();

  const KEY = 'mauzo_nav_collapsed';
  const VERSION = import.meta.env.VITE_APP_VERSION ?? 'dev';

  let collapsed = $state<string[]>(readCollapsed());
  const active = $derived(activeKey(nav.items, page.url));
  const activeSection = $derived(nav.items.find((i) => i.key === active)?.section ?? null);

  function readCollapsed(): string[] {
    try {
      return JSON.parse(localStorage.getItem(KEY) ?? '[]') as string[];
    } catch {
      return [];
    }
  }

  function toggle(key: string) {
    collapsed = collapsed.includes(key) ? collapsed.filter((k) => k !== key) : [...collapsed, key];
    try {
      localStorage.setItem(KEY, JSON.stringify(collapsed));
    } catch {
      /* not persisted */
    }
  }

  // The section holding the current page is always open.
  const isOpen = (key: string) => key === activeSection || !collapsed.includes(key);
</script>

<aside class="side" class:open aria-label="Main navigation">
  <div class="brand">
    <span class="logo"><Icon name="cart" size={30} strokeWidth={2} /></span>
    <span class="words">
      <strong>MauzoPOS</strong>
      <span>Retail Made Simple</span>
    </span>
  </div>

  <nav>
    {#each nav.groups as group (group.key)}
      {#if group.key}
        <button
          class="heading"
          aria-expanded={isOpen(group.key)}
          aria-controls="nav-{group.key}"
          onclick={() => toggle(group.key)}
        >
          {group.label}
          <span class="chev" class:shut={!isOpen(group.key)}><Icon name="chevron-down" size={14} /></span>
        </button>
      {/if}
      {#if !group.key || isOpen(group.key)}
        <ul id="nav-{group.key || 'top'}" class:top={!group.key}>
          {#each group.items as item (item.key)}
            <li>
              {#if isBuilt(item.path)}
                <!-- eslint-disable svelte/no-navigation-without-resolve -- hrefFor() resolves it -->
                <a
                  href={hrefFor(item.path)}
                  class:active={active === item.key}
                  aria-current={active === item.key ? 'page' : undefined}
                  onclick={onnavigate}
                >
                  <Icon name={item.icon} size={19} />
                  <span>{item.label}</span>
                </a>
                <!-- eslint-enable svelte/no-navigation-without-resolve -->
              {:else}
                <span class="soon" aria-disabled="true" title="{item.label} is coming soon">
                  <Icon name={item.icon} size={19} />
                  <span>{item.label}</span>
                  <em>Soon</em>
                </span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    {/each}
  </nav>

  <div class="foot">{VERSION === 'dev' ? 'Development build' : `v${VERSION}`}</div>
</aside>

<style>
  .side {
    width: 248px;
    flex: none;
    background: var(--color-sidebar);
    color: var(--color-sidebar-text);
    border-right: 1px solid var(--color-sidebar-border);
    display: flex;
    flex-direction: column;
    height: 100vh;
    position: sticky;
    top: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-5) var(--space-5) var(--space-4);
  }
  .logo {
    color: #f59e0b;
  }
  .words {
    display: flex;
    flex-direction: column;
    line-height: 1.2;
  }
  .words strong {
    color: #fff;
    font-size: var(--text-lg);
    letter-spacing: -0.01em;
  }
  .words span {
    font-size: var(--text-xs);
    color: var(--color-sidebar-text);
  }
  nav {
    flex: 1;
    overflow-y: auto;
    padding: 0 var(--space-3) var(--space-4);
    scrollbar-width: thin;
    scrollbar-color: var(--color-sidebar-border) transparent;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  ul.top {
    margin-bottom: var(--space-2);
  }
  .heading {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: var(--space-4);
    padding: var(--space-1) var(--space-3);
    background: none;
    border: 0;
    border-radius: var(--radius-sm);
    color: var(--color-sidebar-muted);
    font: inherit;
    font-size: 0.6875rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    cursor: pointer;
  }
  .heading:hover {
    color: var(--color-sidebar-text);
  }
  .chev {
    transition: transform 0.15s;
  }
  .chev.shut {
    transform: rotate(-90deg);
  }
  a,
  .soon {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: 7px var(--space-3);
    border-radius: var(--radius-sm);
    color: inherit;
    text-decoration: none;
    font-size: var(--text-sm);
  }
  a:hover {
    background: var(--color-sidebar-hover);
    color: #fff;
  }
  a.active {
    background: var(--color-sidebar-active);
    color: #fff;
    font-weight: 600;
  }
  ul.top a {
    padding: var(--space-3);
    font-size: var(--text-md);
  }
  .soon {
    color: var(--color-sidebar-muted);
    cursor: default;
  }
  a > span,
  .soon > span {
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .soon em {
    margin-left: auto;
    font-style: normal;
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    padding: 1px 6px;
    border-radius: 999px;
    border: 1px solid var(--color-sidebar-border);
  }
  .foot {
    padding: var(--space-3) var(--space-5);
    font-size: var(--text-xs);
    color: var(--color-sidebar-muted);
    border-top: 1px solid var(--color-sidebar-border);
  }

  @media (max-width: 960px) {
    .side {
      position: fixed;
      inset: 0 auto 0 0;
      z-index: 40;
      transform: translateX(-100%);
      transition: transform 0.2s;
    }
    .side.open {
      transform: none;
      box-shadow: var(--shadow-lg);
    }
  }
</style>
