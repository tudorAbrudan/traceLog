<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from '../api';
  import ServerCard from '../components/ServerCard.svelte';
  import { currentPage, suppressSingleServerAutoOpen } from '../store';

  let servers: any[] = [];
  let loading = true;
  let showForm = false;
  let newName = '';
  let newHost = '';

  /** Only decide auto-open once per Overview mount; never from the 10s poll (avoids surprise redirects). */
  let autoNavResolved = false;

  onMount(async () => {
    await loadServers();

    const interval = setInterval(loadServers, 10000);
    return () => clearInterval(interval);
  });

  async function loadServers() {
    try {
      const list = (await api.listServers()) || [];
      servers = list;
      if (!autoNavResolved) {
        autoNavResolved = true;
        if (list.length === 1 && !get(suppressSingleServerAutoOpen)) {
          currentPage.set('server:' + list[0].id);
        }
      }
    } catch {
      /* keep autoNavResolved false so a later poll can still auto-open after a failed first fetch */
    } finally {
      loading = false;
    }
  }

  async function addServer() {
    if (!newName) return;
    try {
      await api.createServer(newName, newHost);
      newName = ''; newHost = ''; showForm = false;
      await loadServers();
      if (servers.length === 1 && !get(suppressSingleServerAutoOpen)) {
        currentPage.set('server:' + servers[0].id);
      }
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }
</script>

<div class="overview">
  <div class="header">
    <h2>Servers</h2>
    <button class="btn-add" on:click={() => showForm = !showForm}>
      {showForm ? 'Cancel' : '+ Add Server'}
    </button>
  </div>

  {#if showForm}
    <div class="form-card">
      <div class="form-row">
        <div class="field">
          <label for="srv-name">Name</label>
          <input id="srv-name" type="text" bind:value={newName} placeholder="web-server-1" />
        </div>
        <div class="field">
          <label for="srv-host">Host</label>
          <input id="srv-host" type="text" bind:value={newHost} placeholder="10.0.1.5" />
        </div>
        <button class="btn-save" on:click={addServer}>Create</button>
      </div>
      <p class="hint">After creating, use the API key to connect a remote agent.</p>
    </div>
  {/if}

  {#if loading}
    <div class="loading">Loading...</div>
  {:else if servers.length === 0}
    <div class="empty">
      <h3>No servers yet</h3>
      <p>Your local server metrics are being collected. They'll appear here shortly.</p>
    </div>
  {:else}
    <div class="server-grid">
      {#each servers as server (server.id)}
        <ServerCard {server} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .overview { padding: 1.5rem; }
  .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0; }
  .btn-add { padding: 0.5rem 1rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-size: 0.85rem; font-weight: 600; }
  .btn-add:hover { background: #2ea043; }
  .form-card { background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px; padding: 1.25rem; margin-bottom: 1rem; }
  .form-row { display: flex; gap: 0.75rem; align-items: flex-end; flex-wrap: wrap; }
  .field { display: flex; flex-direction: column; gap: 0.25rem; }
  .field label { font-size: 0.8rem; color: var(--text-secondary); }
  .field input { padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 0.85rem; outline: none; }
  .field input:focus { border-color: #58a6ff; }
  .btn-save { padding: 0.5rem 1.25rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-weight: 600; font-size: 0.85rem; }
  .hint { font-size: 0.8rem; color: var(--text-muted); margin-top: 0.75rem; }
  .server-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1rem; }
  .loading, .empty { text-align: center; padding: 4rem 2rem; color: var(--text-muted); }
  .empty h3 { color: var(--text-primary); margin-bottom: 0.5rem; }
</style>
