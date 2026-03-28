<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from '../api';
  import { contextServerId, currentPage } from '../store';
  import LoadingState from '../components/LoadingState.svelte';

  let logs: any[] = [];
  let servers: any[] = [];
  let selectedServer = '';
  let filter = '';
  /** Stored log lines in DB: exact critical, or minimum severity (includes more severe). */
  type LogFilter =
    | 'all'
    | 'critical'
    | 'min_error'
    | 'min_warn'
    | 'min_deprecated'
    | 'min_info'
    | 'min_debug';
  let logFilter: LogFilter = 'all';
  let loading = false;
  let loadError = '';
  let autoRefresh = true;
  let intervalId: any;
  let purgeMsg = '';
  /** What to remove from TraceLog DB (not files on disk). */
  let purgePlan: 'all' | '24h' | '7d' | '30d' = '24h';
  let purgeSource = '';
  let purging = false;

  type SortCol = 'ts' | 'level' | 'source' | 'message';
  let sortCol: SortCol = 'ts';
  /** When true, ascending (older → newer for time); when false, descending. */
  let sortAsc = false;
  let colFilterTime = '';
  let colFilterLevel = '';
  let colFilterSource = '';
  let colFilterMessage = '';

  $: uniqueLevels = [...new Set(logs.map((l: any) => l.level).filter(Boolean))].sort(
    (a, b) => (levelRank[a] ?? 99) - (levelRank[b] ?? 99),
  );
  $: uniqueSources = [...new Set(logs.map((l: any) => l.source).filter(Boolean))].sort();

  const levelRank: Record<string, number> = {
    critical: 0,
    error: 1,
    warn: 2,
    deprecated: 3,
    info: 4,
    debug: 5,
  };

  const logFilterOptions: { id: LogFilter; label: string }[] = [
    { id: 'all', label: 'All levels' },
    { id: 'critical', label: 'Critical only' },
    { id: 'min_error', label: 'Error or higher' },
    { id: 'min_warn', label: 'Warning or higher' },
    { id: 'min_deprecated', label: 'Deprecated or higher' },
    { id: 'min_info', label: 'Info or higher' },
    { id: 'min_debug', label: 'Debug or higher' },
  ];

  onMount(() => {
    void (async () => {
      try {
        servers = (await api.listServers()) ?? [];
        if (servers.length > 0) {
          const ctx = get(contextServerId);
          if (ctx && servers.some((s) => s.id === ctx)) {
            selectedServer = ctx;
          } else {
            selectedServer = servers[0].id;
          }
          await fetchLogs();
        }
      } catch (e) {
        loadError = (e as Error).message || 'Failed to load';
      }
    })();

    intervalId = setInterval(() => {
      if (autoRefresh && selectedServer) fetchLogs();
    }, 5000);

    return () => clearInterval(intervalId);
  });

  async function fetchLogs() {
    if (!selectedServer) return;
    loading = true;
    try {
      const q: Parameters<typeof api.getLogs>[1] = {
        search: filter || undefined,
        range: '24h',
      };
      if (logFilter === 'critical') q.level = 'critical';
      else if (logFilter.startsWith('min_')) {
        q.severity_min = logFilter.replace('min_', '') as 'error' | 'warn' | 'deprecated' | 'info' | 'debug';
      }

      const result = await api.getLogs(selectedServer, q);
      logs = result || [];
      // Reset column filters if their value no longer exists in the new data
      const lvlSet = new Set(logs.map((l: any) => l.level).filter(Boolean));
      const srcSet = new Set(logs.map((l: any) => l.source).filter(Boolean));
      if (colFilterLevel && !lvlSet.has(colFilterLevel)) colFilterLevel = '';
      if (colFilterSource && !srcSet.has(colFilterSource)) colFilterSource = '';
    } catch (e) {
      loadError = (e as Error).message || 'Failed to fetch logs';
      logs = [];
    } finally {
      loading = false;
    }
  }

  function levelColor(l: string): string {
    switch (l) {
      case 'critical': return '#da3633';
      case 'error': return '#f85149';
      case 'warn': return '#d29922';
      case 'deprecated': return '#a371f7';
      case 'info': return '#58a6ff';
      case 'debug': return '#8b949e';
      default: return '#e6edf3';
    }
  }

  function formatTime(ts: string): string {
    return new Date(ts).toLocaleTimeString();
  }

  function parseLogTs(entry: { ts?: string }): number {
    const t = entry?.ts;
    if (t == null || t === '') return 0;
    const n = new Date(t as string).getTime();
    return Number.isFinite(n) ? n : 0;
  }

  function levelRankOf(level: string): number {
    const k = (level || '').toLowerCase();
    return levelRank[k] ?? 50;
  }

  function toggleSort(col: SortCol) {
    if (sortCol === col) {
      sortAsc = !sortAsc;
    } else {
      sortCol = col;
      sortAsc = col === 'ts' ? false : true;
    }
  }

  function sortIndicator(col: SortCol): string {
    if (sortCol !== col) return '';
    return sortAsc ? '▲' : '▼';
  }

  function rowMatchesColumnFilters(entry: any): boolean {
    const ft = colFilterTime.trim().toLowerCase();
    if (ft) {
      const disp = formatTime(entry.ts).toLowerCase();
      const raw = String(entry.ts ?? '').toLowerCase();
      if (!disp.includes(ft) && !raw.includes(ft)) return false;
    }
    const fl = colFilterLevel.trim().toLowerCase();
    if (fl && !(String(entry.level ?? '').toLowerCase().includes(fl))) return false;
    const fs = colFilterSource.trim().toLowerCase();
    if (fs && !(String(entry.source ?? '').toLowerCase().includes(fs))) return false;
    const fm = colFilterMessage.trim().toLowerCase();
    if (fm && !(String(entry.message ?? '').toLowerCase().includes(fm))) return false;
    return true;
  }

  function compareRows(a: any, b: any): number {
    let c = 0;
    switch (sortCol) {
      case 'ts': {
        const da = parseLogTs(a);
        const db = parseLogTs(b);
        c = da - db;
        break;
      }
      case 'level': {
        const ra = levelRankOf(a.level);
        const rb = levelRankOf(b.level);
        if (ra !== rb) c = ra - rb;
        else c = String(a.level ?? '').localeCompare(String(b.level ?? ''), undefined, { sensitivity: 'base' });
        break;
      }
      case 'source':
        c = String(a.source ?? '').localeCompare(String(b.source ?? ''), undefined, { sensitivity: 'base' });
        break;
      case 'message':
        c = String(a.message ?? '').localeCompare(String(b.message ?? ''), undefined, { sensitivity: 'base' });
        break;
      default:
        c = 0;
    }
    return sortAsc ? c : -c;
  }

  $: filteredLogs = logs.filter(rowMatchesColumnFilters);
  $: displayLogs = [...filteredLogs].sort(compareRows);

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
      purgeMsg = `Removed ${r.deleted ?? 0} stored log row(s).`;
      await fetchLogs();
    } catch (e: any) {
      purgeMsg = 'Purge failed: ' + (e.message || String(e));
    } finally {
      purging = false;
    }
  }

  $: if (logFilter || selectedServer) fetchLogs();

  $: selectedServerName = servers.find((s) => s.id === selectedServer)?.name ?? 'this server';

  function openServerDockerSection() {
    if (!selectedServer) return;
    contextServerId.set(selectedServer);
    try {
      sessionStorage.setItem('tracelog-focus-docker', '1');
    } catch {
      /* ignore */
    }
    currentPage.set('server:' + selectedServer);
  }
</script>

<div class="logs-page">
  <div class="header">
    <div class="header-top">
    <h2>Logs</h2>
    <div class="controls">
      <select
        bind:value={selectedServer}
        on:change={() => {
          contextServerId.set(selectedServer);
          fetchLogs();
        }}
      >
        {#each servers as s}
          <option value={s.id}>{s.name}</option>
        {/each}
      </select>
      <input type="text" placeholder="Search logs..." bind:value={filter} on:input={fetchLogs} class="search" />
      <select bind:value={logFilter}>
        {#each logFilterOptions as o}
          <option value={o.id}>{o.label}</option>
        {/each}
      </select>
      <label class="auto-label">
        <input type="checkbox" bind:checked={autoRefresh} /> Auto
      </label>
    </div>
    </div>
    <p class="purge-hint">
      Severity applies to <strong>stored</strong> lines (levels set when the agent ingested them). Lines here are a copy in TraceLog’s database. To free space or
      drop old history, purge below — original files on the server are unchanged.
      <a class="doc-ref" href="https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics" target="_blank" rel="noopener noreferrer">Docs: logs &amp; retention</a>
    </p>
    <div class="docker-nav-bar">
      <p class="docker-nav-text">
        <strong>Docker containers</strong> (stats + <code>docker logs</code> in the browser): open the server page and scroll to the Docker section.
      </p>
      <button type="button" class="btn-docker-nav" disabled={!selectedServer} on:click={openServerDockerSection}>
        Open {selectedServerName} → Docker
      </button>
    </div>
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
      {#if purgeMsg}<span class="purge-result" class:purge-err={purgeMsg.startsWith('Purge failed')}>{purgeMsg}</span>{/if}
    </div>
  </div>

  <div class="log-viewer">
    <LoadingState loading={loading && logs.length === 0} error={loadError}>
    {#if logs.length === 0}
      <div class="status-msg">No logs found. Configure log sources in Settings.</div>
    {:else}
      <div class="log-table">
        <div class="log-header-sticky">
          <div class="log-header-row">
            <div class="col-time">
              <button
                type="button"
                class="th-btn"
                aria-label="Sort by time{sortCol === 'ts' ? (sortAsc ? ', ascending' : ', descending') : ''}"
                on:click={() => toggleSort('ts')}
              >
                Time <span class="sort-mark" aria-hidden="true">{sortIndicator('ts')}</span>
              </button>
            </div>
            <div class="col-level">
              <button
                type="button"
                class="th-btn"
                aria-label="Sort by level{sortCol === 'level' ? (sortAsc ? ', ascending' : ', descending') : ''}"
                on:click={() => toggleSort('level')}
              >
                Level <span class="sort-mark" aria-hidden="true">{sortIndicator('level')}</span>
              </button>
            </div>
            <div class="col-source">
              <button
                type="button"
                class="th-btn"
                aria-label="Sort by source{sortCol === 'source' ? (sortAsc ? ', ascending' : ', descending') : ''}"
                on:click={() => toggleSort('source')}
              >
                Source <span class="sort-mark" aria-hidden="true">{sortIndicator('source')}</span>
              </button>
            </div>
            <div class="col-msg">
              <button
                type="button"
                class="th-btn"
                aria-label="Sort by message{sortCol === 'message' ? (sortAsc ? ', ascending' : ', descending') : ''}"
                on:click={() => toggleSort('message')}
              >
                Message <span class="sort-mark" aria-hidden="true">{sortIndicator('message')}</span>
              </button>
            </div>
          </div>
          <div class="log-filter-row">
            <div class="col-time">
              <input
                class="col-filter"
                type="search"
                placeholder="Filter…"
                bind:value={colFilterTime}
                aria-label="Filter by time"
              />
            </div>
            <div class="col-level">
              <select
                class="col-filter col-filter-select"
                bind:value={colFilterLevel}
                aria-label="Filter by level"
              >
                <option value="">All</option>
                {#each uniqueLevels as lvl}
                  <option value={lvl}>{lvl}</option>
                {/each}
              </select>
            </div>
            <div class="col-source">
              <select
                class="col-filter col-filter-select"
                bind:value={colFilterSource}
                aria-label="Filter by source"
              >
                <option value="">All</option>
                {#each uniqueSources as src}
                  <option value={src}>{src}</option>
                {/each}
              </select>
            </div>
            <div class="col-msg">
              <input
                class="col-filter"
                type="search"
                placeholder="Filter message…"
                bind:value={colFilterMessage}
                aria-label="Filter by message"
              />
            </div>
          </div>
        </div>
        {#if displayLogs.length === 0}
          <div class="status-msg filter-empty">No rows match the column filters.</div>
        {:else}
          {#each displayLogs as entry}
            <div class="log-row">
              <span class="col-time">{formatTime(entry.ts)}</span>
              <span class="col-level" style="color: {levelColor(entry.level)}">{entry.level}</span>
              <span class="col-source" title={entry.source}>{entry.source}</span>
              <span class="col-msg">{entry.message}</span>
            </div>
          {/each}
        {/if}
      </div>
    {/if}
    </LoadingState>
  </div>
</div>

<style>
  .logs-page { padding: 1.5rem; }
  .header { display: flex; flex-direction: column; align-items: stretch; gap: 0.5rem; margin-bottom: 1rem; }
  .header-top { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.5rem; }
  .purge-hint { margin: 0; font-size: 0.78rem; color: var(--text-muted); line-height: 1.4; max-width: 720px; }
  .docker-nav-bar {
    display: flex; flex-wrap: wrap; align-items: center; gap: 0.65rem; margin-bottom: 0.75rem;
    padding: 0.55rem 0.75rem; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 8px;
  }
  .docker-nav-text {
    margin: 0; font-size: 0.78rem; color: var(--text-secondary); line-height: 1.4; flex: 1; min-width: 200px;
  }
  .docker-nav-text code { font-size: 0.85em; background: var(--bg-primary); padding: 1px 4px; border-radius: 4px; }
  .btn-docker-nav {
    padding: 0.35rem 0.75rem; font-size: 0.78rem; font-weight: 600; border-radius: 6px; cursor: pointer;
    border: 1px solid var(--accent); background: transparent; color: var(--accent); white-space: nowrap;
  }
  .btn-docker-nav:hover:not(:disabled) { background: rgba(88, 166, 255, 0.12); }
  .btn-docker-nav:disabled { opacity: 0.45; cursor: not-allowed; }
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
  .purge-result { font-size: 0.8rem; color: #3fb950; }
  .purge-err { color: var(--danger); }
  .log-viewer {
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px;
    min-height: 400px; overflow: auto; max-height: calc(100vh - 180px);
  }
  .status-msg { text-align: center; color: var(--text-muted); font-size: 0.9rem; padding: 4rem; }
  .log-table { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.8rem; }
  .log-header-sticky {
    position: sticky; top: 0; z-index: 2;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
  }
  .log-header-row {
    display: flex; gap: 0.75rem; padding: 0.5rem 1rem 0.35rem;
    font-weight: 600; color: var(--text-secondary);
    align-items: center;
  }
  .log-filter-row {
    display: flex; gap: 0.75rem; padding: 0 1rem 0.5rem;
    align-items: center;
  }
  .th-btn {
    display: inline-flex; align-items: center; gap: 0.2rem;
    margin: 0; padding: 0; border: none; background: transparent;
    font: inherit; font-weight: 600; color: inherit; cursor: pointer;
    text-align: left; border-radius: 4px;
  }
  .th-btn:hover { color: var(--text-primary); }
  .th-btn:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
  .sort-mark { font-size: 0.65rem; opacity: 0.85; min-width: 0.75rem; }
  .col-filter {
    width: 100%; min-width: 0; box-sizing: border-box;
    padding: 0.28rem 0.4rem; font-size: 0.72rem; font-family: inherit;
    border: 1px solid var(--border); border-radius: 6px;
    background: var(--bg-primary); color: var(--text-primary);
  }
  .col-filter:focus { outline: none; border-color: #58a6ff; }
  .col-filter::placeholder { color: var(--text-muted); opacity: 0.8; }
  .col-filter-select { cursor: pointer; }
  .log-row {
    display: flex; gap: 0.75rem; padding: 0.35rem 1rem; border-bottom: 1px solid var(--border);
    color: var(--text-primary); transition: background 0.1s;
  }
  .log-row:hover { background: var(--bg-hover); }
  .filter-empty { padding: 2rem 1rem; text-align: center; color: var(--text-muted); font-size: 0.85rem; }
  .col-time { width: 104px; flex-shrink: 0; color: var(--text-muted); }
  .col-level { width: 72px; flex-shrink: 0; font-weight: 600; text-transform: uppercase; font-size: 0.7rem; }
  .col-source { width: 120px; flex-shrink: 0; color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; }
  .col-msg { flex: 1; min-width: 0; word-break: break-all; }
</style>
