<script lang="ts">
  // Optional customer on the sale (loyalty, receipt name); required only for
  // an on-account payment. Hidden behind a link so cash sales never ask.
  import { cart } from '../cart.svelte';
  import CustomerPick from '../components/CustomerPick.svelte';

  let { required = false }: { required?: boolean } = $props();
  let show = $state(false);
</script>

<div class="cust">
  {#if cart.customerID}
    <span>Customer: <b>{cart.customerName}</b></span>
    <button class="link" onclick={() => cart.setCustomer('', '')}>Remove</button>
  {:else if show || required}
    <CustomerPick />
  {:else}
    <button class="link" onclick={() => (show = true)}>+ Add customer (loyalty / receipt name)</button>
  {/if}
</div>

<style>
  .cust {
    display: flex;
    gap: 10px;
    align-items: center;
    flex-wrap: wrap;
  }
  .link {
    padding: 0;
    font: inherit;
    font-weight: 600;
    color: var(--color-primary);
    background: none;
    border: 0;
    cursor: pointer;
  }
</style>
