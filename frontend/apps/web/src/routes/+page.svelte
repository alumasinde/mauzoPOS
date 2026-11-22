<script lang="ts">
  import { auth } from '#lib/auth.ts';
</script>

<section class="head">
  <h1>Dashboard</h1>
  <p class="muted">Welcome back, {auth.user?.name}.</p>
</section>

<div class="grid">
  <section class="card">
    <h2>Business</h2>
    <dl>
      <dt>Name</dt>
      <dd>{auth.user?.tenant?.name ?? '—'}</dd>
      <dt>Company code</dt>
      <dd>{auth.user?.tenant?.slug ?? '—'}</dd>
    </dl>
  </section>
  <section class="card">
    <h2>Your account</h2>
    <dl>
      <dt>Role</dt>
      <dd>{auth.user?.role ?? '—'}</dd>
      <dt>Sign-in identity</dt>
      <dd>{auth.user?.identity ?? '—'}</dd>
    </dl>
  </section>
</div>

<style>
  .head h1 {
    margin: 0 0 var(--space-1);
  }
  .muted {
    color: var(--color-text-muted);
  }
  .grid {
    display: grid;
    gap: var(--space-5);
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    margin-top: var(--space-5);
  }
  .card {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    padding: var(--space-5);
    box-shadow: var(--shadow-sm);
  }
  .card h2 {
    margin: 0 0 var(--space-3);
    font-size: var(--text-lg);
  }
  dl {
    margin: 0;
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--space-2) var(--space-4);
  }
  dt {
    color: var(--color-text-muted);
    font-size: var(--text-sm);
  }
  dd {
    margin: 0;
  }
</style>
