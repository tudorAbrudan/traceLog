<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';
  import Chart from '../components/Chart.svelte';
  import { currentPage, suppressSingleServerAutoOpen } from '../store';

  export let serverId = '';

  let server: any = null;
  let metrics: any[] = [];
  let range_ = '1h';
  let loading = true;
  let firstLoad = true;

  const ranges = [
    { value: '1h', label: '1H' },
    { value: '6h', label: '6H' },
    { value: '24h', label: '24H' },
    { value: '7d', label: '7D' },
    { value: '30d', label: '30D' },
  ];

  async function loadData() {
    if (firstLoad) loading = true;
    try {
      [server, metrics] = await Promise.all([
        api.getServer(serverId),
        api.getMetrics(serverId, range_),
      ]);
      if (server?.host === 'localhost') {
        try {
          dockerRows = await api.getDockerMetrics(serverId, '1h');
        } catch {
          dockerRows = [];
        }
      } else {
        dockerRows = [];
      }
    } catch (e) {
      console.error('Failed to load server:', e);
    } finally {
      loading = false;
      firstLoad = false;
    }
  }

  onMount(() => {
    loadData();
    const interval = setInterval(loadData, 15000);
    return () => clearInterval(interval);
  });

  function selectRange(r: string) {
    range_ = r;
    firstLoad = true;
    loadData();
  }

  function goBack() {
    suppressSingleServerAutoOpen.set(true);
    currentPage.set('overview');
  }

  function fmtBytes(b: number): string {
    if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB';
    if (b >= 1048576) return (b / 1048576).toFixed(1) + ' MB';
    if (b >= 1024) return (b / 1024).toFixed(1) + ' KB';
    return b + ' B';
  }

  $: latest = metrics.length > 0 ? metrics[metrics.length - 1] : null;
  $: cpuPct = latest?.cpu_percent ?? 0;
  $: memPct = latest && latest.mem_total > 0 ? (latest.mem_used / latest.mem_total * 100) : 0;
  $: diskPct = latest && latest.disk_total > 0 ? (latest.disk_used / latest.disk_total * 100) : 0;
</script>

<div class="detail">
  <div class="top-bar">
    <button class="back" on:click={goBack}>
      <span class="back-arrow">←</span> Back
    </button>
    {#if server}
      <div class="server-info">
        <h2>{server.name}</h2>
        <span class="badge" class:online={server.status === 'online'}>{server.status}</span>
        {#if server.host}
          <span class="host">{server.host}</span>
        {/if}
      </div>
    {/if}
  </div>

  {#if latest}
    <div class="summary-row">
      <div class="summary-item">
        <span class="summary-value" class:warn={cpuPct > 70} class:danger={cpuPct > 90}>{cpuPct.toFixed(1)}%</span>
        <span class="summary-label">CPU</span>
      </div>
      <div class="summary-item">
        <span class="summary-value" class:warn={memPct > 70} class:danger={memPct > 90}>{memPct.toFixed(1)}%</span>
        <span class="summary-label">Memory ({fmtBytes(latest.mem_used)} / {fmtBytes(latest.mem_total)})</span>
      </div>
      <div class="summary-item">
        <span class="summary-value" class:warn={diskPct > 70} class:danger={diskPct > 90}>{diskPct.toFixed(1)}%</span>
        <span class="summary-label">Disk ({fmtBytes(latest.disk_used)} / {fmtBytes(latest.disk_total)})</span>
      </div>
      <div class="summary-item">
        <span class="summary-value">{latest.load_1?.toFixed(2)}</span>
        <span class="summary-label">Load (1m)</span>
      </div>
    </div>
  {/if}

  <div class="range-bar">
    {#each ranges as r}
      <button class:active={range_ === r.value} on:click={() => selectRange(r.value)}>{r.label}</button>
    {/each}
  </div>

  {#if loading && metrics.length === 0}
    <div class="status-msg">Loading metrics...</div>
  {:else if metrics.length === 0}
    <div class="status-msg">No metrics data yet. Data will appear as it's collected.</div>
  {:else}
    <div class="charts-grid">
      <div class="chart-card full">
        <div class="card-head">
          <h3>CPU Usage</h3>
          <span class="card-val">{cpuPct.toFixed(1)}%</span>
        </div>
        <Chart data={metrics} field="cpu_percent" unit="%" color="#58a6ff" label="CPU" />
      </div>

      <div class="chart-card">
        <div class="card-head">
          <h3>Memory</h3>
          <span class="card-val">{memPct.toFixed(1)}%</span>
        </div>
        <Chart data={metrics} field="mem_used" total="mem_total" unit="%" color="#bc8cff" label="Used" />
      </div>

      <div class="chart-card">
        <div class="card-head">
          <h3>Disk</h3>
          <span class="card-val">{diskPct.toFixed(1)}%</span>
        </div>
        <Chart data={metrics} field="disk_used" total="disk_total" unit="%" color="#f0883e" label="Used" />
      </div>

      <div class="chart-card">
        <div class="card-head">
          <h3>Network I/O</h3>
        </div>
        <Chart data={metrics} field="net_rx_bytes" field2="net_tx_bytes" unit="bytes/s" color="#3fb950" color2="#f85149" label="RX" label2="TX" />
      </div>

      <div class="chart-card">
        <div class="card-head">
          <h3>Load Average</h3>
          <span class="card-val">{latest?.load_1?.toFixed(2)} / {latest?.load_5?.toFixed(2)}</span>
        </div>
        <Chart data={metrics} field="load_1" field2="load_5" unit="" color="#58a6ff" color2="#6e7681" label="1m" label2="5m" />
      </div>
    </div>
  {/if}
</div>

<style>
  .detail {
    padding: 1.5rem;
    max-width: 1200px;
  }

  .top-bar {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.25rem;
  }

  .back {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 0.4rem 0.85rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    transition: all 0.15s;
    flex-shrink: 0;
  }
  .back:hover { background: var(--bg-hover); color: var(--text-primary); }
  .back-arrow { font-size: 1rem; }

  .server-info { display: flex; align-items: center; gap: 0.75rem; }
  h2 { margin: 0; color: var(--text-primary); font-size: 1.35rem; }
  .badge {
    padding: 3px 10px; border-radius: 20px; font-size: 0.7rem;
    font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em;
    background: #6e768133; color: #8b949e;
  }
  .badge.online { background: #23863633; color: #3fb950; }
  .host { font-size: 0.8rem; color: var(--text-muted); }

  /* Summary cards */
  .summary-row {
    display: flex; gap: 0.75rem; margin-bottom: 1.25rem; flex-wrap: wrap;
  }
  .summary-item {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; padding: 0.75rem 1rem; flex: 1; min-width: 140px;
  }
  .summary-value {
    display: block; font-size: 1.4rem; font-weight: 700; color: #3fb950;
    line-height: 1.2;
  }
  .summary-value.warn { color: #d29922; }
  .summary-value.danger { color: #f85149; }
  .summary-label { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.15rem; display: block; }

  /* Range selector */
  .range-bar {
    display: inline-flex; gap: 2px; margin-bottom: 1.25rem;
    background: var(--bg-secondary); padding: 3px; border-radius: 8px;
    border: 1px solid var(--border);
  }
  .range-bar button {
    padding: 0.3rem 0.8rem; background: none; border: none;
    color: var(--text-muted); border-radius: 6px; cursor: pointer;
    font-size: 0.78rem; font-weight: 600; transition: all 0.15s;
  }
  .range-bar button:hover { color: var(--text-primary); }
  .range-bar button.active {
    background: var(--accent); color: #fff;
  }

  /* Charts grid */
  .charts-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 0.75rem;
  }

  .chart-card {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1rem 1rem 0.5rem;
    overflow: hidden;
  }

  .chart-card.full {
    grid-column: 1 / -1;
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

  .card-val {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  .status-msg {
    text-align: center; padding: 4rem; color: var(--text-muted);
  }

  @media (max-width: 900px) {
    .charts-grid { grid-template-columns: 1fr; }
    .chart-card.full { grid-column: 1; }
  }

</style>
