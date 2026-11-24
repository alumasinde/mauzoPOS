<script lang="ts">
  // One plain-language status for a terminal row.
  import type { PosTerminal } from '@mauzo/types';
  import { Badge, StatusDot } from '@mauzo/ui';

  let { t }: { t: PosTerminal } = $props();

  const time = (s: string) =>
    new Date(s).toLocaleString('en-KE', { weekday: 'short', hour: '2-digit', minute: '2-digit' });
</script>

{#if t.status === 'disabled'}
  <Badge tone="danger">Disabled</Badge>
{:else if t.status === 'pending'}
  <Badge tone="warning">Not activated</Badge>
  <small>
    {t.code_expires_at
      ? `Code waiting · valid until ${time(t.code_expires_at)}`
      : 'Get a code to pair a till'}
  </small>
{:else}
  <StatusDot status={t.online ? 'ok' : 'unknown'} label={t.online ? 'Online' : 'Offline'} />
  {#if t.code_expires_at}<small>Re-pair code waiting · until {time(t.code_expires_at)}</small>{/if}
{/if}

<style>
  small {
    display: block;
    margin-top: 2px;
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
