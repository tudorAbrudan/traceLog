<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';
  import Chart from '../components/Chart.svelte';
  import { currentPage } from '../store';

  export let serverId = '';

  let server: any = null;
  let metrics: any[] = [];
  let range_ = '1h';
  let loading = true;

  const ranges = ['1h', '6h', '24h', '7d', '30d'];

  async function loadData() {
    loading = true;
    try {
      [server, metrics] = await Promise.all([
        api.getServer(serverId),
        api.getMetrics(serverId, range_),
      ]);
    } catch (e) {
      console.error('Failed to load server:', e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
    const interval = setInterval(loadData, 10000);
    return () => clearInterval(interval);
  });

  function goBack() {
    currentPage.set('overview');
  }

  $: if (range_) loadData();
</script>

<div class="detail">
  <div class="header">
    <button class="back" on:click={goBack}>← Back</button>
    {#if server}
      <h2>{server.name}</h2>
      <span class="status" class:online={server.status === 'online'}>{server.status}</span>
    {/if}
  </div>

  <div class="range-selector">
    {#each ranges as r}
      <button class:active={range_ === r} on:click={() => range_ = r}>{r}</button>
    {/each}
  </div>

  {#if loading && metrics.length === 0}
    <div class="loading">Loading metrics...</div>
  {:else if metrics.length === 0}
    <div class="empty">
      <p>No metrics data yet. Data will appear as it's collected.</p>
    </div>
  {:else}
    <div class="charts-grid">
      <div class="chart-card">
        <h3>CPU Usage</h3>
        <Chart data={metrics} field="cpu_percent" unit="%" color="#58a6ff" />
      </div>
      <div class="chart-card">
        <h3>Memory Usage</h3>
        <Chart data={metrics} field="mem_used" total="mem_total" unit="bytes" color="#bc8cff" />
      </div>
      <div class="chart-card">
        <h3>Disk Usage</h3>
        <Chart data={metrics} field="disk_used" total="disk_total" unit="bytes" color="#f0883e" />
      </div>
      <div class="chart-card">
        <h3>Network I/O</h3>
        <Chart data={metrics} field="net_rx_bytes" field2="net_tx_bytes" unit="bytes/s" color="#3fb950" color2="#f85149" />
      </div>
      <div class="chart-card">
        <h3>Load Average</h3>
        <Chart data={metrics} field="load_1" field2="load_5" unit="" color="#58a6ff" color2="#8b949e" />
      </div>
    </div>
  {/if}
</div>

<style>
  .detail { padding: 1.5rem; }
  .header {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1.5rem;
  }
  .back {
    background: none;
    border: 1px solid var(--border);
    color: var(--text-secondary);
    padding: 0.4rem 0.8rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
  }
  .back:hover { background: var(--bg-secondary); }
  h2 { margin: 0; color: var(--text-primary); }
  .status {
    padding: 2px 10px;
    border-radius: 20px;
    font-size: 0.75rem;
    font-weight: 600;
    background: #6e768166;
    color: #8b949e;
  }
  .status.online { background: #23863666; color: #3fb950; }
  .range-selector {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 1.5rem;
    background: var(--bg-secondary);
    padding: 0.25rem;
    border-radius: 8px;
    width: fit-content;
  }
  .range-selector button {
    padding: 0.35rem 0.75rem;
    background: none;
    border: none;
    color: var(--text-muted);
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
  }
  .range-selector button.active {
    background: var(--bg-primary);
    color: var(--text-primary);
  }
  .charts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
    gap: 1rem;
  }
  .chart-card {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.25rem;
  }
  .chart-card h3 {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin: 0 0 1rem 0;
    font-weight: 500;
  }
  .loading, .empty { text-align: center; padding: 4rem; color: var(--text-muted); }
</style>
