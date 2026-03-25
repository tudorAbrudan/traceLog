<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';

  let checks: any[] = [];
  let showForm = false;
  let loading = true;

  let newName = '';
  let newUrl = '';
  let newInterval = 60;

  onMount(async () => {
    await loadChecks();
    const interval = setInterval(loadChecks, 30000);
    return () => clearInterval(interval);
  });

  async function loadChecks() {
    try {
      checks = (await api.listUptimeChecks()) || [];
    } catch {} finally { loading = false; }
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
          <div class="monitor-info">
            <span class="indicator" class:enabled={check.enabled}></span>
            <div>
              <div class="monitor-name">{check.name}</div>
              <div class="monitor-url">{check.url}</div>
            </div>
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
  .uptime-page { padding: 1.5rem; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
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
  .monitors-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .monitor-card { display: flex; justify-content: space-between; align-items: center; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 10px; padding: 1rem 1.25rem; }
  .monitor-info { display: flex; align-items: center; gap: 0.75rem; }
  .indicator { width: 10px; height: 10px; border-radius: 50%; background: #6e7681; flex-shrink: 0; }
  .indicator.enabled { background: #3fb950; }
  .monitor-name { font-weight: 600; color: var(--text-primary); font-size: 0.9rem; }
  .monitor-url { font-size: 0.8rem; color: var(--text-muted); }
  .monitor-meta { display: flex; align-items: center; gap: 1rem; }
  .interval { font-size: 0.8rem; color: var(--text-muted); }
  .btn-delete { padding: 0.3rem 0.7rem; background: none; border: 1px solid var(--border); color: var(--text-muted); border-radius: 6px; cursor: pointer; font-size: 0.75rem; }
  .btn-delete:hover { border-color: #f85149; color: #f85149; }
</style>
