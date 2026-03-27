<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  export let serverId = '';

  let dockerRows: any[] = [];
  let selectedContainer = '';
  let dockerLogs = '';
  let dockerLogsLoading = false;
  let dockerLogsErr = '';

  async function refreshDockerMetrics() {
    if (!serverId) {
      dockerRows = [];
      return;
    }
    try {
      dockerRows = await api.getDockerMetrics(serverId, '1h');
    } catch {
      dockerRows = [];
    }
  }

  async function fetchDockerLogs() {
    if (!selectedContainer) return;
    dockerLogsLoading = true;
    dockerLogsErr = '';
    try {
      const res = await api.getDockerLogs(serverId, selectedContainer, 800);
      dockerLogs = res.logs || '';
    } catch (e: any) {
      dockerLogsErr = e.message || String(e);
      dockerLogs = '';
    } finally {
      dockerLogsLoading = false;
    }
  }

  function uniqueContainers(rows: any[]) {
    const m = new Map<string, any>();
    for (const r of rows) {
      const key = r.container_id || r.container_name || '';
      if (!key) continue;
      const prev = m.get(key);
      if (!prev || new Date(r.ts) > new Date(prev.ts)) m.set(key, r);
    }
    return Array.from(m.values()).sort((a, b) =>
      (a.container_name || '').localeCompare(b.container_name || ''),
    );
  }

  $: dockerList = uniqueContainers(dockerRows);

  $: if (serverId) {
    void refreshDockerMetrics();
  }

  onMount(() => {
    const iv = setInterval(refreshDockerMetrics, 15000);
    return () => clearInterval(iv);
  });
</script>

<div class="docker-logs-panel">
  <div class="card-head docker-head">
    <h3>Docker container logs</h3>
    <span class="hint">Runs <code>docker logs</code> on the machine for this local server (same as the agent host).</span>
  </div>
  {#if dockerList.length === 0}
    <p class="docker-empty">No container metrics yet. Enable Docker collection and wait for the next scrape.</p>
  {:else}
    <div class="docker-toolbar">
      <select bind:value={selectedContainer}>
        <option value="">Select container…</option>
        {#each dockerList as c}
          <option value={c.container_name || c.container_id}>
            {c.container_name || c.container_id} ({c.cpu_percent?.toFixed?.(1) ?? '?'}% CPU)
          </option>
        {/each}
      </select>
      <button class="btn-logs" disabled={!selectedContainer || dockerLogsLoading} on:click={fetchDockerLogs}>
        {dockerLogsLoading ? 'Loading…' : 'Load logs'}
      </button>
    </div>
    {#if dockerLogsErr}
      <pre class="docker-err">{dockerLogsErr}</pre>
    {/if}
    {#if dockerLogs}
      <pre class="docker-log-out">{dockerLogs}</pre>
    {/if}
  {/if}
</div>

<style>
  .docker-logs-panel {
    margin-top: 1rem;
    margin-bottom: 1rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1rem;
  }
  .card-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
    padding: 0 0.25rem;
  }
  .card-head h3 {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin: 0;
    font-weight: 500;
    letter-spacing: 0.02em;
  }
  .docker-head {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.25rem;
    margin-bottom: 0.75rem;
  }
  .docker-head .hint {
    font-size: 0.72rem;
    color: var(--text-muted);
    font-weight: 400;
  }
  .docker-head code {
    font-size: 0.7rem;
    background: var(--bg-hover);
    padding: 1px 4px;
    border-radius: 4px;
  }
  .docker-empty {
    font-size: 0.85rem;
    color: var(--text-muted);
    margin: 0;
  }
  .docker-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
    margin-bottom: 0.5rem;
  }
  .docker-toolbar select {
    flex: 1;
    min-width: 200px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.45rem 0.6rem;
    font-size: 0.85rem;
  }
  .btn-logs {
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: 8px;
    padding: 0.45rem 1rem;
    font-size: 0.8rem;
    font-weight: 600;
    cursor: pointer;
  }
  .btn-logs:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .docker-log-out,
  .docker-err {
    margin: 0;
    max-height: 360px;
    overflow: auto;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.75rem;
    font-size: 0.72rem;
    line-height: 1.45;
    white-space: pre-wrap;
    word-break: break-all;
  }
  .docker-err {
    color: var(--danger);
    border-color: #f8514944;
  }
</style>
