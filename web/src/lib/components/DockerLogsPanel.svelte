<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  export let serverId = '';
  /** Same values as Logs page — filters loaded container output client-side (keyword heuristics). */
  export let logFilter:
    | 'all'
    | 'critical'
    | 'min_error'
    | 'min_warn'
    | 'min_deprecated'
    | 'min_info'
    | 'min_debug' = 'all';

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

  function fmtBytes(b: number): string {
    if (b == null || !Number.isFinite(b) || b < 0) return '—';
    if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB';
    if (b >= 1048576) return (b / 1048576).toFixed(1) + ' MB';
    if (b >= 1024) return (b / 1024).toFixed(1) + ' KB';
    return b + ' B';
  }

  /** Memory use as % of container cgroup limit (not host RAM %). */
  function memPercentOfLimit(c: { mem_used?: number; mem_limit?: number }): number | null {
    const u = c.mem_used;
    const lim = c.mem_limit;
    if (u == null || lim == null || lim <= 0) return null;
    return (u / lim) * 100;
  }

  function memPctClass(p: number | null): string {
    if (p == null) return '';
    if (p >= 95) return 'mem-hot';
    if (p >= 85) return 'mem-warn';
    return '';
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

  function dockerLineRank(line: string): number {
    const t = line;
    if (/\b(FATAL|CRITICAL|PANIC|EMERG)\b/i.test(t)) return 5;
    if (/\b(ERROR)\b|\] ERROR\b|level=error\b/i.test(t)) return 4;
    if (/\bDEPRECATED\b|deprecated/i.test(t)) return 3;
    if (/\b(WARN|WARNING)\b|\] WARN\b|level=warn/i.test(t)) return 3;
    if (/\b(DEBUG|DBG)\b|\] DEBUG\b|level=debug/i.test(t)) return 1;
    if (/\b(INFO)\b|\] INFO\b|level=info/i.test(t)) return 2;
    return 2;
  }

  function dockerLineIncluded(line: string, f: typeof logFilter): boolean {
    if (f === 'all') return true;
    const r = dockerLineRank(line);
    if (f === 'critical') return r === 5;
    const thr: Record<string, number> = {
      min_error: 4,
      min_warn: 3,
      min_deprecated: 3,
      min_info: 2,
      min_debug: 1,
    };
    const t = thr[f];
    return t !== undefined && r >= t;
  }

  $: dockerDisplayed =
    !dockerLogs || logFilter === 'all'
      ? dockerLogs
      : dockerLogs
          .split('\n')
          .filter((ln) => dockerLineIncluded(ln, logFilter))
          .join('\n');

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
    <h3>Docker containers</h3>
    <span class="hint"
      ><strong>CPU %</strong> is from <code>docker stats</code> (share of the <em>host</em>, not per-container quota).
      <strong>Mem % of limit</strong> is <code>mem_used / mem_limit</code> for that container — a high value here can mean OOM risk even when host RAM looks fine.
      <code>Load logs</code> runs <code>docker logs</code> on the agent host. The Logs page severity menu filters loaded output by keywords (not ingested DB levels).</span>
  </div>
  {#if dockerList.length === 0}
    <p class="docker-empty">No container metrics yet. Enable Docker collection on the agent and wait for the next scrape.</p>
  {:else}
    <div class="docker-stats-wrap">
      <table class="docker-stats-table">
        <thead>
          <tr>
            <th>Container</th>
            <th class="num" title="Host CPU share (docker stats)">CPU %<span class="th-sub">host</span></th>
            <th class="num" title="Used / cgroup limit">Mem %<span class="th-sub">of limit</span></th>
            <th class="num">Memory</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {#each dockerList as c}
            {@const mp = memPercentOfLimit(c)}
            <tr>
              <td class="mono">{c.container_name || c.container_id || '—'}</td>
              <td class="num">{c.cpu_percent != null ? c.cpu_percent.toFixed(1) : '—'}</td>
              <td class="num {memPctClass(mp)}">{mp != null ? mp.toFixed(1) + '%' : '—'}</td>
              <td class="num"
                >{fmtBytes(c.mem_used)} / {c.mem_limit ? fmtBytes(c.mem_limit) : '—'}</td
              >
              <td class="dim">{c.status || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
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
      {#if logFilter !== 'all' && !dockerDisplayed?.trim()}
        <p class="docker-filter-empty">No lines match the current severity filter.</p>
      {:else}
        <pre class="docker-log-out">{dockerDisplayed}</pre>
      {/if}
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
  .docker-stats-wrap {
    overflow-x: auto;
    margin-bottom: 0.75rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-primary);
  }
  .docker-stats-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.75rem;
  }
  .docker-stats-table th,
  .docker-stats-table td {
    padding: 0.35rem 0.5rem;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }
  .docker-stats-table th {
    color: var(--text-muted);
    font-weight: 600;
  }
  .docker-stats-table th.num,
  .docker-stats-table td.num {
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
  .docker-stats-table td.mono {
    font-family: ui-monospace, monospace;
    max-width: 14rem;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .docker-stats-table td.dim {
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .docker-stats-table td.mem-warn {
    color: var(--warning);
    font-weight: 600;
  }
  .docker-stats-table td.mem-hot {
    color: var(--danger);
    font-weight: 700;
  }
  .th-sub {
    display: block;
    font-size: 0.58rem;
    font-weight: 500;
    color: var(--text-muted);
    text-transform: none;
    letter-spacing: 0;
  }
  .docker-stats-table tr:last-child td {
    border-bottom: none;
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
  .docker-filter-empty {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-muted);
  }
</style>
