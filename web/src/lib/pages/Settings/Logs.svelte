<script lang="ts">
  import { api } from '../../api';

  export let servers: any[] = [];

  // Log sources
  let logSources: any[] = [];
  let newLogName = ''; let newLogPath = ''; let newLogFormat = 'plain';
  /** Empty = local hub agent; set to a server id for remote `tracelog agent` tail (path must exist on that host). */
  let newLogServerId = '';

  const ingestLevelOpts = ['critical', 'error', 'warn', 'info', 'debug', 'deprecated'] as const;
  /** Per log source id: which severities to store (empty = all). */
  let ingestPick: Record<string, Record<string, boolean>> = {};

  import { onMount } from 'svelte';

  onMount(async () => {
    logSources = (await api.listLogSources()) || [];
    syncIngestPickFromSources();
  });

  function syncIngestPickFromSources() {
    const next: Record<string, Record<string, boolean>> = {};
    for (const ls of logSources) {
      const cur = ls.ingest_levels;
      const row: Record<string, boolean> = {};
      for (const lv of ingestLevelOpts) {
        row[lv] = Array.isArray(cur) && cur.includes(lv);
      }
      next[ls.id] = row;
    }
    ingestPick = next;
  }

  async function saveIngestLevels(lsId: string) {
    const row = ingestPick[lsId] || {};
    const levels = ingestLevelOpts.filter((lv) => row[lv]);
    try {
      await api.updateLogSource(lsId, { ingest_levels: levels });
      logSources = (await api.listLogSources()) || [];
      syncIngestPickFromSources();
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function addLogSource() {
    const name = newLogName.trim();
    const path = newLogPath.trim();
    if (!name || !path) {
      alert('Enter a display name and an absolute file path (e.g. /var/log/nginx/access.log).');
      return;
    }
    try {
      await api.createLogSource({
        name,
        path,
        format: newLogFormat,
        type: 'file',
        server_id: newLogServerId.trim(),
        enabled: true,
      });
      newLogName = ''; newLogPath = '';
      logSources = (await api.listLogSources()) || [];
      syncIngestPickFromSources();
    } catch (e: any) { alert('Failed: ' + e.message); }
  }

  async function removeLogSource(id: string) {
    await api.deleteLogSource(id);
    logSources = (await api.listLogSources()) || [];
  }

  async function scanLogs() {
    try {
      const d = await api.detect();
      if (d.log_files && d.log_files.length > 0) {
        for (const lf of d.log_files) {
          await api.createLogSource({ name: lf.name, path: lf.path, format: lf.format, type: lf.type, server_id: '', enabled: true });
        }
        logSources = (await api.listLogSources()) || [];
        syncIngestPickFromSources();
        alert(`Found and added ${d.log_files.length} log sources.`);
      } else {
        alert('No common log files found on this system.');
      }
    } catch (e: any) { alert('Scan failed: ' + e.message); }
  }

  function logSourceAgentLabel(sid: string): string {
    if (!sid?.trim()) return 'This hub (local agent)';
    const s = servers.find((x) => x.id === sid);
    return s ? `Remote: ${s.name}` : sid.slice(0, 12);
  }
</script>

<div class="section">
  <h3>Log Sources</h3>
  <p class="hint">
    <strong>Local hub (serve mode):</strong> sources with agent <em>This hub</em> are loaded when TraceLog starts on this machine.
    <strong>Restart TraceLog</strong> after add/remove (e.g. <code>sudo systemctl restart tracelog</code>).
    <strong>Remote agent:</strong> choose a monitored server — the file path must exist <em>on that host</em>; <code>tracelog agent</code> polls the hub about every <strong>2 minutes</strong> and starts tailing without a restart.
    The agent <strong>tails from the end of each file</strong> — only new lines after tail starts appear in Logs.
  </p>
  <p class="hint">Scan checks <strong>this</strong> machine for usual paths and adds local sources only.</p>
  <button class="btn-secondary" on:click={scanLogs}>Scan for common log files</button>
  <div class="add-form add-form-logs">
    <input type="text" bind:value={newLogName} placeholder="Name" />
    <input type="text" bind:value={newLogPath} placeholder="/var/log/..." />
    <select bind:value={newLogFormat} class="log-format-select">
      <option value="plain">Plain</option>
      <option value="nginx">Nginx</option>
      <option value="apache">Apache</option>
    </select>
    <select bind:value={newLogServerId} class="log-agent-select" title="Which agent tails this path">
      <option value="">This hub (local agent)</option>
      {#each servers as srv}
        <option value={srv.id}>Remote: {srv.name}</option>
      {/each}
    </select>
    <button class="btn-save" on:click={addLogSource}>Add</button>
  </div>
  <p class="hint">For <strong>This hub</strong>, the file must exist here and nginx/apache formats are validated from file samples. For <strong>Remote</strong>, only name/path/format are checked on the hub — ensure the path is correct on the agent host.</p>
  <p class="hint">
    <strong>Store only chosen severities</strong> (per source): unchecked = ingest nothing for that level. If <em>none</em> are checked, <strong>all</strong> levels are stored. After changing filters, <strong>restart TraceLog</strong> so the agent reloads config. The hub also drops non-matching lines as a safeguard.
  </p>
  {#if logSources.length === 0}
    <p class="hint">No log sources configured. Click "Scan" or add manually above.</p>
  {:else}
    <div class="item-list">
      {#each logSources as ls (ls.id)}
        <div class="item-row ingest-row">
          <div class="ingest-main">
            <strong>{ls.name}</strong>
            <span class="item-detail">{logSourceAgentLabel(ls.server_id)} · {ls.path || ls.container} ({ls.format})</span>
            <div class="ingest-levels">
              {#each ingestLevelOpts as lv}
                <label class="ingest-cb"
                  ><input type="checkbox" bind:checked={ingestPick[ls.id][lv]} /> {lv}</label
                >
              {/each}
            </div>
            <button type="button" class="btn-secondary ingest-save" on:click={() => saveIngestLevels(ls.id)}
              >Save levels</button
            >
          </div>
          <button class="btn-delete" on:click={() => removeLogSource(ls.id)}>Delete</button>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .ingest-row { align-items: flex-start; }
  .ingest-main { flex: 1; min-width: 0; }
  .ingest-levels { display: flex; flex-wrap: wrap; gap: 0.35rem 0.9rem; margin: 0.45rem 0; }
  .ingest-cb { font-size: 0.72rem; color: var(--text-secondary); cursor: pointer; user-select: none; }
  .ingest-save { margin-bottom: 0 !important; margin-top: 0.35rem; }
  .add-form-logs { align-items: flex-end; }
  .log-agent-select { min-width: 180px; flex: 1 1 160px; }
</style>
