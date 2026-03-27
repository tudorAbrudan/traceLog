<script lang="ts">
  export let loading = false;
  export let error = '';
</script>

{#if loading}
  <div class="ls-loading">Loading…</div>
{:else if error}
  <div class="ls-error">{error}</div>
{:else}
  <slot />
{/if}

<style>
  .ls-loading {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
  }
  .ls-error {
    color: var(--danger, #f85149);
    font-size: 0.82rem;
    margin: 0.5rem 0;
    padding: 0.4rem 0.75rem;
    background: rgba(248, 81, 73, 0.08);
    border: 1px solid rgba(248, 81, 73, 0.25);
    border-radius: 6px;
  }
</style>
