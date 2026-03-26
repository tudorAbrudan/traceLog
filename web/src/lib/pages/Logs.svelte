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
  /** What to remove from TraceLog DB (not files on disk). */
  let purgePlan: 'all' | '24h' | '7d' | '30d' = '24h';
  let purgeSource = '';
  let purging = false;

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

  async function purgeStoredLogs() {
    if (!selectedServer) return;
    const diskNote =
      'This only removes data stored inside TraceLog’s database. It does not delete or truncate log files on the server (e.g. under /var/log).';
    const src = purgeSource.trim();
    const srcHint = src ? ` Only source name: "${src}".` : '';

    let ok = false;
    if (purgePlan === 'all') {
      ok = confirm(
        `Delete all ingested log lines for this server from TraceLog?${srcHint}\n\n${diskNote}`,
      );
    } else {
      ok = confirm(
        `Delete ingested log lines older than ${purgePlan} for this server?${srcHint}\n\n${diskNote}`,
      );
    }
    if (!ok) return;

    purging = true;
    try {
      const body: {
        server_id: string;
        mode: 'all' | 'older_than';
        range?: string;
        source?: string;
      } = {
        server_id: selectedServer,
        mode: purgePlan === 'all' ? 'all' : 'older_than',
      };
      if (purgePlan !== 'all') body.range = purgePlan;
      if (src) body.source = src;

      const r = await api.purgeIngestedLogs(body);
      alert(`Removed ${r.deleted ?? 0} stored log row(s).`);
      await fetchLogs();
    } catch (e: any) {
      alert('Purge failed: ' + e.message);
    } finally {
      purging = false;
    }
  }

  $: if (level || selectedServer) fetchLogs();
</script>

<div class="logs-page">
  <div class="header">
    <div class="header-top">
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
    <p class="purge-hint">
      Lines here are a copy in TraceLog’s database. To free space or drop old history, purge below — original files on the server are unchanged.
      <a class="doc-ref" href="https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics" target="_blank" rel="noopener noreferrer">Docs: logs &amp; retention</a>
    </p>
    <div class="purge-bar">
      <span class="purge-label">Remove stored copy:</span>
      <select bind:value={purgePlan} class="purge-select">
        <option value="24h">Older than 24 hours</option>
        <option value="7d">Older than 7 days</option>
        <option value="30d">Older than 30 days</option>
        <option value="all">Everything for this server</option>
      </select>
      <input
        type="text"
        class="purge-source"
        placeholder="Optional: source name (exact match)"
        bind:value={purgeSource}
        title="If set, only rows with this Source field are deleted"
      />
      <button type="button" class="btn-purge" disabled={purging || !selectedServer} on:click={purgeStoredLogs}>
        {purging ? '…' : 'Purge'}
      </button>
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
  .header { display: flex; flex-direction: column; align-items: stretch; gap: 0.5rem; margin-bottom: 1rem; }
  .header-top { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.5rem; }
  .purge-hint { margin: 0; font-size: 0.78rem; color: var(--text-muted); line-height: 1.4; max-width: 720px; }
  .doc-ref { display: inline-block; margin-left: 0.35rem; color: var(--accent); font-size: inherit; }
  .purge-bar {
    display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; padding: 0.5rem 0.65rem;
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 8px;
  }
  .purge-label { font-size: 0.8rem; color: var(--text-secondary); }
  .purge-select, .purge-source {
    padding: 0.35rem 0.5rem; font-size: 0.8rem; border-radius: 6px;
    border: 1px solid var(--border); background: var(--bg-primary); color: var(--text-primary);
  }
  .purge-source { width: 200px; }
  .btn-purge {
    padding: 0.35rem 0.85rem; font-size: 0.8rem; font-weight: 600; border-radius: 6px; cursor: pointer;
    border: 1px solid #f85149; background: transparent; color: #f85149;
  }
  .btn-purge:hover:not(:disabled) { background: rgba(248, 81, 73, 0.12); }
  .btn-purge:disabled { opacity: 0.5; cursor: not-allowed; }
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
