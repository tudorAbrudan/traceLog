<script lang="ts">
  import { api } from '../api';
  import General from './Settings/General.svelte';
  import Logs from './Settings/Logs.svelte';
  import Notifications from './Settings/Notifications.svelte';
  import Servers from './Settings/Servers.svelte';
  import Alerts from './Settings/Alerts.svelte';
  import Account from './Settings/Account.svelte';
  import About from './Settings/About.svelte';

  let activeTab = 'general';

  // Shared state passed to multiple tabs
  let servers: any[] = [];
  let channels: any[] = [];

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'logs', label: 'Log Sources' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'servers', label: 'Servers' },
    { id: 'alerts', label: 'Alerts' },
    { id: 'account', label: 'Account' },
    { id: 'about', label: 'About' },
  ];

  async function loadTab(tab: string) {
    activeTab = tab;
    try {
      if (tab === 'logs') {
        if (servers.length === 0) {
          servers = (await api.listServers()) || [];
        }
      }
      if (tab === 'notifications') channels = (await api.listNotificationChannels()) || [];
      if (tab === 'servers') servers = (await api.listServers()) || [];
      if (tab === 'alerts') {
        channels = (await api.listNotificationChannels()) || [];
        servers = (await api.listServers()) || [];
      }
    } catch {}
  }

  async function reloadChannels() {
    try {
      channels = (await api.listNotificationChannels()) || [];
    } catch {}
  }

  async function reloadServers() {
    try {
      servers = (await api.listServers()) || [];
    } catch {}
  }
</script>

<div class="settings">
  <h2>Settings</h2>

  <div class="settings-layout">
    <nav class="settings-nav">
      {#each tabs as tab}
        <button class:active={activeTab === tab.id} on:click={() => loadTab(tab.id)}>{tab.label}</button>
      {/each}
    </nav>

    <div class="settings-content">
      {#if activeTab === 'general'}
        <General />
      {:else if activeTab === 'logs'}
        <Logs {servers} />
      {:else if activeTab === 'notifications'}
        <Notifications on:channelsChanged={reloadChannels} />
      {:else if activeTab === 'servers'}
        <Servers {servers} on:serversChanged={reloadServers} />
      {:else if activeTab === 'alerts'}
        <Alerts {servers} {channels} />
      {:else if activeTab === 'account'}
        <Account />
      {:else if activeTab === 'about'}
        <About />
      {/if}
    </div>
  </div>
</div>

<style>
  /* Shell-only: layout and nav */
  .settings { padding: 1.5rem; }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0 0 1.5rem 0; }
  .settings-layout { display: flex; gap: 1.5rem; }
  .settings-nav { display: flex; flex-direction: column; gap: 2px; min-width: 160px; }
  .settings-nav button { padding: 0.5rem 0.75rem; background: none; border: none; color: var(--text-secondary); text-align: left; border-radius: 6px; cursor: pointer; font-size: 0.85rem; }
  .settings-nav button:hover { background: var(--bg-secondary); }
  .settings-nav button.active { background: var(--bg-secondary); color: var(--text-primary); font-weight: 600; }
  .settings-content { flex: 1; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; }

  /* Shared styles used by sub-components rendered inside .settings-content */
  :global(.section h3) { font-size: 1rem; margin: 0 0 0.75rem 0; color: var(--text-primary); }
  :global(select), :global(.add-form input), :global(.add-form textarea) { padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 0.85rem; }
  :global(.add-form textarea) { width: 100%; min-height: 60px; font-family: monospace; font-size: 0.8rem; resize: vertical; }
  :global(.add-form) { display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap; align-items: flex-end; }
  :global(.alert-form) { align-items: center; }
  :global(.hint-inline) { font-size: 0.8rem; color: var(--text-muted); }
  :global(.btn-save) { padding: 0.5rem 1.25rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-weight: 600; font-size: 0.85rem; }
  :global(.btn-secondary) { padding: 0.5rem 1rem; background: var(--bg-primary); border: 1px solid var(--border); color: var(--text-primary); border-radius: 8px; cursor: pointer; font-size: 0.85rem; margin-bottom: 1rem; }
  :global(.hint) { color: var(--text-muted); font-size: 0.85rem; margin-bottom: 1rem; }
  :global(code) { background: var(--bg-primary); padding: 2px 6px; border-radius: 4px; font-size: 0.8rem; }
  :global(.item-list) { display: flex; flex-direction: column; gap: 0.5rem; }
  :global(.item-row) { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 8px; }
  :global(.item-row strong) { color: var(--text-primary); font-size: 0.85rem; }
  :global(.item-detail) { display: block; font-size: 0.75rem; color: var(--text-muted); }
  :global(.item-actions) { display: flex; gap: 0.5rem; flex-wrap: wrap; }
  :global(.btn-delete) { padding: 0.3rem 0.7rem; background: none; border: 1px solid var(--border); color: var(--text-muted); border-radius: 6px; cursor: pointer; font-size: 0.75rem; }
  :global(.btn-delete:hover) { border-color: #f85149; color: #f85149; }

  @media (max-width: 900px) {
    .settings { padding: 1rem 0.5rem; }
    .settings-layout { flex-direction: column; gap: 0.75rem; }
    .settings-nav { flex-direction: row; flex-wrap: wrap; overflow-x: auto; min-width: auto; }
    .settings-nav button { font-size: 0.78rem; padding: 0.45rem 0.55rem; }
    .settings-content { padding: 1rem; }
    :global(.item-row) { flex-direction: column; align-items: flex-start; gap: 0.5rem; }
  }
</style>
