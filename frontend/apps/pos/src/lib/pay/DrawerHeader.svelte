<script lang="ts">
  // Navy title bar of the payment drawer.
  import { Icon } from '@mauzo/ui';
  import { pulse } from '../pulse.svelte';

  let { title, onclose }: { title: string; onclose: () => void } = $props();
</script>

<header>
  <span class="hic"><Icon name="credit-card" size={26} /></span>
  <div>
    <h2 id="pay-title">{title}</h2>
    <p>New sale · {pulse.info?.terminal ?? 'Till'}</p>
  </div>
  <button class="x" onclick={onclose} aria-label="Close"><Icon name="x" size={26} /></button>
</header>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 18px;
    color: #fff;
    background: #0f2347;
  }
  .hic {
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    border: 2px solid rgb(255 255 255 / 0.85);
    border-radius: 10px;
  }
  h2 {
    margin: 0;
    font-size: 1.5rem;
  }
  p {
    margin: 2px 0 0;
    opacity: 0.8;
  }
  .x {
    margin-left: auto;
    display: grid;
    place-items: center;
    width: 48px;
    height: 48px;
    color: #fff;
    background: none;
    border: 0;
    border-radius: 50%;
    cursor: pointer;
  }
  .x:hover {
    background: rgb(255 255 255 / 0.12);
  }
</style>
