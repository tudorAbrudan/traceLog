<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';
  import { currentPage } from '../store';

  let servers: any[] = [];
  let selectedServer = '';
  let processes: any[] = [];
  let loading = true;
  let sortBy = 'cpu_percent';
  let sortDir: 'asc' | 'desc' = 'desc';

  onMount(async () => {
    try {
      servers = await api.listServers();
      if (servers.length > 0) {
        selectedServer = servers[0].id;
        await loadProcesses();
      }
    } catch (e) {
      console.error('Failed to load servers', e);
    } finally {
      loading = false;
    }

    const interval = setInterval(loadProcesses, 10000);
    return () => clearInterval(interval);
  });

  async function loadProcesses() {
    if (!selectedServer) return;
    try {
      processes = await api.getProcesses(selectedServer, true);
    } catch (e) {
      console.error('Failed to load processes', e);
    }
  }

  function fmtBytes(b: number): string {
    if (b >= 1073741824) return (b / 1073741824).toFixed(1) + ' GB';
    if (b >= 1048576) return (b / 1048576).toFixed(1) + ' MB';
    if (b >= 1024) return (b / 1024).toFixed(1) + ' KB';
    return b + ' B';
  }

  function sort(field: string) {
    if (sortBy === field) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = field;
      sortDir = 'desc';
    }
  }

  $: sorted = [...processes].sort((a, b) => {
    const va = a[sortBy] ?? 0;
    const vb = b[sortBy] ?? 0;
    return sortDir === 'asc' ? (va > vb ? 1 : -1) : (va < vb ? 1 : -1);
  });
</script>

<div class="processes">
  <div class="header">
    <h2>Processes</h2>
    <div class="controls">
      {#if servers.length > 1}
        <select bind:value={selectedServer} on:change={loadProcesses}>
          {#each servers as s}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
      {/if}
      <span class="count">{processes.length} processes</span>
    </div>
  </div>

  {#if loading}
    <div class="status-msg">Loading processes...</div>
  {:else if processes.length === 0}
    <div class="status-msg">No process data yet. Data will appear as it's collected.</div>
  {:else}
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th class="sortable" on:click={() => sort('pid')}>PID {sortBy === 'pid' ? (sortDir === 'asc' ? '↑' : '↓') : ''}</th>
            <th class="sortable" on:click={() => sort('name')}>Name {sortBy === 'name' ? (sortDir === 'asc' ? '↑' : '↓') : ''}</th>
            <th class="sortable num" on:click={() => sort('cpu_percent')}>CPU % {sortBy === 'cpu_percent' ? (sortDir === 'asc' ? '↑' : '↓') : ''}</th>
            <th class="sortable num" on:click={() => sort('mem_rss')}>Memory {sortBy === 'mem_rss' ? (sortDir === 'asc' ? '↑' : '↓') : ''}</th>
            <th class="num">Threads</th>
            <th>Status</th>
            <th>Command</th>
          </tr>
        </thead>
        <tbody>
          {#each sorted as proc}
            <tr>
              <td class="mono">{proc.pid}</td>
              <td class="name">{proc.name}</td>
              <td class="num" class:warn={proc.cpu_percent > 50} class:danger={proc.cpu_percent > 80}>
                {proc.cpu_percent.toFixed(1)}%
              </td>
              <td class="num">{fmtBytes(proc.mem_rss)}</td>
              <td class="num">{proc.threads}</td>
              <td><span class="status-badge" class:running={proc.status === 'running' || proc.status === 'sleep'}>{proc.status}</span></td>
              <td class="cmdline">{proc.cmdline || '—'}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .processes { padding: 1.5rem; max-width: 1400px; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  h2 { margin: 0; font-size: 1.3rem; color: var(--text-primary); }
  .controls { display: flex; align-items: center; gap: 0.75rem; }
  .count { font-size: 0.8rem; color: var(--text-muted); }

  select {
    background: var(--bg-secondary); color: var(--text-primary);
    border: 1px solid var(--border); border-radius: 6px;
    padding: 0.35rem 0.6rem; font-size: 0.8rem;
  }

  .table-wrap {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 10px; overflow: auto;
  }
  table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
  th {
    text-align: left; padding: 0.6rem 0.75rem; color: var(--text-muted);
    font-weight: 600; border-bottom: 1px solid var(--border);
    font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.04em;
    white-space: nowrap;
  }
  th.sortable { cursor: pointer; user-select: none; }
  th.sortable:hover { color: var(--text-primary); }
  th.num, td.num { text-align: right; }
  td {
    padding: 0.45rem 0.75rem; border-bottom: 1px solid var(--border);
    color: var(--text-secondary);
  }
  td.mono { font-variant-numeric: tabular-nums; }
  td.name { color: var(--text-primary); font-weight: 500; }
  td.warn { color: var(--warning); }
  td.danger { color: var(--danger); font-weight: 600; }
  .cmdline {
    max-width: 350px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
    font-size: 0.72rem; color: var(--text-muted);
  }
  .status-badge {
    padding: 2px 6px; border-radius: 4px; font-size: 0.68rem;
    background: #6e768122; color: var(--text-muted);
  }
  .status-badge.running { background: #23863622; color: #3fb950; }
  .status-msg { text-align: center; padding: 4rem; color: var(--text-muted); }
  tr:last-child td { border-bottom: none; }
</style>
