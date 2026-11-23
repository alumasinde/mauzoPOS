<script lang="ts">
  import { onMount } from 'svelte';
  import { cart } from '../cart.svelte';
  import { desktop, type Category, type POSLayout, type Product } from '../desktop';
  import { formatMoney } from '../money';

  let { currency = 'KSh', layout = null }: { currency?: string; layout?: POSLayout | null } = $props();

  const gridStyle = $derived(
    layout?.grid_columns ? `grid-template-columns: repeat(${layout.grid_columns}, 1fr);` : '',
  );

  const bridge = desktop();
  let categories = $state<Category[]>([]);
  let active = $state(0); // 0 = all products
  let products = $state<Product[]>([]);
  let query = $state('');
  let searching = $state(false);
  let searchEl: HTMLInputElement | null = $state(null);

  const TILE_ICONS = ['🛒', '🧺', '🥤', '🥛', '🧴', '🍿', '🧼', '🥩', '🍞', '🧃'];

  async function loadCategories() {
    if (!bridge) return;
    categories = await bridge.Categories().catch(() => []);
  }

  async function loadGrid() {
    if (!bridge) return;
    products = await bridge.Grid(active).catch(() => []);
  }

  async function search() {
    if (!bridge) return;
    const q = query.trim();
    if (q.length < 2) {
      searching = false;
      return loadGrid();
    }
    searching = true;
    products = await bridge.SearchProducts(q).catch(() => []);
  }

  // Scans land here via App: barcode → scanLocal → add to cart.
  export async function handleCode(code: string) {
    if (!bridge) return;
    const item = await bridge.Scan(code);
    if (item.unknown) {
      query = code;
      await search();
      if (products.length === 0) {
        window.dispatchEvent(new CustomEvent('pos:unknown', { detail: code }));
      }
      return;
    }
    query = '';
    searching = false;
    await cart.add(item);
    void loadGrid();
  }

  function pick(c: number | null) {
    active = c ?? 0;
    query = '';
    searching = false;
    void loadGrid();
  }

  onMount(() => {
    void loadCategories();
    void loadGrid();
    searchEl?.focus();
  });
</script>

<section class="area">
  <div class="search">
    <span class="bc">▌▍▌▌</span>
    <input
      bind:this={searchEl}
      bind:value={query}
      oninput={search}
      placeholder="Scan barcode or search product…"
      aria-label="Scan or search"
    />
    <button class="find" onclick={search} aria-label="Search">🔍</button>
  </div>

  <div class="tiles">
    {#if layout?.tables_first}
      <button class="tile tables" onclick={() => window.dispatchEvent(new CustomEvent('pos:held'))}>
        <span class="ti">🍽</span><span class="tn">Tables</span>
      </button>
    {/if}
    <button class="tile" class:on={active === 0} onclick={() => pick(null)}>
      <span class="ti">🛒</span><span class="tn">All Products</span>
    </button>
    {#each categories as c, i (c.id)}
      <button class="tile" class:on={active === c.id} onclick={() => pick(c.id)}>
        <span class="ti">{TILE_ICONS[i % TILE_ICONS.length]}</span>
        <span class="tn">{c.name}</span>
        <span class="tc">{c.count}</span>
      </button>
    {/each}
  </div>

  <div class="grid" style={gridStyle}>
    {#each products as p (p.variant_id ? `v${p.variant_id}` : p.id)}
      <button class="prod" onclick={() => cart.addProduct(p)}>
        <span class="img">
          {p.name[0]}{#if layout?.weighted_items && p.sell_by === 'weight'}<i class="kg">⚖</i>{/if}
        </span>
        <span class="pn">
          {p.name}{#if p.variant_name}
            · {p.variant_name}{/if}
        </span>
        <span class="pp">{currency} {formatMoney(p.price)}{p.sell_by === 'weight' ? '/kg' : ''}</span>
      </button>
    {:else}
      <p class="empty">
        {searching ? `No products match “${query}”.` : 'No products — sync from the server.'}
      </p>
    {/each}
  </div>
</section>

<style>
  .area {
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: var(--color-surface-muted);
  }
  .search {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin: var(--space-3);
    padding: 0 var(--space-3);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    height: 48px;
  }
  .bc {
    font-size: 1.4rem;
    letter-spacing: -1px;
    color: var(--color-text-muted);
  }
  input {
    flex: 1;
    border: none;
    outline: none;
    font: inherit;
    font-size: var(--text-lg);
    background: none;
  }
  .find {
    border: none;
    background: none;
    font-size: 1.2rem;
    cursor: pointer;
  }
  .tiles {
    display: flex;
    gap: var(--space-2);
    padding: 0 var(--space-3) var(--space-2);
    overflow-x: auto;
    flex-shrink: 0;
  }
  .tile {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-width: 108px;
    height: 76px;
    border: none;
    border-radius: var(--radius-md);
    background: var(--color-header);
    color: var(--color-header-text);
    cursor: pointer;
    position: relative;
  }
  .tile.on {
    background: var(--color-primary);
  }
  .tile.tables {
    background: var(--color-warning, #b45309);
  }
  .kg {
    position: absolute;
    right: -2px;
    bottom: -2px;
    font-style: normal;
    font-size: 0.9rem;
    background: var(--color-surface);
    border-radius: 50%;
    padding: 1px 3px;
  }
  .ti {
    font-size: 1.5rem;
  }
  .tn {
    font-size: var(--text-sm);
    font-weight: 600;
  }
  .tc {
    position: absolute;
    top: 4px;
    right: 6px;
    font-size: 0.65rem;
    background: rgb(255 255 255 / 0.2);
    border-radius: 8px;
    padding: 0 5px;
  }
  .grid {
    flex: 1;
    overflow-y: auto;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: var(--space-3);
    padding: var(--space-3);
    align-content: start;
  }
  .prod {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-1);
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-3);
    cursor: pointer;
    min-height: 118px;
    justify-content: center;
  }
  .prod:hover {
    border-color: var(--color-primary);
    box-shadow: 0 2px 8px rgb(0 0 0 / 0.08);
  }
  .prod:active {
    transform: scale(0.97);
  }
  .img {
    width: 56px;
    height: 56px;
    border-radius: var(--radius-sm);
    display: grid;
    place-items: center;
    position: relative;
    font-size: 1.6rem;
    font-weight: 700;
    background: var(--color-surface-muted);
    color: var(--color-primary);
  }
  .pn {
    font-weight: 600;
    font-size: var(--text-sm);
    text-align: center;
    line-height: 1.2;
  }
  .pp {
    font-size: var(--text-sm);
    color: var(--color-text-muted);
    font-variant-numeric: tabular-nums;
  }
  .empty {
    grid-column: 1 / -1;
    color: var(--color-text-muted);
    padding: var(--space-6);
    text-align: center;
  }
</style>
