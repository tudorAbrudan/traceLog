<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from '../api';
  import ServerCard from '../components/ServerCard.svelte';
  import { contextServerId, currentPage, suppressSingleServerAutoOpen } from '../store';
  import LoadingState from '../components/LoadingState.svelte';

  /** One auto-jump to the lone server per tab session; blocks stray Overview ticks from changing the route again. */
  const SINGLE_SERVER_AUTO_NAV_KEY = 'tracelog-single-server-auto-nav-done';

  let servers: any[] = [];
  let loading = true;
  let listError = '';
  let showForm = false;
  let newName = '';
  let newHost = '';
  let addError = '';

  /** After the first successful list fetch, auto-open runs at most once; later polls only refresh `servers`. */
  let autoNavResolved = false;

  onMount(() => {
    void fetchServersAndMaybeAutoOpenOnce();
    const interval = setInterval(() => {
      void fetchServersAndMaybeAutoOpenOnce();
    }, 10000);
    return () => clearInterval(interval);
  });

  async function refreshServerList(): Promise<boolean> {
    try {
      const list = (await api.listServers()) || [];
      servers = list;
      return true;
    } catch (e) {
      listError = (e as Error).message || 'Failed to load servers';
      return false;
    } finally {
      loading = false;
    }
  }

  function applySingleServerAutoOpenOnce() {
    if (autoNavResolved) return;
    autoNavResolved = true;
    if (servers.length !== 1 || get(suppressSingleServerAutoOpen)) return;
    try {
      if (sessionStorage.getItem(SINGLE_SERVER_AUTO_NAV_KEY)) return;
    } catch {
      /* private mode */
    }
    contextServerId.set(servers[0].id);
    currentPage.set('server:' + servers[0].id);
    try {
      sessionStorage.setItem(SINGLE_SERVER_AUTO_NAV_KEY, '1');
    } catch {
      /* ignore */
    }
  }

  /** Used on mount and every 10s: refresh list; only the first successful refresh may change the route. */
  async function fetchServersAndMaybeAutoOpenOnce() {
    const ok = await refreshServerList();
    if (ok) applySingleServerAutoOpenOnce();
  }

  async function addServer() {
    if (!newName) return;
    addError = '';
    try {
      await api.createServer(newName, newHost);
      newName = ''; newHost = ''; showForm = false;
      await refreshServerList();
      autoNavResolved = true;
      if (servers.length === 1 && !get(suppressSingleServerAutoOpen)) {
        contextServerId.set(servers[0].id);
        currentPage.set('server:' + servers[0].id);
        try {
          sessionStorage.setItem(SINGLE_SERVER_AUTO_NAV_KEY, '1');
        } catch {
          /* ignore */
        }
      }
    } catch (e: any) {
      addError = e.message || 'Failed to create server';
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
      {#if addError}<p class="error-msg">{addError}</p>{/if}
      <p class="hint">After creating, use the API key to connect a remote agent.</p>
    </div>
  {/if}

  <LoadingState {loading} error={listError}>
    {#if servers.length === 0}
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
  </LoadingState>
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
  .error-msg { color: var(--danger); font-size: 0.82rem; margin: 0.5rem 0 0; padding: 0.4rem 0.75rem; background: rgba(248,81,73,0.08); border: 1px solid rgba(248,81,73,0.25); border-radius: 6px; }
  .server-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 1rem; }
  .empty { text-align: center; padding: 4rem 2rem; color: var(--text-muted); }
  .empty h3 { color: var(--text-primary); margin-bottom: 0.5rem; }
</style>
