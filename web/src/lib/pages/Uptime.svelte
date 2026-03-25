<script lang="ts">
  let checks: any[] = [];
</script>

<div class="uptime-page">
  <div class="header">
    <h2>Uptime Monitors</h2>
    <button class="btn-add">+ Add Monitor</button>
  </div>

  {#if checks.length === 0}
    <div class="empty">
      <div class="empty-icon">🔍</div>
      <h3>No monitors configured</h3>
      <p>Add HTTP endpoints to monitor uptime, response time, and status codes.</p>
    </div>
  {:else}
    <div class="monitors-list">
      {#each checks as check}
        <div class="monitor-row">
          <span class="indicator online"></span>
          <span class="name">{check.name}</span>
          <span class="url">{check.url}</span>
          <span class="response-time">{check.response_time}ms</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .uptime-page { padding: 1.5rem; }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0; }
  .btn-add {
    padding: 0.5rem 1rem;
    background: #238636;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
  }
  .empty {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
  }
  .empty-icon { font-size: 3rem; margin-bottom: 1rem; }
  .empty h3 { color: var(--text-primary); margin-bottom: 0.5rem; }
  .empty p { font-size: 0.9rem; }
</style>
