<script lang="ts">
  // The activation code, shown once, big enough to read across a counter,
  // with the exact steps to type at the till.
  import type { IssuedTerminalCode } from '@mauzo/types';
  import { Button, Icon, Modal, toast } from '@mauzo/ui';

  interface Props {
    open: boolean;
    terminal: string;
    issued: IssuedTerminalCode | null;
  }
  let { open = $bindable(), terminal, issued }: Props = $props();

  // The till talks to the same API the back office uses.
  const server = (import.meta.env.VITE_API_URL as string | undefined) || window.location.origin;
  const until = $derived(
    issued
      ? new Date(issued.expires_at).toLocaleString('en-KE', {
          weekday: 'short',
          hour: '2-digit',
          minute: '2-digit',
        })
      : '',
  );

  async function copy(text: string, what: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(`${what} copied`);
    } catch {
      toast.error('Copy failed — select and copy it instead');
    }
  }
</script>

<Modal bind:open title={issued?.repair ? `Re-pair ${terminal}` : `Activate ${terminal}`} size="md">
  {#if issued}
    {#if issued.repair}
      <p class="warn">
        <Icon name="alert" size={16} />
        <span>
          This till is already paired. Using this code moves <b>{terminal}</b> to the new computer and signs the
          old one out.
        </span>
      </p>
    {/if}
    <div class="code" aria-label="Activation code">{issued.code}</div>
    <p class="valid">Valid until <b>{until}</b> · works once · not shown again</p>

    <ol>
      <li>Open <b>Mauzo POS</b> on the till computer.</li>
      <li>
        Server address:
        <code>{server}</code>
        <button class="cp" onclick={() => copy(server, 'Server address')} aria-label="Copy server address">
          <Icon name="copy" size={14} />
        </button>
      </li>
      <li>Device name: anything that identifies the computer, e.g. “{terminal} — front counter”.</li>
      <li>Activation code: type the code above (upper or lower case, the dash is optional).</li>
    </ol>
  {/if}
  {#snippet footer()}
    <Button variant="secondary" onclick={() => issued && copy(issued.code, 'Code')}>Copy code</Button>
    <Button onclick={() => (open = false)}>Done</Button>
  {/snippet}
</Modal>

<style>
  .warn {
    display: flex;
    gap: var(--space-2);
    align-items: flex-start;
    margin: 0 0 var(--space-3);
    padding: var(--space-3);
    border-radius: var(--radius-sm);
    background: color-mix(in srgb, var(--color-warning) 14%, transparent);
  }
  .code {
    margin: var(--space-2) 0;
    padding: 22px;
    text-align: center;
    font-family: var(--font-mono, monospace);
    font-size: 2.6rem;
    font-weight: 800;
    letter-spacing: 0.14em;
    color: var(--color-text);
    background: var(--color-surface-muted);
    border: 2px dashed var(--color-border);
    border-radius: var(--radius-lg);
    user-select: all;
  }
  .valid {
    margin: 0 0 var(--space-4);
    text-align: center;
    color: var(--color-text-muted);
  }
  ol {
    margin: 0;
    padding-left: 1.2em;
    display: grid;
    gap: var(--space-2);
  }
  code {
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--color-surface-muted);
  }
  .cp {
    padding: 2px 4px;
    vertical-align: middle;
    background: none;
    border: 0;
    color: var(--color-text-muted);
    cursor: pointer;
  }
</style>
