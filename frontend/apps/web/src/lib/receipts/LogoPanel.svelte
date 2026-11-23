<script lang="ts">
  // Upload the receipt logo. The API converts it to black-and-white dots
  // (what a thermal head can print); the preview shows that exact result.
  import { ApiError } from '@mauzo/api-client';
  import { Button, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';
  import { auth } from '#lib/auth.svelte.ts';

  let { logoUrl, onchanged }: { logoUrl: string | null; onchanged: () => void } = $props();

  let busy = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  async function upload(file: File) {
    if (file.size > 2 * 1024 * 1024) {
      toast.error('The image must be 2 MB or smaller');
      return;
    }
    busy = true;
    try {
      const fd = new FormData();
      fd.append('file', file);
      const base = import.meta.env.VITE_API_URL ?? '';
      const res = await fetch(`${base}/api/v1/t/receipts/logo`, {
        method: 'PUT',
        body: fd,
        headers: auth.token ? { Authorization: `Bearer ${auth.token}` } : {},
      });
      if (!res.ok) {
        const body = (await res.json().catch(() => null)) as { error?: { message?: string } } | null;
        throw new Error(body?.error?.message ?? 'Upload failed');
      }
      toast.success('Logo updated');
      onchanged();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Upload failed');
    } finally {
      busy = false;
      if (input) input.value = '';
    }
  }

  async function remove() {
    if (!confirm('Remove the receipt logo?')) return;
    busy = true;
    try {
      await api.del('/api/v1/t/receipts/logo');
      toast.success('Logo removed');
      onchanged();
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Could not remove');
    } finally {
      busy = false;
    }
  }
</script>

<div class="panel">
  <div class="stage">
    {#if logoUrl}
      <img src={logoUrl} alt="Receipt logo as printed" />
    {:else}
      <span>No logo yet</span>
    {/if}
  </div>
  <div class="side">
    <p>
      Use a simple, high-contrast image (PNG or JPEG, up to 2 MB). It is resized to fit a 58 mm roll and
      converted to black dots, as shown here.
    </p>
    <input
      type="file"
      accept="image/png,image/jpeg,image/gif"
      bind:this={input}
      hidden
      onchange={(e) => e.currentTarget.files?.[0] && upload(e.currentTarget.files[0])}
    />
    <div class="row">
      <Button onclick={() => input?.click()} loading={busy}>
        {logoUrl ? 'Replace logo' : 'Upload logo'}
      </Button>
      {#if logoUrl}<Button variant="ghost" onclick={remove} disabled={busy}>Remove</Button>{/if}
    </div>
  </div>
</div>

<style>
  .panel {
    display: grid;
    grid-template-columns: minmax(200px, 300px) 1fr;
    gap: var(--space-5);
    align-items: start;
  }
  .stage {
    display: grid;
    place-items: center;
    min-height: 140px;
    padding: var(--space-4);
    background: #fffef9;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    color: #777;
  }
  .stage img {
    max-width: 100%;
    image-rendering: pixelated;
  }
  .side p {
    margin: 0 0 var(--space-3);
    color: var(--color-text-muted);
  }
  .row {
    display: flex;
    gap: var(--space-2);
  }
  @media (max-width: 640px) {
    .panel {
      grid-template-columns: 1fr;
    }
  }
</style>
