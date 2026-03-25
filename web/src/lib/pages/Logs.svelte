<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  let logs: any[] = [];
  let servers: any[] = [];
  let selectedServer = '';
  let filter = '';
  let level = 'all';
  let loading = false;
  let autoRefresh = true;
  let intervalId: any;

  const levels = ['all', 'error', 'warn', 'info', 'debug'];

  onMount(async () => {
    try {
      servers = await api.listServers();
      if (servers.length > 0) {
        selectedServer = servers[0].id;
        await fetchLogs();
      }
    } catch {}

    intervalId = setInterval(() => {
      if (autoRefresh && selectedServer) fetchLogs();
    }, 5000);

    return () => clearInterval(intervalId);
  });

  async function fetchLogs() {
    if (!selectedServer) return;
    loading = true;
    try {
      const result = await api.getLogs(selectedServer, {
        level: level !== 'all' ? level : undefined,
        search: filter || undefined,
        range: '24h',
      });
      logs = result || [];
    } catch {
      logs = [];
    } finally {
      loading = false;
    }
  }

  function levelColor(l: string): string {
    switch (l) {
      case 'error': return '#f85149';
      case 'warn': return '#d29922';
      case 'info': return '#58a6ff';
      case 'debug': return '#8b949e';
      default: return '#e6edf3';
    }
  }

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString();
  }

  $: if (level || selectedServer) fetchLogs();
</script>

<div class="logs-page">
  <div class="header">
    <h2>Logs</h2>
    <div class="controls">
      <select bind:value={selectedServer} on:change={fetchLogs}>
        {#each servers as s}
          <option value={s.id}>{s.name}</option>
        {/each}
      </select>
      <input type="text" placeholder="Search logs..." bind:value={filter} on:input={fetchLogs} class="search" />
      <select bind:value={level}>
        {#each levels as l}
          <option value={l}>{l === 'all' ? 'All levels' : l.toUpperCase()}</option>
        {/each}
      </select>
      <label class="auto-label">
        <input type="checkbox" bind:checked={autoRefresh} /> Auto
      </label>
    </div>
  </div>

  <div class="log-viewer">
    {#if loading && logs.length === 0}
      <div class="status-msg">Loading...</div>
    {:else if logs.length === 0}
      <div class="status-msg">No logs found. Configure log sources in Settings.</div>
    {:else}
      <div class="log-table">
        <div class="log-header-row">
          <span class="col-time">Time</span>
          <span class="col-level">Level</span>
          <span class="col-source">Source</span>
          <span class="col-msg">Message</span>
        </div>
        {#each logs as entry}
          <div class="log-row">
            <span class="col-time">{formatTime(entry.ts)}</span>
            <span class="col-level" style="color: {levelColor(entry.level)}">{entry.level}</span>
            <span class="col-source">{entry.source}</span>
            <span class="col-msg">{entry.message}</span>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .logs-page { padding: 1.5rem; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem; }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0; }
  .controls { display: flex; gap: 0.5rem; align-items: center; flex-wrap: wrap; }
  .search {
    padding: 0.45rem 0.75rem; background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 6px; color: var(--text-primary); font-size: 0.85rem; width: 200px; outline: none;
  }
  .search:focus { border-color: #58a6ff; }
  select {
    padding: 0.45rem 0.5rem; background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 6px; color: var(--text-primary); font-size: 0.85rem;
  }
  .auto-label { font-size: 0.8rem; color: var(--text-muted); display: flex; align-items: center; gap: 0.25rem; }
  .log-viewer {
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px;
    min-height: 400px; overflow: auto; max-height: calc(100vh - 180px);
  }
  .status-msg { text-align: center; color: var(--text-muted); font-size: 0.9rem; padding: 4rem; }
  .log-table { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.8rem; }
  .log-header-row {
    display: flex; gap: 0.75rem; padding: 0.6rem 1rem; border-bottom: 1px solid var(--border);
    font-weight: 600; color: var(--text-secondary); position: sticky; top: 0; background: var(--bg-secondary);
  }
  .log-row {
    display: flex; gap: 0.75rem; padding: 0.35rem 1rem; border-bottom: 1px solid var(--border);
    color: var(--text-primary); transition: background 0.1s;
  }
  .log-row:hover { background: var(--bg-hover); }
  .col-time { width: 80px; flex-shrink: 0; color: var(--text-muted); }
  .col-level { width: 50px; flex-shrink: 0; font-weight: 600; text-transform: uppercase; font-size: 0.7rem; }
  .col-source { width: 100px; flex-shrink: 0; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; }
  .col-msg { flex: 1; word-break: break-all; }
</style>
