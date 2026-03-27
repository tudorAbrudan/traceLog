<script lang="ts">
  export let error = '';
</script>

{#if error}
  <div class="ed-error">{error}</div>
{/if}

<style>
  .ed-error {
    color: var(--danger, #f85149);
    font-size: 0.82rem;
    margin: 0.5rem 0;
    padding: 0.4rem 0.75rem;
    background: rgba(248, 81, 73, 0.08);
    border: 1px solid rgba(248, 81, 73, 0.25);
    border-radius: 6px;
  }
</style>
