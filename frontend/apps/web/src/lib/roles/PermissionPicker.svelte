<script lang="ts">
  import type { PermissionGroup } from '@mauzo/types';
  import { Badge } from '@mauzo/ui';
  import { auth } from '#lib/auth.svelte.ts';

  interface Props {
    groups: PermissionGroup[];
    selected: string[];
    readonly?: boolean;
  }

  let { groups, selected = $bindable(), readonly = false }: Props = $props();

  // You can only grant what you hold (the API enforces this too). Permissions
  // of disabled modules are left to the API, which knows the underlying grant.
  function grantable(code: string, groupEnabled: boolean): boolean {
    return !readonly && (!groupEnabled || auth.can(code));
  }

  function toggle(code: string, on: boolean) {
    selected = on ? [...selected, code] : selected.filter((c) => c !== code);
  }

  function setGroup(g: PermissionGroup, on: boolean) {
    const codes = g.permissions.filter((p) => grantable(p.code, g.enabled)).map((p) => p.code);
    selected = on ? [...new Set([...selected, ...codes])] : selected.filter((c) => !codes.includes(c));
  }
</script>

<div class="groups">
  {#each groups as g (g.key)}
    {@const count = g.permissions.filter((p) => selected.includes(p.code)).length}
    <details open={count > 0}>
      <summary>
        <span class="name">{g.name}</span>
        {#if !g.enabled}<Badge tone="warning" title="Module is disabled for your business">
            module off
          </Badge>{/if}
        <span class="count">{count}/{g.permissions.length}</span>
        {#if !readonly}
          <button type="button" class="link" onclick={() => setGroup(g, count < g.permissions.length)}>
            {count < g.permissions.length ? 'Select all' : 'Clear'}
          </button>
        {/if}
      </summary>
      <ul>
        {#each g.permissions as p (p.code)}
          <li>
            <label class:disabled={!grantable(p.code, g.enabled)}>
              <input
                type="checkbox"
                checked={selected.includes(p.code)}
                disabled={!grantable(p.code, g.enabled)}
                onchange={(e) => toggle(p.code, e.currentTarget.checked)}
              />
              <span>{p.description}</span>
              {#if p.sensitive}<Badge tone="danger" title="Audited; needs supervisor PIN at the POS">
                  sensitive
                </Badge>{/if}
              <code>{p.code}</code>
            </label>
          </li>
        {/each}
      </ul>
    </details>
  {/each}
</div>

<style>
  .groups {
    display: grid;
    gap: var(--space-2);
    max-height: 50vh;
    overflow: auto;
    padding-right: var(--space-1);
  }
  details {
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: var(--space-2) var(--space-3);
  }
  summary {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    cursor: pointer;
    font-weight: 600;
  }
  .name {
    flex: 1;
  }
  .count {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
    font-weight: 400;
  }
  .link {
    border: none;
    background: none;
    color: var(--color-primary);
    cursor: pointer;
    font: inherit;
    font-size: var(--text-sm);
  }
  ul {
    list-style: none;
    margin: var(--space-2) 0 0;
    padding: 0;
    display: grid;
    gap: var(--space-1);
  }
  label {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-sm);
  }
  label.disabled {
    color: var(--color-text-muted);
  }
  code {
    margin-left: auto;
    color: var(--color-text-muted);
    font-size: var(--text-xs);
  }
</style>
