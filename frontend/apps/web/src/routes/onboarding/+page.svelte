<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError } from '@mauzo/api-client';
  import type { OnboardingState } from '@mauzo/types';
  import { Button, Icon, Modal, PageHeader, TextField, toast } from '@mauzo/ui';
  import { api } from '#lib/api.ts';

  const B = '/api/v1/t/onboarding';
  const STEPS = [
    'Business',
    'Profile',
    'Plan',
    'Modules',
    'Branch',
    'Admin',
    'Preferences',
    'Terminal',
  ] as const;

  let st = $state<OnboardingState | null>(null);
  let loading = $state(true);
  let saving = $state(false);
  let err = $state<string | null>(null);
  let doneOpen = $state(false);

  // step forms
  let biz = $state({ name: '', kra_pin: '', address: '', vat_registered: false });
  let typeCode = $state('');
  let profileId = $state('');
  let planId = $state('');
  let modules = $state<string[]>([]);
  let branch = $state({ code: 'MAIN', name: 'Main Branch', location: '' });
  let admin = $state({ name: '', email: '', password: '' });
  let prefs = $state({ methods: ['cash', 'mpesa'], include_vat: true, rate: '16', width: '80', footer: '' });
  let terminal = $state({ branch_id: '', name: 'Counter 1' });

  const cur = $derived(st ? Math.min(st.step + 1, 8) : 1);
  const profilesOfType = $derived((st?.profiles ?? []).filter((p) => p.business_type === typeCode));
  const chosenPlan = $derived(st?.plans.find((p) => p.id === planId));

  async function load() {
    loading = true;
    try {
      st = (await api.get<OnboardingState>(B)) ?? null;
      if (st) {
        if (st.business_type && !typeCode) typeCode = st.business_type;
        if (st.subscription && !planId) planId = st.subscription.plan_id;
        if (st.modules.length && modules.length === 0) modules = [...st.modules];
      }
    } catch (e) {
      toast.error(e instanceof ApiError ? e.message : 'Failed to load onboarding');
    } finally {
      loading = false;
    }
  }

  async function send(step: number, path: string, payload?: unknown) {
    if (cur < step) return;
    saving = true;
    err = null;
    try {
      const method = path === 'branch' || path === 'terminal' || path === 'complete' ? 'POST' : 'PUT';
      const url = `${B}/${path}`;
      st =
        (method === 'PUT'
          ? await api.put<OnboardingState>(url, payload)
          : await api.post<OnboardingState>(url, payload)) ?? st;
      if (path === 'complete') doneOpen = true;
    } catch (e) {
      err = e instanceof ApiError ? e.message : 'Save failed';
    } finally {
      saving = false;
    }
  }

  function prefsBody() {
    return {
      values: {
        'payments.methods': prefs.methods,
        'tax.prices_include_vat': prefs.include_vat,
        'tax.default_rate': prefs.include_vat ? Number(prefs.rate) : 0,
        'receipt.paper_width': Number(prefs.width),
        ...(prefs.footer ? { 'receipt.footer': prefs.footer } : {}),
      },
    };
  }

  function toggleModule(m: string) {
    modules = modules.includes(m) ? modules.filter((x) => x !== m) : [...modules, m];
  }

  function toggleMethod(m: string) {
    prefs.methods = prefs.methods.includes(m) ? prefs.methods.filter((x) => x !== m) : [...prefs.methods, m];
  }

  onMount(load);
</script>

<div class="obw">
  <PageHeader
    title="Business setup"
    sub="Complete your business setup to start using MauzoPOS"
    icon="store"
  />
  {#if loading}<p>Loading…</p>
  {:else if st}
    {#if st.done}
      <div class="done"><Icon name="check-circle" size={18} /> Setup complete — your business is ready.</div>
    {/if}
    <ol class="steps">
      {#each STEPS as s, i (s)}
        <li class:done={st.step > i} class:cur={cur === i + 1}>
          <span class="dot">
            {#if st.step > i}<Icon name="check" size={15} strokeWidth={3} />{:else}{i + 1}{/if}
          </span>
          <span class="lbl">{s}</span>
        </li>
      {/each}
    </ol>
    <div class="wizard">
      <div class="panel">
        {#if err}<p class="error">{err}</p>{/if}
        {#if cur === 1}
          <h2>Business details</h2>
          <TextField label="Business name" bind:value={biz.name} required />
          <TextField label="KRA PIN (optional)" bind:value={biz.kra_pin} />
          <TextField label="Address" bind:value={biz.address} />
          <label class="check">
            <input type="checkbox" bind:checked={biz.vat_registered} /> Registered for VAT
          </label>
          <Button disabled={saving} onclick={() => send(1, 'business', biz)}>Save &amp; continue</Button>
        {:else if cur === 2}
          <h2>Business type &amp; profile</h2>
          <select bind:value={typeCode}>
            <option value="" disabled>Choose a business type…</option>
            {#each st.business_types as t (t.code)}<option value={t.code}>{t.name}</option>{/each}
          </select>
          {#each profilesOfType as p (p.id)}
            <label class="card">
              <input type="radio" bind:group={profileId} value={p.id} />
              <span>
                <strong>{p.name}</strong> — {p.definition.modules.length} modules,
                {p.definition.categories.length} default categories
              </span>
            </label>
          {/each}
          <Button
            disabled={saving || !profileId}
            onclick={() => send(2, 'profile', { profile_id: profileId })}
          >
            Save &amp; continue
          </Button>
        {:else if cur === 3}
          <h2>Choose a plan</h2>
          {#each st.plans as p (p.id)}
            <label class="card">
              <input type="radio" bind:group={planId} value={p.id} />
              <span>
                <strong>{p.name}</strong> — KES {p.price_kes}/{p.billing_interval}
                ({p.max_branches ?? '∞'} branches · {p.max_users ?? '∞'} users)
              </span>
            </label>
          {/each}
          <Button disabled={saving || !planId} onclick={() => send(3, 'plan', { plan_id: planId })}>
            Save &amp; continue
          </Button>
        {:else if cur === 4}
          <h2>Modules</h2>
          <p class="muted">Included in {chosenPlan?.name ?? 'your plan'}:</p>
          <div class="grid">
            {#each chosenPlan?.modules ?? [] as m (m)}
              <label class="check">
                <input type="checkbox" checked={modules.includes(m)} onchange={() => toggleModule(m)} />
                {m}
              </label>
            {/each}
          </div>
          <Button disabled={saving} onclick={() => send(4, 'modules', { modules })}>
            Save &amp; continue
          </Button>
        {:else if cur === 5}
          <h2>First branch</h2>
          <TextField label="Code" bind:value={branch.code} required />
          <TextField label="Name" bind:value={branch.name} required />
          <TextField label="Location" bind:value={branch.location} />
          {#if st.branches.length}<p class="muted">Added: {st.branches.map((b) => b.name).join(', ')}</p>{/if}
          <Button disabled={saving} onclick={() => send(5, 'branch', { ...branch, is_main: true })}>
            Save &amp; continue
          </Button>
        {:else if cur === 6}
          <h2>Extra admin (optional)</h2>
          <TextField label="Name" bind:value={admin.name} />
          <TextField label="Email" type="email" bind:value={admin.email} />
          <TextField label="Password" type="password" bind:value={admin.password} />
          <Button disabled={saving} onclick={() => send(6, 'admin', admin)}>
            {admin.name ? 'Add &amp; continue' : 'Skip'}
          </Button>
        {:else if cur === 7}
          <h2>Preferences</h2>
          <p class="muted">Payment methods</p>
          <div class="grid">
            {#each ['cash', 'card', 'mpesa', 'bank', 'credit'] as m (m)}
              <label class="check">
                <input type="checkbox" checked={prefs.methods.includes(m)} onchange={() => toggleMethod(m)} />
                {m}
              </label>
            {/each}
          </div>
          <label class="check">
            <input type="checkbox" bind:checked={prefs.include_vat} /> Prices include VAT
          </label>
          {#if prefs.include_vat}
            <label class="field">
              <span>VAT rate</span>
              <select bind:value={prefs.rate}>
                <option value="16">16%</option><option value="0">0%</option>
              </select>
            </label>
          {/if}
          <label class="field">
            <span>Receipt paper width</span>
            <select bind:value={prefs.width}>
              <option value="80">80mm</option><option value="58">58mm</option>
            </select>
          </label>
          <TextField label="Receipt footer" bind:value={prefs.footer} />
          <Button disabled={saving} onclick={() => send(7, 'preferences', prefsBody())}>
            Save &amp; continue
          </Button>
        {:else}
          <h2>POS terminal</h2>
          <label class="field">
            <span>Branch</span>
            <select bind:value={terminal.branch_id}>
              <option value="" disabled>Choose…</option>
              {#each st.branches as b (b.id)}<option value={b.id}>{b.name}</option>{/each}
            </select>
          </label>
          <TextField label="Terminal name" bind:value={terminal.name} required />
          {#if st.terminals.length}<p class="muted">
              Registered: {st.terminals.map((t) => t.name).join(', ')}
            </p>{/if}
          <Button disabled={saving || !terminal.branch_id} onclick={() => send(8, 'terminal', terminal)}>
            Save
          </Button>
          {#if st.terminals.length}
            <Button variant="primary" disabled={saving} onclick={() => send(8, 'complete')}>
              Finish setup
            </Button>
          {/if}
        {/if}
      </div>
      <aside class="side">
        <div class="prog">
          <span class="bigcheck" class:all={st.step >= 8}><Icon name="check-circle" size={40} /></span>
          <h3>Setup {st.step >= 8 ? 'complete' : 'progress'}</h3>
          <p class="frac">{Math.min(st.step, 8)} / 8</p>
          <div class="bar"><i style="width:{(Math.min(st.step, 8) / 8) * 100}%"></i></div>
          {#if st.step < 8}
            <p class="hint">
              {cur === 8
                ? 'Terminal configuration is almost complete.'
                : 'Keep going — each step saves as you go.'}
            </p>
            <p class="next"><span>Next step:</span> {STEPS[Math.min(st.step, 7)]}</p>
          {:else}
            <p class="hint">All steps complete — sign in on the POS to start selling.</p>
          {/if}
        </div>
      </aside>
    </div>
  {/if}
</div>

<Modal bind:open={doneOpen} title="All set">
  <p>Your business is configured and the first terminal is registered. You can now sign in on the POS.</p>
  <Button onclick={() => (doneOpen = false)}>Done</Button>
</Modal>

<style src="./onboarding.css"></style>
