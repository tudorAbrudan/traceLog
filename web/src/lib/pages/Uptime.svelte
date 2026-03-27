<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  let checks: any[] = [];
  let showForm = false;
  let loading = true;
  /** check_id -> results oldest-first, capped */
  let resultsByCheck: Record<string, any[]> = {};

  let newName = '';
  let newUrl = '';
  let newInterval = 60;

  onMount(() => {
    void loadChecks();
    const interval = setInterval(() => void loadChecks(), 30000);
    return () => clearInterval(interval);
  });

  async function loadChecks() {
    try {
      const list = (await api.listUptimeChecks()) || [];
      checks = list;
      const next: Record<string, any[]> = {};
      await Promise.all(
        list.map(async (c: { id: string }) => {
          try {
            const r = await api.getUptimeResults(c.id, '7d');
            const arr = (r || []).slice().reverse();
            next[c.id] = arr.length > 200 ? arr.slice(-200) : arr;
          } catch {
            next[c.id] = [];
          }
        }),
      );
      resultsByCheck = next;
    } catch {
      /* keep */
    } finally {
      loading = false;
    }
  }

  async function addCheck() {
    if (!newName || !newUrl) return;
    try {
      await api.createUptimeCheck({
        name: newName,
        url: newUrl,
        interval_seconds: newInterval,
        timeout_seconds: 10,
        enabled: true,
      });
      newName = ''; newUrl = ''; showForm = false;
      await loadChecks();
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function removeCheck(id: string) {
    if (!confirm('Delete this monitor?')) return;
    try {
      await api.deleteUptimeCheck(id);
      await loadChecks();
    } catch {}
  }
</script>

<div class="uptime-page">
  <div class="header">
    <h2>Uptime Monitors</h2>
    <button class="btn-add" on:click={() => showForm = !showForm}>
      {showForm ? 'Cancel' : '+ Add Monitor'}
    </button>
  </div>

  <p class="lead">
    HTTP checks run from the TraceLog hub. Monitoring a URL on the <strong>same machine</strong> only tells you the process responds locally; for real availability use a public URL or an external vantage point. History bars: <span class="lg">green</span> = up, <span class="lr">red</span> = down (last 7 days, up to 200 samples).
  </p>

  {#if showForm}
    <div class="form-card">
      <div class="form-row">
        <div class="field">
          <label for="upt-name">Name</label>
          <input id="upt-name" type="text" bind:value={newName} placeholder="My Website" />
        </div>
        <div class="field">
          <label for="upt-url">URL</label>
          <input id="upt-url" type="url" bind:value={newUrl} placeholder="https://example.com" />
        </div>
        <div class="field">
          <label for="upt-int">Interval</label>
          <select id="upt-int" bind:value={newInterval}>
            <option value={30}>30s</option>
            <option value={60}>1 min</option>
            <option value={300}>5 min</option>
          </select>
        </div>
        <button class="btn-save" on:click={addCheck}>Create</button>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="empty">Loading...</div>
  {:else if checks.length === 0}
    <div class="empty-card">
      <h3>No monitors configured</h3>
      <p>Add HTTP endpoints to monitor uptime, response time, and status codes.</p>
    </div>
  {:else}
    <div class="monitors-list">
      {#each checks as check (check.id)}
        <div class="monitor-card">
          <div class="monitor-block">
            <div class="monitor-info">
              <span class="indicator" class:enabled={check.enabled}></span>
              <div>
                <div class="monitor-name">{check.name}</div>
                <div class="monitor-url">{check.url}</div>
              </div>
            </div>
            {#if resultsByCheck[check.id]?.length}
              <div class="uptime-strip" role="img" aria-label={`Uptime history for ${check.name}`}>
                {#each resultsByCheck[check.id] as r}
                  <div
                    class="uptime-cell"
                    class:up={r.up}
                    class:down={!r.up}
                    title="{new Date(r.ts).toLocaleString()} — {r.up ? 'Up' : 'Down'}{r.status_code ? ` · HTTP ${r.status_code}` : ''}{r.error ? ` · ${r.error}` : ''}"
                  ></div>
                {/each}
              </div>
            {:else}
              <div class="no-history muted">No history in this window yet.</div>
            {/if}
          </div>
          <div class="monitor-meta">
            <span class="interval">Every {check.interval_seconds}s</span>
            <button class="btn-delete" on:click={() => removeCheck(check.id)}>Delete</button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .uptime-page { padding: 1.5rem; max-width: 900px; }
  .lead {
    font-size: 0.82rem; color: var(--text-secondary); line-height: 1.5; margin: -0.5rem 0 1.25rem 0;
  }
  .lg { color: #3fb950; font-weight: 600; }
  .lr { color: #f85149; font-weight: 600; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0; }
  .btn-add { padding: 0.5rem 1rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-size: 0.85rem; font-weight: 600; }
  .btn-add:hover { background: #2ea043; }
  .form-card { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; margin-bottom: 1rem; }
  .form-row { display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap; }
  .field { display: flex; flex-direction: column; gap: 0.25rem; }
  .field label { font-size: 0.8rem; color: var(--text-secondary); }
  .field input, .field select { padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 0.85rem; }
  .field input:focus { border-color: #58a6ff; outline: none; }
  .btn-save { padding: 0.5rem 1.25rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-weight: 600; font-size: 0.85rem; }
  .empty, .empty-card { text-align: center; padding: 4rem 2rem; color: var(--text-muted); }
  .empty-card { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px; }
  .empty-card h3 { color: var(--text-primary); margin-bottom: 0.5rem; }
  .monitors-list { display: flex; flex-direction: column; gap: 0.65rem; }
  .monitor-card {
    display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem;
    background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px; padding: 1rem 1.25rem;
  }
  .monitor-block { flex: 1; min-width: 0; }
  .monitor-info { display: flex; align-items: center; gap: 0.75rem; }
  .indicator { width: 10px; height: 10px; border-radius: 50%; background: #6e7681; flex-shrink: 0; }
  .indicator.enabled { background: #3fb950; }
  .monitor-name { font-weight: 600; color: var(--text-primary); font-size: 0.9rem; }
  .monitor-url { font-size: 0.8rem; color: var(--text-muted); word-break: break-all; }
  .monitor-meta { display: flex; flex-direction: column; align-items: flex-end; gap: 0.5rem; flex-shrink: 0; }
  .interval { font-size: 0.8rem; color: var(--text-muted); white-space: nowrap; }
  .btn-delete { padding: 0.3rem 0.7rem; background: none; border: 1px solid var(--border); color: var(--text-muted); border-radius: 6px; cursor: pointer; font-size: 0.75rem; }
  .btn-delete:hover { border-color: #f85149; color: #f85149; }

  .uptime-strip {
    display: flex; flex-wrap: wrap; gap: 2px; margin-top: 0.65rem; align-items: stretch;
    min-height: 16px;
  }
  .uptime-cell {
    flex: 1 1 4px; min-width: 3px; max-width: 10px; height: 14px; border-radius: 2px;
    background: #6e7681;
  }
  .uptime-cell.up { background: #3fb950; }
  .uptime-cell.down { background: #f85149; }
  .no-history { font-size: 0.75rem; margin-top: 0.5rem; }
  .muted { color: var(--text-muted); }
</style>
