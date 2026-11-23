<script lang="ts">
  import type { ModuleInfo } from '@mauzo/types';
  import { Badge, Button, Switch } from '@mauzo/ui';

  interface Props {
    module: ModuleInfo;
    names: Record<string, string>;
    businessType?: string | null;
    busy: boolean;
    ontoggle: (m: ModuleInfo, on: boolean) => void;
    onconfigure: (m: ModuleInfo) => void;
  }

  let { module: m, names, businessType, busy, ontoggle, onconfigure }: Props = $props();

  const REASONS: Record<string, string> = {
    platform_inactive: 'Temporarily unavailable on the platform',
    not_in_plan: 'Not included in your plan',
    dependency_off: 'A module it needs is off',
  };
  const recommended = $derived(!!businessType && m.recommended_for.includes(businessType));
  const locked = $derived(m.core || (!m.enabled && (m.in_plan === false || m.platform_active === false)));
</script>

<li class="row" class:off={!m.effective}>
  <div class="info">
    <div class="title">
      <strong>{m.name}</strong>
      {#if m.core}<Badge tone="info">Core</Badge>{/if}
      {#if recommended}<Badge tone="success">Recommended</Badge>{/if}
      {#if m.source === 'platform'}<Badge title="Enabled by Mauzo support">Platform</Badge>{/if}
      {#if m.in_plan === false}<Badge tone="warning">Upgrade plan</Badge>{/if}
    </div>
    <p>{m.description}</p>
    {#if m.depends.length}
      <p class="deps">Needs: {m.depends.map((d) => names[d] ?? d).join(', ')}</p>
    {/if}
    {#if m.enabled && !m.effective && m.reason}
      <p class="warn">{REASONS[m.reason]}</p>
    {/if}
  </div>
  <div class="ctl">
    {#if m.has_config && m.effective}
      <Button size="sm" variant="ghost" onclick={() => onconfigure(m)}>Settings</Button>
    {/if}
    <Switch
      checked={!!m.enabled}
      label={`${m.enabled ? 'Disable' : 'Enable'} ${m.name}`}
      disabled={locked}
      {busy}
      onchange={(on) => ontoggle(m, on)}
    />
  </div>
</li>

<style>
  .row {
    display: flex;
    gap: var(--space-4);
    align-items: center;
    padding: var(--space-3) var(--space-4);
    border-bottom: 1px solid var(--color-border);
  }
  .row:last-child {
    border-bottom: none;
  }
  .info {
    flex: 1;
    min-width: 0;
  }
  .title {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    flex-wrap: wrap;
  }
  p {
    margin: var(--space-1) 0 0;
    font-size: var(--text-sm);
    color: var(--color-text-muted);
  }
  .deps {
    font-size: var(--text-xs);
  }
  .warn {
    color: #9a6700;
  }
  .off .title strong {
    color: var(--color-text-muted);
  }
  .ctl {
    display: flex;
    gap: var(--space-3);
    align-items: center;
  }
</style>
