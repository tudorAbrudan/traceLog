<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  let servers: any[] = [];
  let selectedServer = '';
  let range_ = '24h';
  let stats: any = null;
  let recentLogs: any[] = [];
  let loading = true;

  const ranges = [
    { value: '1h', label: '1H' },
    { value: '6h', label: '6H' },
    { value: '24h', label: '24H' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
  ];

  onMount(async () => {
    try {
      servers = await api.listServers();
      if (servers.length > 0) {
        selectedServer = servers[0].id;
        await loadData();
      }
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }

    const interval = setInterval(loadData, 30000);
    return () => clearInterval(interval);
  });

  async function loadData() {
    if (!selectedServer) return;
    try {
      [stats, recentLogs] = await Promise.all([
        api.getAccessStats(selectedServer, range_),
        api.getRecentAccessLogs(selectedServer),
      ]);
    } catch (e) {
      console.error(e);
    }
  }

  function selectRange(r: string) {
    range_ = r;
    loadData();
  }

  function statusClass(code: number): string {
    if (code >= 500) return 'status-5xx';
    if (code >= 400) return 'status-4xx';
    if (code >= 300) return 'status-3xx';
    return 'status-2xx';
  }
</script>

<div class="analytics">
  <div class="header">
    <h2>HTTP Analytics</h2>
    <div class="controls">
      {#if servers.length > 1}
        <select bind:value={selectedServer} on:change={loadData}>
          {#each servers as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
      {/if}
      <div class="range-bar">
        {#each ranges as r}
          <button class:active={range_ === r.value} on:click={() => selectRange(r.value)}>{r.label}</button>
        {/each}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="status-msg">Loading analytics...</div>
  {:else if !stats || stats.total_requests === 0}
    <div class="status-msg">No HTTP request data yet. Configure nginx access log monitoring in Settings to see data.</div>
  {:else}
    <div class="stats-grid">
      <div class="stat-card">
        <span class="stat-value">{stats.total_requests.toLocaleString()}</span>
        <span class="stat-label">Total Requests</span>
      </div>
      <div class="stat-card">
        <span class="stat-value" class:danger={stats.error_rate > 5}>{stats.error_rate.toFixed(1)}%</span>
        <span class="stat-label">Error Rate</span>
      </div>
      <div class="stat-card">
        <span class="stat-value">{stats.avg_duration_ms.toFixed(0)}ms</span>
        <span class="stat-label">Avg Response Time</span>
      </div>
      <div class="stat-card">
        <div class="status-breakdown">
          {#each Object.entries(stats.status_codes || {}) as [code, count]}
            <span class="status-chip {code.startsWith('2') ? 'ok' : code.startsWith('4') ? 'warn' : code.startsWith('5') ? 'err' : ''}">
              {code}: {count}
            </span>
          {/each}
        </div>
        <span class="stat-label">Status Codes</span>
      </div>
    </div>

    <div class="tables-row">
      <div class="table-section">
        <h3>Top Paths</h3>
        <table>
          <thead><tr><th>Path</th><th class="num">Count</th></tr></thead>
          <tbody>
            {#each stats.top_paths || [] as p}
              <tr><td class="path">{p.path}</td><td class="num">{p.count}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="table-section">
        <h3>Top IPs</h3>
        <table>
          <thead><tr><th>IP</th><th class="num">Count</th></tr></thead>
          <tbody>
            {#each stats.top_ips || [] as ip}
              <tr><td class="mono">{ip.ip}</td><td class="num">{ip.count}</td></tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    {#if recentLogs.length > 0}
      <div class="recent-section">
        <h3>Recent Requests</h3>
        <div class="recent-table">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Method</th>
                <th>Path</th>
                <th class="num">Status</th>
                <th>IP</th>
              </tr>
            </thead>
            <tbody>
              {#each recentLogs.slice(0, 50) as log}
                <tr>
                  <td class="mono">{new Date(log.ts).toLocaleTimeString()}</td>
                  <td><span class="method">{log.method}</span></td>
                  <td class="path">{log.path}</td>
                  <td class="num {statusClass(log.status_code)}">{log.status_code}</td>
                  <td class="mono">{log.ip}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .analytics { padding: 1.5rem; max-width: 1400px; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem; }
  h2 { margin: 0; font-size: 1.3rem; color: var(--text-primary); }
  h3 { font-size: 0.85rem; color: var(--text-secondary); margin: 0 0 0.5rem; }
  .controls { display: flex; align-items: center; gap: 0.75rem; }

  select {
    background: var(--bg-secondary); color: var(--text-primary);
    border: 1px solid var(--border); border-radius: 6px;
    padding: 0.35rem 0.6rem; font-size: 0.8rem;
  }

  .range-bar {
    display: inline-flex; gap: 2px; background: var(--bg-secondary);
    padding: 3px; border-radius: 8px; border: 1px solid var(--border);
  }
  .range-bar button {
    padding: 0.3rem 0.7rem; background: none; border: none; color: var(--text-muted);
    border-radius: 6px; cursor: pointer; font-size: 0.75rem; font-weight: 600;
  }
  .range-bar button:hover { color: var(--text-primary); }
  .range-bar button.active { background: var(--accent); color: #fff; }

  .stats-grid {
    display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 0.75rem; margin-bottom: 1.25rem;
  }
  .stat-card {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 1rem;
  }
  .stat-value { display: block; font-size: 1.6rem; font-weight: 700; color: var(--text-primary); line-height: 1.2; }
  .stat-value.danger { color: var(--danger); }
  .stat-label { font-size: 0.72rem; color: var(--text-muted); margin-top: 0.25rem; display: block; }
  .status-breakdown { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 0.25rem; }
  .status-chip {
    padding: 2px 6px; border-radius: 4px; font-size: 0.72rem; font-weight: 600;
    background: var(--bg-hover); color: var(--text-secondary);
  }
  .status-chip.ok { background: #23863622; color: var(--success); }
  .status-chip.warn { background: #d2992222; color: var(--warning); }
  .status-chip.err { background: #f8514922; color: var(--danger); }

  .tables-row {
    display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; margin-bottom: 1.25rem;
  }
  .table-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem;
  }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th {
    text-align: left; padding: 0.4rem 0.5rem; color: var(--text-muted);
    font-size: 0.72rem; font-weight: 600; border-bottom: 1px solid var(--border);
  }
  th.num, td.num { text-align: right; }
  td { padding: 0.35rem 0.5rem; color: var(--text-secondary); border-bottom: 1px solid var(--border); }
  td.mono { font-variant-numeric: tabular-nums; font-size: 0.75rem; }
  td.path { max-width: 250px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  tr:last-child td { border-bottom: none; }

  .method {
    padding: 1px 5px; border-radius: 3px; font-size: 0.7rem;
    background: var(--bg-hover); color: var(--accent); font-weight: 600;
  }
  .status-2xx { color: var(--success); }
  .status-3xx { color: var(--accent); }
  .status-4xx { color: var(--warning); }
  .status-5xx { color: var(--danger); font-weight: 600; }

  .recent-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem;
  }
  .recent-table { overflow-x: auto; }
  .status-msg { text-align: center; padding: 4rem; color: var(--text-muted); }

  @media (max-width: 900px) {
    .tables-row { grid-template-columns: 1fr; }
  }
</style>
