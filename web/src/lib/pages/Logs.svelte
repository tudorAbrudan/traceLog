<script lang="ts">
  let filter = '';
  let level = 'all';
  let source = 'all';

  const levels = ['all', 'error', 'warn', 'info', 'debug'];
</script>

<div class="logs-page">
  <div class="header">
    <h2>Logs</h2>
    <div class="controls">
      <input type="text" placeholder="Search logs..." bind:value={filter} class="search" />
      <select bind:value={level}>
        {#each levels as l}
          <option value={l}>{l === 'all' ? 'All levels' : l.toUpperCase()}</option>
        {/each}
      </select>
    </div>
  </div>

  <div class="log-viewer">
    <div class="empty">
      <p>Log viewer will show real-time logs here.</p>
      <p class="hint">Configure log sources in Settings → Log Sources</p>
    </div>
  </div>
</div>

<style>
  .logs-page { padding: 1.5rem; }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0; }
  .controls { display: flex; gap: 0.5rem; }
  .search {
    padding: 0.45rem 0.75rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 0.85rem;
    width: 250px;
    outline: none;
  }
  .search:focus { border-color: #58a6ff; }
  select {
    padding: 0.45rem 0.5rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 0.85rem;
  }
  .log-viewer {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    min-height: 400px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .empty { text-align: center; color: var(--text-muted); font-size: 0.9rem; }
  .hint { font-size: 0.8rem; }
</style>
