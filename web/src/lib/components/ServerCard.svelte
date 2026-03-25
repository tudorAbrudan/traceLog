<script lang="ts">
  import { currentPage } from '../store';

  export let server: any;

  function openDetail() {
    currentPage.set(`server:${server.id}`);
  }

  function timeSince(dateStr: string): string {
    if (!dateStr) return 'Never';
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
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

  <div class="meta">
    <span>Last seen: {timeSince(server.last_seen_at)}</span>
  </div>
</button>

<style>
  .card {
    display: block;
    width: 100%;
    text-align: left;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.25rem;
    cursor: pointer;
    transition: border-color 0.15s, background 0.15s;
    color: inherit;
    font: inherit;
  }
  .card:hover {
    border-color: #58a6ff44;
    background: var(--bg-hover);
  }
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }
  .name-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #6e7681;
    flex-shrink: 0;
  }
  .indicator.online { background: #3fb950; }
  h3 { font-size: 1rem; margin: 0; color: var(--text-primary); }
  .status {
    padding: 2px 8px;
    border-radius: 20px;
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    background: #6e768133;
    color: #8b949e;
  }
  .status.online { background: #23863633; color: #3fb950; }
  .host {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-bottom: 0.5rem;
  }
  .meta {
    font-size: 0.75rem;
    color: var(--text-muted);
  }
</style>
