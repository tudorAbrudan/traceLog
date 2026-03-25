<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api';
  import ServerCard from '../components/ServerCard.svelte';

  let servers: any[] = [];
  let health: any = {};
  let loading = true;

  onMount(async () => {
    try {
      [servers, health] = await Promise.all([
        api.listServers(),
        api.health(),
      ]);
    } catch (e) {
      console.error('Failed to load data:', e);
    } finally {
      loading = false;
    }

    // Auto-refresh every 10s
    const interval = setInterval(async () => {
      try {
        servers = await api.listServers();
      } catch {}
    }, 10000);

    return () => clearInterval(interval);
  });
</script>

<div class="overview">
  <div class="header">
    <h2>Servers</h2>
    <button class="btn-add" on:click={() => {}}>+ Add Server</button>
  </div>

  {#if loading}
    <div class="loading">Loading...</div>
  {:else if servers === null || servers.length === 0}
    <div class="empty">
      <div class="empty-icon">📡</div>
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
  .overview {
    padding: 1.5rem;
  }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }
  h2 {
    font-size: 1.4rem;
    color: var(--text-primary);
    margin: 0;
  }
  .btn-add {
    padding: 0.5rem 1rem;
    background: #238636;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
  }
  .btn-add:hover { background: #2ea043; }
  .server-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 1rem;
  }
  .loading, .empty {
    text-align: center;
    padding: 4rem 2rem;
    color: var(--text-muted);
  }
  .empty-icon { font-size: 3rem; margin-bottom: 1rem; }
  .empty h3 { color: var(--text-primary); margin-bottom: 0.5rem; }
  .empty p { font-size: 0.9rem; }
</style>
