<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  let servers: any[] = [];
  let selectedServer = '';
  let range_ = '24h';
  let stats: any = null;
  let recentLogs: any[] = [];
  let loading = true;
  let blacklistText = '';
  let blacklistDirty = false;
  let savingBl = false;
  let badLogs: any[] = [];
  let badLoading = false;
  let badFilterIP = '';

  const ranges = [
    { value: '1h', label: '1H' },
    { value: '6h', label: '6H' },
    { value: '24h', label: '24H' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
  ];

  function whoisHref(ip: string): string {
    return `https://ipwho.is/${encodeURIComponent(ip)}`;
  }

  function ipinfoHref(ip: string): string {
    return `https://ipinfo.io/${encodeURIComponent(ip)}`;
  }

  function isBlacklistedIP(ip: string): boolean {
    return !!(stats?.blacklisted_in_top && stats.blacklisted_in_top.includes(ip));
  }

  onMount(() => {
    let interval: ReturnType<typeof setInterval> | undefined;
    void (async () => {
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
      interval = setInterval(loadData, 30000);
    })();
    return () => {
      if (interval) clearInterval(interval);
    };
  });

  async function loadData() {
    if (!selectedServer) return;
    try {
      const [st, recent, policy] = await Promise.all([
        api.getAccessStats(selectedServer, range_, 50),
        api.getRecentAccessLogs(selectedServer),
        api.getAccessIPPolicy().catch(() => ({ ips: [] as string[] })),
      ]);
      stats = st;
      recentLogs = recent;
      const ips = (policy && policy.ips) || [];
      if (!blacklistDirty) {
        blacklistText = ips.join('\n');
      }
      await refreshBadRequests();
    } catch (e) {
      console.error(e);
    }
  }

  async function saveBlacklist() {
    savingBl = true;
    try {
      const ips = blacklistText
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean);
      await api.putAccessIPPolicy(ips);
      blacklistDirty = false;
      await loadData();
    } catch (e: any) {
      alert('Save failed: ' + (e.message || e));
    } finally {
      savingBl = false;
    }
  }

  async function refreshBadRequests() {
    if (!selectedServer) return;
    badLoading = true;
    try {
      badLogs = await api.getAccessBadRequests(selectedServer, {
        range: range_,
        ip: badFilterIP.trim() || undefined,
        limit: 200,
      });
    } catch {
      badLogs = [];
    } finally {
      badLoading = false;
    }
  }

  function selectRange(r: string) {
    range_ = r;
    loadData();
  }

  function showBadForIP(ip: string) {
    badFilterIP = ip;
    refreshBadRequests();
  }

  function clearBadIPFilter() {
    badFilterIP = '';
    refreshBadRequests();
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
        <select
          bind:value={selectedServer}
          on:change={() => {
            blacklistDirty = false;
            loadData();
          }}
        >
          {#each servers as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
      {/if}
      <div class="range-bar">
        {#each ranges as r}
          <button type="button" class:active={range_ === r.value} on:click={() => selectRange(r.value)}>{r.label}</button>
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
        <span class="stat-value">{stats.unique_ip_count?.toLocaleString?.() ?? '—'}</span>
        <span class="stat-label">Unique IPs</span>
      </div>
      <div class="stat-card">
        <span class="stat-value" class:danger={stats.error_rate > 5}>{stats.error_rate.toFixed(1)}%</span>
        <span class="stat-label">Error rate (4xx/5xx)</span>
      </div>
      <div class="stat-card">
        <span class="stat-value">{stats.avg_duration_ms.toFixed(0)}ms</span>
        <span class="stat-label">Avg response time</span>
      </div>
      <div class="stat-card">
        <span class="stat-value" class:danger={stats.blacklisted_hits > 0}>{stats.blacklisted_hits?.toLocaleString?.() ?? 0}</span>
        <span class="stat-label">Req. from blacklist (est.)</span>
        {#if stats.blacklist_hits_note}
          <span class="stat-sublabel">{stats.blacklist_hits_note}</span>
        {/if}
      </div>
      <div class="stat-card wide">
        <div class="status-breakdown">
          {#each Object.entries(stats.status_codes || {}) as [code, count]}
            <span class="status-chip {code.startsWith('2') ? 'ok' : code.startsWith('4') ? 'warn' : code.startsWith('5') ? 'err' : ''}">
              {code}: {count}
            </span>
          {/each}
        </div>
        <span class="stat-label">Status codes</span>
      </div>
    </div>

    <div class="policy-box">
      <h3>IP blacklist (analytics)</h3>
      <p class="policy-hint">
        One IP or CIDR per line (e.g. <code>203.0.113.50</code> or <code>10.0.0.0/8</code>).
        TraceLog highlights matching traffic and estimates request volume; it does <strong>not</strong> block clients — use nginx,
        firewall, or your CDN for that.
        <a class="doc-ref" href="https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics" target="_blank" rel="noopener noreferrer">Docs</a>
      </p>
      <textarea
        class="policy-ta"
        bind:value={blacklistText}
        on:input={() => (blacklistDirty = true)}
        rows="4"
        placeholder="10.0.0.0/8&#10;192.0.2.1"
      ></textarea>
      <button type="button" class="btn-save-bl" disabled={savingBl} on:click={saveBlacklist}>{savingBl ? 'Saving…' : 'Save blacklist'}</button>
    </div>

    <div class="tables-row three">
      <div class="table-section">
        <h3>Top paths</h3>
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
        <h3>Top method + path</h3>
        <table>
          <thead><tr><th>Method</th><th>Path</th><th class="num">Count</th></tr></thead>
          <tbody>
            {#each stats.top_method_paths || [] as r}
              <tr>
                <td><span class="method">{r.method}</span></td>
                <td class="path">{r.path}</td>
                <td class="num">{r.count}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="table-section">
        <h3>Top IPs</h3>
        <table>
          <thead><tr><th>IP</th><th class="num">Count</th><th>Lookup</th></tr></thead>
          <tbody>
            {#each stats.top_ips || [] as row}
              <tr class:bl-row={isBlacklistedIP(row.ip)}>
                <td class="mono ip-cell">
                  {#if isBlacklistedIP(row.ip)}<span class="bl-badge">list</span>{/if}
                  {row.ip}
                </td>
                <td class="num">{row.count}</td>
                <td class="lookup">
                  <a href={whoisHref(row.ip)} target="_blank" rel="noopener noreferrer">WHOIS</a>
                  <span class="sep">·</span>
                  <a href={ipinfoHref(row.ip)} target="_blank" rel="noopener noreferrer">ipinfo</a>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>

    <div class="bad-section">
      <h3>Bad requests (4xx / 5xx)</h3>
      <p class="policy-hint">
        Per-IP counts in the selected time range. Click a row to load recent bad request lines for that IP below.
      </p>
      <div class="bad-toolbar">
        {#if badFilterIP}
          <span class="bad-filter">Filter: <code>{badFilterIP}</code></span>
          <button type="button" class="btn-clear" on:click={clearBadIPFilter}>Show all bad requests</button>
        {/if}
        <button type="button" class="btn-secondary" on:click={refreshBadRequests} disabled={badLoading}>Refresh list</button>
      </div>
      <div class="bad-tables">
        <table class="bad-by-ip">
          <thead><tr><th>IP</th><th class="num">Bad count</th><th></th></tr></thead>
          <tbody>
            {#each stats.bad_requests_by_ip || [] as row}
              <tr class:bl-row={isBlacklistedIP(row.ip)}>
                <td class="mono">{row.ip}</td>
                <td class="num">{row.bad_count}</td>
                <td>
                  <button type="button" class="linkish" on:click={() => showBadForIP(row.ip)}>Lines</button>
                  <a href={whoisHref(row.ip)} target="_blank" rel="noopener noreferrer" class="linkish">WHOIS</a>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
        <div class="bad-lines-wrap">
          <h4>Recent bad request lines {#if badFilterIP}for {badFilterIP}{/if}</h4>
          {#if badLoading}
            <div class="muted">Loading…</div>
          {:else if badLogs.length === 0}
            <div class="muted">No rows. Pick an IP above or clear the filter.</div>
          {:else}
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
                  {#each badLogs as log}
                    <tr>
                      <td class="mono">{new Date(log.ts).toLocaleString()}</td>
                      <td><span class="method">{log.method}</span></td>
                      <td class="path">{log.path}</td>
                      <td class="num {statusClass(log.status_code)}">{log.status_code}</td>
                      <td class="mono">
                        {log.ip}
                        <a href={whoisHref(log.ip)} target="_blank" rel="noopener noreferrer" class="mini-whois">↗</a>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    </div>

    {#if recentLogs.length > 0}
      <div class="recent-section">
        <h3>Recent requests (all statuses)</h3>
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
                  <td class="mono">
                    {log.ip}
                    <a href={whoisHref(log.ip)} target="_blank" rel="noopener noreferrer" class="mini-whois" title="WHOIS">↗</a>
                  </td>
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
  .analytics { padding: 1.5rem; max-width: 1500px; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.5rem; }
  h2 { margin: 0; font-size: 1.3rem; color: var(--text-primary); }
  h3 { font-size: 0.85rem; color: var(--text-secondary); margin: 0 0 0.5rem; }
  h4 { font-size: 0.8rem; color: var(--text-muted); margin: 0 0 0.5rem; }
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
    display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 0.75rem; margin-bottom: 1rem;
  }
  .stat-card {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 1rem;
  }
  .stat-card.wide { grid-column: span 2; }
  @media (max-width: 700px) {
    .stat-card.wide { grid-column: span 1; }
  }
  .stat-value { display: block; font-size: 1.5rem; font-weight: 700; color: var(--text-primary); line-height: 1.2; }
  .stat-value.danger { color: var(--danger); }
  .stat-label { font-size: 0.72rem; color: var(--text-muted); margin-top: 0.25rem; display: block; }
  .stat-sublabel { font-size: 0.65rem; color: var(--text-muted); display: block; margin-top: 0.2rem; line-height: 1.3; }
  .status-breakdown { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 0.25rem; }
  .status-chip {
    padding: 2px 6px; border-radius: 4px; font-size: 0.72rem; font-weight: 600;
    background: var(--bg-hover); color: var(--text-secondary);
  }
  .status-chip.ok { background: #23863622; color: var(--success); }
  .status-chip.warn { background: #d2992222; color: var(--warning); }
  .status-chip.err { background: #f8514922; color: var(--danger); }

  .policy-box {
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px;
    padding: 0.85rem 1rem; margin-bottom: 1rem;
  }
  .policy-hint { font-size: 0.78rem; color: var(--text-muted); margin: 0 0 0.5rem; line-height: 1.45; }
  .policy-ta {
    width: 100%; min-height: 4.5rem; font-family: monospace; font-size: 0.78rem;
    padding: 0.5rem; border-radius: 8px; border: 1px solid var(--border);
    background: var(--bg-primary); color: var(--text-primary); resize: vertical;
  }
  .btn-save-bl {
    margin-top: 0.5rem; padding: 0.4rem 1rem; font-size: 0.8rem; font-weight: 600;
    border-radius: 8px; border: none; cursor: pointer; background: #238636; color: #fff;
  }
  .btn-save-bl:disabled { opacity: 0.6; cursor: wait; }

  .tables-row {
    display: grid; gap: 0.75rem; margin-bottom: 1.25rem;
  }
  .tables-row.three { grid-template-columns: repeat(3, 1fr); }
  @media (max-width: 1100px) {
    .tables-row.three { grid-template-columns: 1fr; }
  }
  .table-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem; min-width: 0;
  }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th {
    text-align: left; padding: 0.4rem 0.5rem; color: var(--text-muted);
    font-size: 0.72rem; font-weight: 600; border-bottom: 1px solid var(--border);
  }
  th.num, td.num { text-align: right; }
  td { padding: 0.35rem 0.5rem; color: var(--text-secondary); border-bottom: 1px solid var(--border); }
  td.mono { font-variant-numeric: tabular-nums; font-size: 0.75rem; }
  td.path { max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  tr:last-child td { border-bottom: none; }
  tr.bl-row { background: rgba(248, 81, 73, 0.06); }

  .method {
    padding: 1px 5px; border-radius: 3px; font-size: 0.7rem;
    background: var(--bg-hover); color: var(--accent); font-weight: 600;
  }
  .status-2xx { color: var(--success); }
  .status-3xx { color: var(--accent); }
  .status-4xx { color: var(--warning); }
  .status-5xx { color: var(--danger); font-weight: 600; }

  .lookup { font-size: 0.72rem; white-space: nowrap; }
  .lookup a { color: var(--accent); }
  .sep { color: var(--text-muted); margin: 0 0.15rem; }
  .bl-badge {
    display: inline-block; font-size: 0.6rem; font-weight: 700; text-transform: uppercase;
    padding: 1px 4px; border-radius: 3px; background: #f8514933; color: #f85149; margin-right: 4px;
  }
  .ip-cell { vertical-align: middle; }

  .bad-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.85rem; margin-bottom: 1.25rem;
  }
  .bad-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; margin-bottom: 0.65rem; }
  .bad-filter { font-size: 0.78rem; color: var(--text-secondary); }
  .btn-clear, .btn-secondary {
    padding: 0.25rem 0.6rem; font-size: 0.75rem; border-radius: 6px; cursor: pointer;
    border: 1px solid var(--border); background: var(--bg-primary); color: var(--text-primary);
  }
  .btn-secondary:disabled { opacity: 0.5; }
  .bad-tables {
    display: grid; grid-template-columns: minmax(200px, 280px) 1fr; gap: 1rem;
  }
  @media (max-width: 900px) {
    .bad-tables { grid-template-columns: 1fr; }
  }
  .bad-by-ip { font-size: 0.78rem; }
  .linkish {
    background: none; border: none; color: var(--accent); cursor: pointer; font-size: inherit;
    padding: 0; margin-right: 0.5rem; text-decoration: underline;
  }
  a.linkish { text-decoration: underline; }
  .bad-lines-wrap { min-width: 0; }
  .muted { font-size: 0.8rem; color: var(--text-muted); padding: 0.5rem 0; }

  .recent-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem;
  }
  .recent-table { overflow-x: auto; }
  .mini-whois { margin-left: 0.25rem; font-size: 0.7rem; color: var(--accent); text-decoration: none; }
  .mini-whois:hover { text-decoration: underline; }

  .status-msg { text-align: center; padding: 4rem; color: var(--text-muted); }
  code { background: var(--bg-primary); padding: 1px 5px; border-radius: 4px; font-size: 0.85em; }
  .doc-ref { margin-left: 0.35rem; color: var(--accent); font-size: inherit; }
</style>
