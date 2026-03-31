<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { get } from 'svelte/store';
  import uPlot from 'uplot';
  import 'uplot/dist/uPlot.min.css';
  import { api } from '../api';
  import { contextServerId } from '../store';
  import { fmtBytes } from '../utils/format';
  import LoadingState from '../components/LoadingState.svelte';

  // --- State ---
  let servers: any[] = [];
  let selectedServer = '';
  let range_ = '24h';
  let stats: any = null;
  let timeline: { points: Array<{ ts: string; count: number; avg_duration_ms: number }>; bucket_minutes: number } | null = null;
  let recentLogs: any[] = [];
  let loading = true;
  let loadError = '';
  let blacklistText = '';
  let blacklistDirty = false;
  let savingBl = false;
  let saveError = '';
  let badLogs: any[] = [];
  let badLoading = false;
  let badFilterIP = '';
  let slowLogs: any[] = [];
  let slowLoading = false;
  let slowMinMs = 500;

  let activeTab: 'overview' | 'paths' | 'clients' | 'requests' = 'overview';

  let pathsLoaded = false;
  let clientsLoaded = false;
  let pathsLoading = false;
  let clientsLoading = false;

  // Debounce range changes to avoid rapid API calls
  let rangeDebounceTimer: ReturnType<typeof setTimeout> | undefined;

  const ranges = [
    { value: '1h', label: '1H' },
    { value: '6h', label: '6H' },
    { value: '24h', label: '24H' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
  ];

  // --- Timeline chart ---
  let chartEl: HTMLDivElement | undefined;
  let uplot: uPlot | null = null;
  let chartPending: ReturnType<typeof setTimeout> | null = null;

  function buildTimelineChart() {
    if (chartPending) { clearTimeout(chartPending); chartPending = null; }
    if (uplot) { uplot.destroy(); uplot = null; }
    if (!chartEl || !timeline?.points?.length) return;

    const w = chartEl.clientWidth;
    if (w < 50) {
      chartPending = setTimeout(buildTimelineChart, 100);
      return;
    }

    const xs = timeline.points.map((p) => new Date(p.ts).getTime() / 1000);
    const ys = timeline.points.map((p) => p.count);
    const ms = timeline.points.map((p) => p.avg_duration_ms);

    uplot = new uPlot(
      {
        width: w,
        height: 180,
        padding: [8, 8, 0, 0],
        series: [
          {},
          {
            label: 'Requests',
            stroke: '#4dabf7',
            fill: '#4dabf722',
            width: 1.5,
            points: { show: false },
            scale: 'req',
          },
          {
            label: 'Timp mediu (ms)',
            stroke: '#a78bfa',
            width: 1.5,
            points: { show: false },
            scale: 'ms',
          },
        ],
        axes: [
          {
            stroke: '#8b949e88',
            ticks: { stroke: '#8b949e22', width: 1 },
            grid: { stroke: '#8b949e11', width: 1 },
            font: '10px system-ui, sans-serif',
            gap: 4,
            size: 36,
          },
          {
            scale: 'req',
            stroke: '#4dabf7aa',
            ticks: { stroke: '#4dabf722', width: 1 },
            grid: { stroke: '#8b949e15', width: 1 },
            font: '10px system-ui, sans-serif',
            gap: 4,
            size: 50,
          },
          {
            scale: 'ms',
            side: 1,
            stroke: '#a78bfaaa',
            ticks: { stroke: '#a78bfa22', width: 1 },
            grid: { show: false },
            font: '10px system-ui, sans-serif',
            gap: 4,
            size: 50,
          },
        ],
        cursor: { show: true, x: true, y: false },
        legend: { show: false },
        scales: { x: { time: true }, req: {}, ms: {} },
      },
      [xs, ys, ms],
      chartEl,
    );
  }

  function scheduleBuildChart() {
    if (chartPending) clearTimeout(chartPending);
    chartPending = setTimeout(() => {
      chartPending = null;
      buildTimelineChart();
    }, 50);
  }

  let chartResizeObs: ResizeObserver | null = null;

  // --- Threat scoring ---
  const SCANNER_PATHS = [
    '/wp-admin', '/wp-login', '/xmlrpc.php', '/wp-content', '/wp-includes',
    '/.env', '/.git/', '/.htaccess', '/.svn/',
    '/config.php', '/configuration.php', '/settings.php',
    '/phpmyadmin', '/pma/', '/admin/login',
    '/setup.php', '/install.php', '/installer.php',
    '/actuator/', '/api/swagger', '/swagger-ui',
    '/cgi-bin/', '/shell', '/webshell',
    '/etc/passwd', '/etc/shadow',
    '../', '%2e%2e', '..%2f',
    '/composer.json', '/package.json', '/.DS_Store',
    '/db.sql', '/backup.sql', '/database.sql',
  ];

  const SCANNER_UAS = [
    'zgrab', 'masscan', 'nuclei', 'nikto', 'sqlmap',
    'dirbuster', 'gobuster', 'wfuzz', 'ffuf',
    'libwww-perl', 'python-requests', 'python-urllib',
    'go-http-client',
  ];

  function getSubnet24(ip: string): string {
    const parts = ip.split('.');
    return parts.length === 4 ? parts.slice(0, 3).join('.') : '';
  }

  $: subnetCounts = (() => {
    const map: Record<string, number> = {};
    for (const ip of (stats?.top_ips ?? [])) {
      const s = getSubnet24(ip.ip);
      if (s) map[s] = (map[s] || 0) + 1;
    }
    return map;
  })();

  // Build a lookup map: ip → bad_count from bad_requests_by_ip
  $: badCountByIP = (() => {
    const m: Record<string, number> = {};
    for (const b of (stats?.bad_requests_by_ip ?? [])) m[b.ip] = b.bad_count;
    return m;
  })();

  function scoredIP(ipStat: any): { score: number; badges: string[]; errRate: number; errCount: number } {
    let score = 0;
    const badges: string[] = [];

    // Use bad_requests_by_ip for accurate error count (top_ips.error_count field not present)
    const errCount = badCountByIP[ipStat.ip] ?? 0;
    const errRate = ipStat.count > 0 ? errCount / ipStat.count : 0;
    if (errRate > 0.8) score += 4;
    else if (errRate > 0.5) score += 2;

    const ipReqs = [...recentLogs, ...badLogs].filter((r) => r.ip === ipStat.ip);

    const hasScannerPath = ipReqs.some(
      (r) => r.path && SCANNER_PATHS.some((p) => r.path.toLowerCase().includes(p)),
    );
    if (hasScannerPath) { score += 4; badges.push('SCANNER'); }

    const hasBotUA = ipReqs.some(
      (r) => r.user_agent && SCANNER_UAS.some((ua) => r.user_agent.toLowerCase().includes(ua)),
    );
    if (hasBotUA) { score += 3; badges.push('BOT'); }

    const subnet = getSubnet24(ipStat.ip);
    if (subnet && (subnetCounts[subnet] ?? 0) >= 2) {
      score += 2; badges.push('SUBNET');
    }

    return { score, badges, errRate, errCount };
  }

  $: scoredIPs = (stats?.top_ips ?? [])
    .map((ip: any) => ({ ...ip, ...scoredIP(ip) }))
    .sort((a: any, b: any) => b.score - a.score);

  $: recommendedToBlock = (scoredIPs as any[]).filter(
    (r) => r.score >= 3 && !isBlacklistedIP(r.ip),
  );

  // Fetch threat assessments for recommended IPs
  $: if (recommendedToBlock.length > 0) {
    void (async () => {
      try {
        const ips = recommendedToBlock.map((r: any) => r.ip);
        const trafficScores = Object.fromEntries(recommendedToBlock.map((r: any) => [r.ip, r.score]));
        const res = await api.getThreatAssessments(ips, trafficScores);
        if (res?.assessments) {
          const assessmentMap = new Map(res.assessments.map((a: any) => [a.ip, a]));
          recommendedToBlock = recommendedToBlock.map((ip: any) => ({
            ...ip,
            assessment: assessmentMap.get(ip.ip),
            ipinfo: (assessmentMap.get(ip.ip) as any)?.ipinfo,
          }));
        }
      } catch (e) {
        console.warn('Failed to fetch threat assessments:', e);
      }
    })();
  }

  function addToBlacklist(ip: string) {
    const lines = blacklistText.trim() ? blacklistText.trim().split('\n').map((s: string) => s.trim()) : [];
    if (!lines.includes(ip)) {
      blacklistText = [...lines, ip].join('\n');
      blacklistDirty = true;
    }
  }

  function addAllRecommended() {
    const existing = new Set(
      blacklistText.trim().split('\n').map((s: string) => s.trim()).filter(Boolean),
    );
    const toAdd = (recommendedToBlock as any[]).map((r) => r.ip).filter((ip: string) => !existing.has(ip));
    if (!toAdd.length) return;
    blacklistText = [...existing, ...toAdd].join('\n');
    blacklistDirty = true;
  }

  // Rebuild chart when tab or data changes
  $: if (activeTab === 'overview' && timeline && chartEl) {
    scheduleBuildChart();
  }

  // Lazy load Requests tab data when tab is activated
  $: if (activeTab === 'requests') {
    void loadTabData();
  }

  $: if (activeTab === 'paths' && stats && !pathsLoaded) void loadPathsData();
  $: if (activeTab === 'clients' && stats && !clientsLoaded) void loadClientsData();

  // --- Helpers ---
  function whoisHref(ip: string): string {
    return `https://ipwho.is/${encodeURIComponent(ip)}`;
  }

  function ipinfoHref(ip: string): string {
    return `https://ipinfo.io/${encodeURIComponent(ip)}`;
  }

  function isBlacklistedIP(ip: string): boolean {
    return !!(stats?.blacklisted_in_top && stats.blacklisted_in_top.includes(ip));
  }

  /** Non-empty, non-comment lines (same rules as save). */
  function parseBlacklistLines(raw: string): string[] {
    return raw
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0 && !s.startsWith('#'));
  }

  function nginxDenySnippetFromLines(lines: string[]): string {
    const header = `# TraceLog — nginx deny directives (NOT applied by TraceLog)
# Add this block inside http { }, server { }, or location { } on the host that terminates TLS.
# Test before reload: sudo nginx -t && sudo systemctl reload nginx
# Behind CDN/proxy: set real_ip / trusted headers so the address you deny matches logged clients.
`;
    const body = lines.map((l) => `deny ${l};`).join('\n');
    return `${header}\n${body}\n`;
  }

  async function copyNginxDenySnippet() {
    const lines = parseBlacklistLines(blacklistText);
    if (lines.length === 0) {
      alert('Add at least one IP or CIDR line (one per line), then try again.');
      return;
    }
    const snip = nginxDenySnippetFromLines(lines);
    try {
      await navigator.clipboard.writeText(snip);
      alert('Nginx snippet copied to clipboard. Paste into your server config, test with nginx -t, then reload.');
    } catch {
      window.prompt('Copy this snippet (Ctrl+C):', snip);
    }
  }

  function downloadNginxDenySnippet() {
    const lines = parseBlacklistLines(blacklistText);
    if (lines.length === 0) {
      alert('Add at least one IP or CIDR line (one per line), then try again.');
      return;
    }
    const snip = nginxDenySnippetFromLines(lines);
    const blob = new Blob([snip], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'tracelog-ip-deny-snippet.conf';
    a.click();
    URL.revokeObjectURL(url);
  }

  function statusClass(code: number): string {
    if (code >= 500) return 'status-5xx';
    if (code >= 400) return 'status-4xx';
    if (code >= 300) return 'status-3xx';
    return 'status-2xx';
  }

  // --- Data loading ---
  async function refreshBadRequests() {
    if (!selectedServer) return;
    badLoading = true;
    try {
      badLogs = (await api.getAccessBadRequests(selectedServer, {
        range: range_,
        ip: badFilterIP.trim() || undefined,
        limit: 200,
      })) ?? [];
    } catch {
      badLogs = [];
    } finally {
      badLoading = false;
    }
  }

  async function refreshSlowRequests() {
    if (!selectedServer) return;
    slowLoading = true;
    try {
      const min = Math.max(1, Math.min(3_600_000, Number(slowMinMs) || 500));
      slowLogs = (await api.getAccessSlowRequests(selectedServer, {
        range: range_,
        min_ms: min,
        limit: 200,
      })) ?? [];
    } catch {
      slowLogs = [];
    } finally {
      slowLoading = false;
    }
  }

  async function loadData() {
    if (!selectedServer) return;
    loading = true;
    loadError = '';
    pathsLoaded = false;
    clientsLoaded = false;
    try {
      const [s, tl, logs, policy] = await Promise.all([
        api.getAccessStats(selectedServer, range_, 20, 'overview'),
        api.getAccessTimeline(selectedServer, range_).catch(() => null),
        api.getRecentAccessLogs(selectedServer),
        api.getAccessIPPolicy().catch(() => ({ ips: [] })),
      ]);
      stats = s;
      if (tl) timeline = tl as typeof timeline;
      recentLogs = logs ?? [];
      if (!blacklistDirty) blacklistText = (policy as any).ips?.join('\n') ?? '';
    } catch (e: any) {
      loadError = e?.message ?? 'Failed to load data';
    } finally {
      loading = false;
    }
    if (activeTab === 'requests') {
      void refreshBadRequests();
      void refreshSlowRequests();
    }
  }

  async function loadPathsData() {
    if (pathsLoaded || pathsLoading || !selectedServer) return;
    pathsLoading = true;
    try {
      const s = await api.getAccessStats(selectedServer, range_, 20, 'paths');
      if (s && stats) {
        stats = { ...stats, top_paths: s.top_paths, top_method_paths: s.top_method_paths, top_paths_by_duration: s.top_paths_by_duration };
      }
      pathsLoaded = true;
    } catch (_) { /* silent */ } finally {
      pathsLoading = false;
    }
  }

  async function loadClientsData() {
    if (clientsLoaded || clientsLoading || !selectedServer) return;
    clientsLoading = true;
    try {
      const s = await api.getAccessStats(selectedServer, range_, 20, 'clients');
      if (s && stats) {
        stats = { ...stats, top_ips: s.top_ips, bad_requests_by_ip: s.bad_requests_by_ip };
      }
      clientsLoaded = true;
    } catch (_) { /* silent */ } finally {
      clientsLoading = false;
    }
  }

  async function loadTabData() {
    // Lazy load tab-specific data only when tab is activated
    if (activeTab === 'requests' && badLogs.length === 0 && !badLoading && !slowLoading) {
      await refreshBadRequests();
      await refreshSlowRequests();
    }
  }

  async function saveBlacklist() {
    savingBl = true;
    saveError = '';
    try {
      const ips = blacklistText
        .split('\n')
        .map((s) => s.trim())
        .filter(Boolean);
      await api.putAccessIPPolicy(ips);
      blacklistDirty = false;
      await loadData();
    } catch (e: any) {
      saveError = e.message || 'Save failed';
    } finally {
      savingBl = false;
    }
  }

  function selectRange(r: string) {
    range_ = r;
    // Debounce range changes (wait 300ms for rapid clicks before loading)
    if (rangeDebounceTimer) clearTimeout(rangeDebounceTimer);
    rangeDebounceTimer = setTimeout(() => {
      void loadData();
      rangeDebounceTimer = undefined;
    }, 300);
  }

  function showBadForIP(ip: string) {
    badFilterIP = ip;
    activeTab = 'requests';
    void refreshBadRequests();
  }

  function clearBadIPFilter() {
    badFilterIP = '';
    void refreshBadRequests();
  }

  onMount(() => {
    let interval: ReturnType<typeof setInterval> | undefined;
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
          await loadData();
        }
      } catch (e) {
        loadError = (e as Error).message || 'Failed to load';
      } finally {
        loading = false;
      }
      interval = setInterval(() => { void loadData(); }, 30000);
    })();

    chartResizeObs = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const w = Math.floor(entry.contentRect.width);
        if (w < 50) continue;
        if (uplot) {
          uplot.setSize({ width: w, height: 180 });
        } else if (timeline?.points?.length) {
          scheduleBuildChart();
        }
      }
    });

    return () => {
      if (interval) clearInterval(interval);
    };
  });

  onDestroy(() => {
    if (chartPending) clearTimeout(chartPending);
    chartResizeObs?.disconnect();
    if (uplot) { uplot.destroy(); uplot = null; }
  });

  // Attach resize observer once chartEl is bound and we switch to overview
  $: if (chartEl && chartResizeObs) {
    chartResizeObs.observe(chartEl);
  }
</script>

<div class="analytics">
  <!-- Top controls: always visible -->
  <div class="header">
    <h2>HTTP Analytics</h2>
    <div class="controls">
      {#if servers.length > 1}
        <select
          bind:value={selectedServer}
          on:change={() => {
            contextServerId.set(selectedServer);
            blacklistDirty = false;
            void loadData();
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
      {#if loading}
        <span class="range-loading">Se incarca...</span>
      {:else if stats}
        <span class="range-loaded">Date: {range_}</span>
      {/if}
    </div>
  </div>

  <LoadingState {loading} error={loadError}>
    {#if !stats || stats.total_requests === 0}
      <div class="status-msg">
        No HTTP request data in this range. Add your nginx (or apache) <strong>access</strong> log under Settings &rarr; Log Sources with format <strong>nginx</strong> (or apache), then <strong>restart TraceLog</strong> so the agent tails the file. Only requests logged <em>after</em> the restart are ingested; try hitting your site to generate new lines.
      </div>
    {:else}
      <!-- Tab bar -->
      <div class="tabs">
        <button type="button" class="tab-btn" class:active={activeTab === 'overview'} on:click={() => (activeTab = 'overview')}>Overview</button>
        <button type="button" class="tab-btn" class:active={activeTab === 'paths'} on:click={() => (activeTab = 'paths')}>Paths</button>
        <button type="button" class="tab-btn" class:active={activeTab === 'clients'} on:click={() => (activeTab = 'clients')}>Clients</button>
        <button type="button" class="tab-btn" class:active={activeTab === 'requests'} on:click={() => (activeTab = 'requests')}>Requests</button>
      </div>

      <!-- Tab: Overview -->
      {#if activeTab === 'overview'}
        <div class="page-intro">
          <strong>IP rankings</strong> are on this page (sidebar: <strong>HTTP Analytics</strong>). Use the time range above.
          <ul>
            <li><strong>Top client IPs</strong> — clients ranked by <em>total</em> request count (Clients tab, up to 50 rows). <strong>Unique IPs</strong> in the summary is how many distinct addresses appear in the range.</li>
            <li><strong>Top IPs by error responses</strong> — clients ranked by how many <strong>4xx / 5xx</strong> responses they received. Use <strong>Lines</strong> to see sample failing requests for one IP.</li>
            <li>
              <strong>Self-traffic</strong> — rankings can ignore chosen <strong>User-Agent</strong> substrings (e.g. TraceLog's uptime client) under <strong>Settings &rarr; General</strong>, so tables reflect your app's visitors rather than monitoring probes.
            </li>
            <li>
              <strong>Slow requests</strong> — lists ingested access lines whose <strong>request time</strong> (ms from the log) is at least your threshold; sorted slowest first. Requires your access log format to include duration.
            </li>
          </ul>
        </div>

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
            <span class="stat-label">Req. from IP list (est., analytics only)</span>
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

        <!-- Traffic timeline chart -->
        <div class="timeline-chart-wrap table-section">
          <div class="chart-header">
            <h3 class="section-label">Trafic HTTP</h3>
            <div class="chart-legend">
              <span class="legend-dot" style="background:#4dabf7"></span><span>Requests</span>
              <span class="legend-dot" style="background:#a78bfa"></span><span>Timp mediu (ms)</span>
            </div>
          </div>
          {#if !timeline?.points?.length}
            <div class="muted">No timeline data available for this range.</div>
          {:else}
            <div class="timeline-chart" bind:this={chartEl}></div>
          {/if}
        </div>
      {/if}

      <!-- Tab: Paths -->
      {#if activeTab === 'paths'}
        {#if pathsLoading}
          <div class="tab-loading">Incarca datele pentru {range_}...</div>
        {/if}
        <div class="two-col">
          <div class="table-section">
            <h3>Top URL paths</h3>
            <p class="section-lead">Most requested paths in this range.</p>
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
            <h3>Top paths by avg duration</h3>
            <p class="section-lead">Paths ranked by mean response time (ms).</p>
            {#if !(stats.top_paths_by_duration ?? []).length}
              <div class="muted">No duration data available (requires access log format with timing).</div>
            {:else}
              <table>
                <thead><tr><th>Path</th><th class="num">Avg ms</th><th class="num">Count</th></tr></thead>
                <tbody>
                  {#each stats.top_paths_by_duration ?? [] as p}
                    <tr>
                      <td class="path">{p.path}</td>
                      <td class="num slow-ms">{Math.round(p.avg_duration_ms ?? 0)}</td>
                      <td class="num">{p.count}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            {/if}
          </div>
        </div>

        <div class="table-section" style="margin-top: 0.75rem">
          <h3>Top method + path</h3>
          <p class="section-lead">Combined HTTP method and path by volume.</p>
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
      {/if}

      <!-- Tab: Clients -->
      {#if activeTab === 'clients'}
        {#if clientsLoading}
          <div class="tab-loading">Incarca datele pentru {range_}...</div>
        {/if}
        {#if recommendedToBlock.length > 0}
          <div class="recommend-box">
            <div class="recommend-header">
              <div>
                <h3 class="recommend-title">Recommended to block &mdash; {recommendedToBlock.length} IP{recommendedToBlock.length === 1 ? '' : 's'}</h3>
                <p class="section-lead">Automatic detection based on error rate, scanner paths, bot User-Agents, and subnet clustering. Review before blocking &mdash; legitimate services (monitoring, CDN) can score high with many requests.</p>
              </div>
              <div class="recommend-actions">
                {#if recommendedToBlock.length > 1}
                  <button type="button" class="btn-add-all" on:click={addAllRecommended}>Add all to IP list</button>
                {/if}
              </div>
            </div>
            <table>
              <thead>
                <tr>
                  <th>IP</th>
                  <th class="num">Requests</th>
                  <th class="num">Errors</th>
                  <th class="num">Err%</th>
                  <th>Threat</th>
                  <th>Why</th>
                  <th>Country / Risk</th>
                  <th>Decision</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {#each recommendedToBlock as row}
                  <tr>
                    <td class="mono">{row.ip}</td>
                    <td class="num">{row.count}</td>
                    <td class="num">{row.errCount}</td>
                    <td class="num">{row.count > 0 ? (row.errRate * 100).toFixed(0) : 0}%</td>
                    <td>
                      {#if row.score >= 6}
                        <span class="badge-threat">THREAT</span>
                      {:else}
                        <span class="badge-suspicious">SUSPICIOUS</span>
                      {/if}
                    </td>
                    <td>
                      {#each row.badges as badge}
                        {#if badge === 'SCANNER'}
                          <span class="badge-scanner">{badge}</span>
                        {:else if badge === 'BOT'}
                          <span class="badge-bot">{badge}</span>
                        {:else if badge === 'SUBNET'}
                          <span class="badge-subnet">{badge}</span>
                        {/if}
                      {/each}
                    </td>
                    <td class="ipinfo-cell">
                      <span class="country">{row.ipinfo?.country || '-'}</span>
                      {#if row.ipinfo?.abuse_confidence}
                        <span class="abuse-badge" class:abuse-high={row.ipinfo.abuse_confidence > 50}>
                          {row.ipinfo.abuse_confidence.toFixed(0)}% abuse
                        </span>
                      {/if}
                    </td>
                    <td>
                      <span class="decision" class:decision-block={row.assessment?.decision === 'block'} class:decision-monitor={row.assessment?.decision === 'monitor'} class:decision-allow={row.assessment?.decision === 'allow'}>
                        {row.assessment?.decision?.toUpperCase() || '?'}
                      </span>
                    </td>
                    <td>
                      <button type="button" class="btn-add-one" on:click={() => addToBlacklist(row.ip)}>Add to list</button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
            <p class="recommend-note">Adding to list only highlights in analytics. Use <strong>Save list</strong> below, then export to nginx / firewall to actually block.</p>
          </div>
        {/if}

        <div class="table-section" style="margin-bottom: 0.75rem">
          <h3>Top client IPs</h3>
          <p class="section-lead">Ranking by total requests with automatic threat scoring. Up to 50 rows.</p>
          <table>
            <thead>
              <tr>
                <th>IP address</th>
                <th class="num">Requests</th>
                <th class="num">Errors</th>
                <th class="num">Err%</th>
                <th class="num">Bytes</th>
                <th>Threat</th>
                <th>Badges</th>
                <th>Links</th>
              </tr>
            </thead>
            <tbody>
              {#each scoredIPs as row}
                <tr class:bl-row={isBlacklistedIP(row.ip)}>
                  <td class="mono ip-cell">
                    {#if isBlacklistedIP(row.ip)}<span class="bl-badge">list</span>{/if}
                    {row.ip}
                  </td>
                  <td class="num">{row.count}</td>
                  <td class="num">{row.errCount}</td>
                  <td class="num">{row.count > 0 ? (row.errRate * 100).toFixed(0) : 0}%</td>
                  <td class="num">{fmtBytes(row.bytes_sent ?? 0)}</td>
                  <td>
                    {#if row.score >= 6}
                      <span class="badge-threat">THREAT</span>
                    {:else if row.score >= 3}
                      <span class="badge-suspicious">SUSPICIOUS</span>
                    {/if}
                  </td>
                  <td>
                    {#each row.badges as badge}
                      {#if badge === 'SCANNER'}
                        <span class="badge-scanner">{badge}</span>
                      {:else if badge === 'BOT'}
                        <span class="badge-bot">{badge}</span>
                      {:else if badge === 'SUBNET'}
                        <span class="badge-subnet">{badge}</span>
                      {/if}
                    {/each}
                  </td>
                  <td class="lookup">
                    <a href={whoisHref(row.ip)} target="_blank" rel="noopener noreferrer">WHOIS</a>
                    <span class="sep">&middot;</span>
                    <a href={ipinfoHref(row.ip)} target="_blank" rel="noopener noreferrer">ipinfo</a>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- IP Blacklist section -->
        <div class="policy-box">
          <h3>IP list &mdash; analytics &amp; nginx export</h3>
          <p class="policy-hint">
            <strong>What TraceLog does with this list</strong> (after you click <em>Save</em>): it stores the lines in the hub database
            and uses them only inside <strong>HTTP Analytics</strong> &mdash; to <strong>highlight</strong> matching client IPs in the tables
            and to show the <strong>"Req. from blacklist (est.)"</strong> counter. Data is whatever was already ingested from your access
            logs; nothing is changed on disk and <strong>no requests are refused</strong> by TraceLog.
          </p>
          <ul class="policy-list">
            <li>
              <strong>Not a firewall:</strong> visitors can still reach your site. To actually
              <strong>block</strong> addresses, configure <strong>nginx</strong> <code>deny</code>, a <strong>firewall</strong>, or your
              <strong>CDN</strong> / WAF.
            </li>
            <li>
              <strong>Export for nginx:</strong> use the buttons below to generate <code>deny &hellip;;</code> lines from the textarea (saved or
              not). You edit and reload nginx yourself &mdash; TraceLog does not modify the server.
            </li>
            <li>
              <strong>Why this lives on HTTP Analytics:</strong> the list is meant to be tuned while you look at <strong>top IPs</strong> and
              error traffic on the same page; blocking itself always happens outside TraceLog.
            </li>
          </ul>
          <p class="policy-hint policy-format">
            Format: one IP or CIDR per line (e.g. <code>203.0.113.50</code> or <code>10.0.0.0/8</code>). Lines starting with
            <code>#</code> are ignored for export.
            <a class="doc-ref" href="https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics" target="_blank" rel="noopener noreferrer">Docs</a>
          </p>
          <textarea
            class="policy-ta"
            bind:value={blacklistText}
            on:input={() => (blacklistDirty = true)}
            rows="4"
            placeholder="10.0.0.0/8&#10;192.0.2.1"
            aria-label="IP list for analytics and nginx export"
          ></textarea>
          <div class="policy-actions">
            <button type="button" class="btn-save-bl" disabled={savingBl} on:click={saveBlacklist}>{savingBl ? 'Saving...' : 'Save list'}</button>
            <button type="button" class="btn-export-bl" on:click={copyNginxDenySnippet}>Copy nginx deny snippet</button>
            <button type="button" class="btn-export-bl" on:click={downloadNginxDenySnippet}>Download .conf</button>
          </div>
          {#if saveError}<p class="error-msg">{saveError}</p>{/if}
        </div>
      {/if}

      <!-- Tab: Requests -->
      {#if activeTab === 'requests'}
        <div class="bad-section">
          <h3>Top IPs by error responses (4xx / 5xx)</h3>
          <p class="policy-hint">
            Ranking by number of <strong>client or server error</strong> responses per IP in the selected range (up to 50 IPs).
            Use <strong>Lines</strong> to load recent failing requests for that address. <strong>Show all bad requests</strong> lists samples from every IP.
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
              <thead><tr><th>IP address</th><th class="num">4xx/5xx count</th><th></th></tr></thead>
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
              <h4>Sample error request lines {#if badFilterIP}for {badFilterIP}{/if}</h4>
              {#if badLoading}
                <div class="muted">Loading...</div>
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
                            <a href={whoisHref(log.ip)} target="_blank" rel="noopener noreferrer" class="mini-whois">&#8599;</a>
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

        <div class="slow-section">
          <h3>Slow requests</h3>
          <p class="policy-hint">
            Rows where <strong>duration</strong> from the access log is at least the threshold (same time range as above). Sorted by <strong>slowest first</strong>. Uses the same User-Agent exclusions as top tables when configured under <strong>Settings &rarr; General</strong>.
          </p>
          <div class="slow-toolbar">
            <label class="slow-min-label">
              Min duration (ms)
              <input
                type="number"
                class="slow-min-input"
                bind:value={slowMinMs}
                min="1"
                max="3600000"
                step="1"
              />
            </label>
            <button type="button" class="btn-secondary" on:click={refreshSlowRequests} disabled={slowLoading}>
              {slowLoading ? 'Loading...' : 'Refresh'}
            </button>
          </div>
          {#if slowLoading && slowLogs.length === 0}
            <div class="muted">Loading...</div>
          {:else if slowLogs.length === 0}
            <div class="muted">No requests at or above this threshold in the range (or duration not present in logs).</div>
          {:else}
            <div class="recent-table slow-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th class="num">Duration</th>
                    <th>Time</th>
                    <th>Method</th>
                    <th>Path</th>
                    <th class="num">Status</th>
                    <th>IP</th>
                    <th>User-Agent</th>
                  </tr>
                </thead>
                <tbody>
                  {#each slowLogs as log}
                    <tr>
                      <td class="num slow-ms">{Math.round(Number(log.duration_ms) || 0)} ms</td>
                      <td class="mono">{new Date(log.ts).toLocaleString()}</td>
                      <td><span class="method">{log.method}</span></td>
                      <td class="path">{log.path}</td>
                      <td class="num {statusClass(log.status_code)}">{log.status_code}</td>
                      <td class="mono">
                        {log.ip}
                        <a href={whoisHref(log.ip)} target="_blank" rel="noopener noreferrer" class="mini-whois">&#8599;</a>
                      </td>
                      <td class="ua-cell" title={log.user_agent || ''}>{log.user_agent || '—'}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
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
                        <a href={whoisHref(log.ip)} target="_blank" rel="noopener noreferrer" class="mini-whois" title="WHOIS">&#8599;</a>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/if}
      {/if}
    {/if}
  </LoadingState>
</div>

<style>
  .analytics { padding: 1.5rem; max-width: none; }

  .page-intro {
    font-size: 0.82rem; color: var(--text-secondary); line-height: 1.5; margin-bottom: 1rem;
    padding: 0.75rem 1rem; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px;
  }
  .page-intro ul { margin: 0.4rem 0 0 0; padding-left: 1.2rem; }
  .page-intro li { margin-bottom: 0.25rem; }

  .section-lead { font-size: 0.72rem; color: var(--text-muted); margin: -0.25rem 0 0.5rem 0; line-height: 1.35; }
  .section-label { font-size: 0.85rem; color: var(--text-secondary); margin: 0 0 0.5rem; }

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

  /* Tabs */
  .tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--border); margin-bottom: 1.25rem; }
  .tab-btn {
    background: none; border: none; padding: 0.5rem 1rem; cursor: pointer;
    font-size: 0.82rem; font-weight: 500; color: var(--text-muted);
    border-bottom: 2px solid transparent; margin-bottom: -1px;
  }
  .tab-btn:hover { color: var(--text-primary); }
  .tab-btn.active { color: var(--accent); border-bottom-color: var(--accent); font-weight: 600; }

  /* Stats */
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

  /* Timeline chart */
  .timeline-chart-wrap { margin: 0 0 1rem; }
  .timeline-chart { width: 100%; min-height: 180px; }
  .chart-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 0.5rem; flex-wrap: wrap; gap: 0.5rem; }
  .chart-header h3 { margin: 0; }
  .chart-legend { display: flex; align-items: center; gap: 0.75rem; font-size: 0.8rem; color: var(--text-muted); }
  .chart-legend span:not(.legend-dot) { display: flex; align-items: center; gap: 0.3rem; }
  .legend-dot { display: inline-block; width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }

  /* Two-col layout */
  .two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
  @media (max-width: 900px) { .two-col { grid-template-columns: 1fr; } }

  /* Table sections */
  .table-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem; min-width: 0; overflow-x: auto;
    margin-bottom: 0.75rem;
  }

  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th {
    text-align: left; padding: 0.4rem 0.5rem; color: var(--text-muted);
    font-size: 0.72rem; font-weight: 600; border-bottom: 1px solid var(--border);
  }
  th.num, td.num { text-align: right; }
  td { padding: 0.35rem 0.5rem; color: var(--text-secondary); border-bottom: 1px solid var(--border); }
  td.mono { font-variant-numeric: tabular-nums; font-size: 0.75rem; }
  td.path {
    max-width: 72rem; white-space: normal; word-break: break-word;
    font-size: 0.78rem; line-height: 1.35;
  }
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

  /* Slow requests duration */
  .slow-ms { color: #f0883e; font-weight: 700; }

  /* Threat badges */
  .badge-threat { background: #f8514922; color: #f85149; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; font-weight: 700; }
  .badge-suspicious { background: #d2992222; color: #d29922; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; font-weight: 700; }
  .badge-scanner { background: #f0883e22; color: #f0883e; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; }
  .badge-bot { background: #bc8cff22; color: #bc8cff; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; }
  .badge-subnet { background: #58a6ff22; color: #58a6ff; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; }

  /* IP threat assessment */
  .ipinfo-cell { font-size: 0.75rem; }
  .ipinfo-cell .country { display: block; color: var(--text-primary); margin-bottom: 0.2rem; }
  .ipinfo-cell .abuse-badge { display: block; padding: 2px 4px; border-radius: 3px; background: #f8514922; color: #f85149; font-size: 0.65rem; font-weight: 600; }
  .ipinfo-cell .abuse-badge.abuse-high { background: #f85149; color: white; }
  .decision { display: inline-block; padding: 2px 6px; border-radius: 4px; font-size: 0.68rem; font-weight: 700; }
  .decision-block { background: #f8514944; color: #f85149; }
  .decision-monitor { background: #d2992244; color: #d29922; }
  .decision-allow { background: #3fb95044; color: #3fb950; }

  /* Recommended to block panel */
  .recommend-box {
    background: #f8514908; border: 1px solid #f8514944;
    border-radius: 10px; padding: 0.85rem 1rem; margin-bottom: 0.75rem;
  }
  .recommend-header {
    display: flex; justify-content: space-between; align-items: flex-start;
    gap: 0.75rem; margin-bottom: 0.5rem; flex-wrap: wrap;
  }
  .recommend-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
  .recommend-title { margin: 0 0 0.15rem; font-size: 0.88rem; color: #f85149; }
  .recommend-note { font-size: 0.72rem; color: var(--text-muted); margin: 0.5rem 0 0; line-height: 1.4; }
  .btn-add-all {
    padding: 0.35rem 0.85rem; font-size: 0.78rem; font-weight: 600;
    border-radius: 7px; cursor: pointer; white-space: nowrap;
    border: 1px solid #f85149; background: #f8514911; color: #f85149;
  }
  .btn-add-all:hover { background: #f8514922; }
  .btn-add-one {
    padding: 0.2rem 0.55rem; font-size: 0.72rem; font-weight: 600;
    border-radius: 5px; cursor: pointer; white-space: nowrap;
    border: 1px solid var(--border); background: var(--bg-primary); color: var(--text-primary);
  }
  .btn-add-one:hover { border-color: var(--accent); color: var(--accent); }


  /* Policy / blacklist box */
  .policy-box {
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px;
    padding: 0.85rem 1rem; margin-bottom: 1rem;
  }
  .policy-hint { font-size: 0.78rem; color: var(--text-muted); margin: 0 0 0.5rem; line-height: 1.45; }
  .policy-hint.policy-format { margin-top: 0.35rem; }
  .policy-list {
    margin: 0 0 0.5rem 0; padding-left: 1.15rem; font-size: 0.78rem; color: var(--text-secondary); line-height: 1.45;
  }
  .policy-list li { margin-bottom: 0.35rem; }
  .policy-list li:last-child { margin-bottom: 0; }
  .policy-ta {
    width: 100%; min-height: 4.5rem; font-family: monospace; font-size: 0.78rem;
    padding: 0.5rem; border-radius: 8px; border: 1px solid var(--border);
    background: var(--bg-primary); color: var(--text-primary); resize: vertical;
  }
  .policy-actions {
    display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; margin-top: 0.5rem;
  }
  .btn-save-bl {
    padding: 0.4rem 1rem; font-size: 0.8rem; font-weight: 600;
    border-radius: 8px; border: none; cursor: pointer; background: #238636; color: #fff;
  }
  .btn-save-bl:disabled { opacity: 0.6; cursor: wait; }
  .btn-export-bl {
    padding: 0.4rem 0.85rem; font-size: 0.78rem; font-weight: 600;
    border-radius: 8px; cursor: pointer; border: 1px solid var(--border);
    background: var(--bg-primary); color: var(--text-primary);
  }
  .btn-export-bl:hover { border-color: var(--accent); color: var(--accent); }

  /* Bad requests */
  .bad-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.85rem; margin-bottom: 0.75rem;
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

  /* Slow requests */
  .slow-section {
    margin-bottom: 0.75rem; padding: 0.85rem 1rem 1rem;
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px;
  }
  .slow-toolbar {
    display: flex; flex-wrap: wrap; align-items: flex-end; gap: 0.65rem; margin-bottom: 0.65rem;
  }
  .slow-min-label {
    font-size: 0.78rem; color: var(--text-secondary);
    display: flex; flex-direction: column; gap: 0.2rem;
  }
  .slow-min-input {
    width: 7rem; padding: 0.35rem 0.45rem;
    background: var(--bg-primary); border: 1px solid var(--border);
    border-radius: 6px; color: var(--text-primary); font-size: 0.85rem;
  }
  .slow-ms { font-weight: 600; color: var(--accent); }
  .slow-table-wrap { overflow-x: auto; }
  .ua-cell {
    max-width: 220px; overflow: hidden; text-overflow: ellipsis;
    white-space: nowrap; font-size: 0.72rem; color: var(--text-muted);
  }

  /* Recent requests */
  .recent-section {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem;
  }
  .recent-table { overflow-x: auto; }
  .mini-whois { margin-left: 0.25rem; font-size: 0.7rem; color: var(--accent); text-decoration: none; }
  .mini-whois:hover { text-decoration: underline; }

  .status-msg { text-align: center; padding: 4rem; color: var(--text-muted); }
  .error-msg { color: var(--danger); font-size: 0.82rem; margin: 0.5rem 0; padding: 0.4rem 0.75rem; background: rgba(248,81,73,0.08); border: 1px solid rgba(248,81,73,0.25); border-radius: 6px; }
  code { background: var(--bg-primary); padding: 1px 5px; border-radius: 4px; font-size: 0.85em; }
  .doc-ref { margin-left: 0.35rem; color: var(--accent); font-size: inherit; }

  .range-loading { font-size: 0.75rem; color: var(--text-muted); animation: pulse 1s infinite; }
  .range-loaded { font-size: 0.75rem; color: var(--text-muted); }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.4} }
  .tab-loading { padding: 1rem; color: var(--text-muted); font-size: 0.9rem; }
</style>
