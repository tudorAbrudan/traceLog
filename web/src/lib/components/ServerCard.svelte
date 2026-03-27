<script lang="ts">
  import { onMount } from 'svelte';
  import { contextServerId, currentPage, navDrawerOpen, suppressSingleServerAutoOpen } from '../store';
  import { api } from '../api';

  export let server: any;

  let cpuPercent = 0;
  let memPercent = 0;
  let diskPercent = 0;
  let hasMetrics = false;

  onMount(async () => {
    try {
      const metrics = await api.getMetrics(server.id, '1h');
      if (metrics && metrics.length > 0) {
        const latest = metrics[metrics.length - 1];
        cpuPercent = latest.cpu_percent || 0;
        memPercent = latest.mem_total > 0 ? (latest.mem_used / latest.mem_total) * 100 : 0;
        diskPercent = latest.disk_total > 0 ? (latest.disk_used / latest.disk_total) * 100 : 0;
        hasMetrics = true;
      }
    } catch {}
  });

  function openDetail() {
    suppressSingleServerAutoOpen.set(true);
    contextServerId.set(server.id);
    currentPage.set(`server:${server.id}`);
    navDrawerOpen.set(false);
  }

  function timeSince(dateStr: string): string {
    if (!dateStr) return 'Never';
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  }

  function barColor(pct: number): string {
    if (pct > 90) return '#f85149';
    if (pct > 70) return '#d29922';
    return '#3fb950';
  }
</script>

<button class="card" on:click={openDetail}>
  <div class="card-header">
    <div class="name-row">
      <span class="indicator" class:online={server.status === 'online'}></span>
      <h3>{server.name}</h3>
    </div>
    <span class="status" class:online={server.status === 'online'}>{server.status}</span>
  </div>

  {#if server.host}
    <div class="host">{server.host}</div>
  {/if}

  {#if hasMetrics}
    <div class="metrics-bars">
      <div class="bar-row">
        <span class="bar-label">CPU</span>
        <div class="bar-track">
          <div class="bar-fill" style="width:{cpuPercent}%;background:{barColor(cpuPercent)}"></div>
        </div>
        <span class="bar-value">{cpuPercent.toFixed(0)}%</span>
      </div>
      <div class="bar-row">
        <span class="bar-label">RAM</span>
        <div class="bar-track">
          <div class="bar-fill" style="width:{memPercent}%;background:{barColor(memPercent)}"></div>
        </div>
        <span class="bar-value">{memPercent.toFixed(0)}%</span>
      </div>
      <div class="bar-row">
        <span class="bar-label">Disk</span>
        <div class="bar-track">
          <div class="bar-fill" style="width:{diskPercent}%;background:{barColor(diskPercent)}"></div>
        </div>
        <span class="bar-value">{diskPercent.toFixed(0)}%</span>
      </div>
    </div>
  {/if}

  <div class="meta">
    <span>Last seen: {timeSince(server.last_seen_at)}</span>
  </div>
</button>

<style>
  .card {
    display: block; width: 100%; text-align: left; background: var(--bg-secondary);
    border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem;
    cursor: pointer; transition: border-color 0.15s, background 0.15s; color: inherit; font: inherit;
  }
  .card:hover { border-color: #58a6ff44; background: var(--bg-hover); }
  .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem; }
  .name-row { display: flex; align-items: center; gap: 0.5rem; }
  .indicator { width: 8px; height: 8px; border-radius: 50%; background: #6e7681; flex-shrink: 0; }
  .indicator.online { background: #3fb950; }
  h3 { font-size: 1rem; margin: 0; color: var(--text-primary); }
  .status { padding: 2px 8px; border-radius: 20px; font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; background: #6e768133; color: #8b949e; }
  .status.online { background: #23863633; color: #3fb950; }
  .host { font-size: 0.8rem; color: var(--text-muted); margin-bottom: 0.75rem; }
  .metrics-bars { display: flex; flex-direction: column; gap: 0.4rem; margin-bottom: 0.75rem; }
  .bar-row { display: flex; align-items: center; gap: 0.5rem; }
  .bar-label { width: 30px; font-size: 0.7rem; color: var(--text-muted); font-weight: 600; }
  .bar-track { flex: 1; height: 6px; background: var(--bg-primary); border-radius: 3px; overflow: hidden; }
  .bar-fill { height: 100%; border-radius: 3px; transition: width 0.3s; }
  .bar-value { width: 32px; text-align: right; font-size: 0.7rem; color: var(--text-secondary); }
  .meta { font-size: 0.75rem; color: var(--text-muted); }
</style>
