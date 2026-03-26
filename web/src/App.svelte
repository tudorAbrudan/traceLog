<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from './lib/api';
  import { user, isAuthenticated, currentPage } from './lib/store';

  import Login from './lib/pages/Login.svelte';
  import SetupWizard from './lib/pages/SetupWizard.svelte';
  import Overview from './lib/pages/Overview.svelte';
  import ServerDetail from './lib/pages/ServerDetail.svelte';
  import Logs from './lib/pages/Logs.svelte';
  import Uptime from './lib/pages/Uptime.svelte';
  import Settings from './lib/pages/Settings.svelte';
  import Processes from './lib/pages/Processes.svelte';
  import HttpAnalytics from './lib/pages/HttpAnalytics.svelte';
  import Sidebar from './lib/components/Sidebar.svelte';

  let checking = true;
  let needsSetup = false;

  onMount(async () => {
    try {
      const health = await api.health();
      if (!health.setup_done) {
        needsSetup = true;
        checking = false;
        return;
      }
    } catch {}

    try {
      const res = await api.me();
      if (res.user) {
        user.set(res.user);
        isAuthenticated.set(true);
        api.setCsrfToken(res.csrf_token);
      }
    } catch {
      // Not authenticated
    } finally {
      checking = false;
    }
  });

  function getServerId(page: string): string {
    return page.startsWith('server:') ? page.slice(7) : '';
  }
</script>

{#if checking}
  <div class="loading-screen">
    <div class="spinner"></div>
  </div>
{:else if needsSetup}
  <SetupWizard />
{:else if !$isAuthenticated}
  <Login />
{:else}
  <div class="app-layout">
    <Sidebar />
    <main class="main-content">
      {#if $currentPage === 'overview'}
        <Overview />
      {:else if $currentPage.startsWith('server:')}
        <ServerDetail serverId={getServerId($currentPage)} />
      {:else if $currentPage === 'logs'}
        <Logs />
      {:else if $currentPage === 'processes'}
        <Processes />
      {:else if $currentPage === 'http'}
        <HttpAnalytics />
      {:else if $currentPage === 'uptime'}
        <Uptime />
      {:else if $currentPage === 'settings'}
        <Settings />
      {:else}
        <Overview />
      {/if}
    </main>
  </div>
{/if}

<style>
  .loading-screen {
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-primary);
  }
  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border);
    border-top-color: #58a6ff;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .app-layout {
    display: flex;
    min-height: 100vh;
    background: var(--bg-primary);
  }
  .main-content {
    flex: 1;
    margin-left: 220px;
    min-height: 100vh;
  }
</style>
