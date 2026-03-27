<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from './lib/api';
  import { user, isAuthenticated, currentPage, navDrawerOpen } from './lib/store';

  const PAGE_STORAGE_KEY = 'tracelog-current-page';
  const KNOWN_TOP_PAGES = new Set(['overview', 'processes', 'logs', 'http', 'uptime', 'settings']);

  function isPersistablePage(v: string): boolean {
    if (KNOWN_TOP_PAGES.has(v)) return true;
    return /^server:[a-zA-Z0-9_-]{4,128}$/.test(v);
  }

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

  onMount(() => {
    const mq = window.matchMedia('(min-width: 901px)');
    const closeDrawer = () => {
      if (mq.matches) navDrawerOpen.set(false);
    };
    mq.addEventListener('change', closeDrawer);
    closeDrawer();

    const unsubPersist = currentPage.subscribe((v) => {
      if (get(isAuthenticated) && isPersistablePage(v)) {
        try {
          sessionStorage.setItem(PAGE_STORAGE_KEY, v);
        } catch {
          /* ignore quota / private mode */
        }
      }
    });
    const unsubAuth = isAuthenticated.subscribe((auth) => {
      if (!auth) return;
      try {
        const raw = sessionStorage.getItem(PAGE_STORAGE_KEY);
        if (raw && isPersistablePage(raw)) {
          currentPage.set(raw);
        }
      } catch {
        /* ignore */
      }
    });

    void (async () => {
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
        /* not authenticated */
      } finally {
        checking = false;
      }
    })();

    return () => {
      mq.removeEventListener('change', closeDrawer);
      unsubPersist();
      unsubAuth();
    };
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
  <div class="app-layout" class:drawer-open={$navDrawerOpen}>
    <button
      type="button"
      class="mobile-menu-btn"
      aria-label="Open navigation menu"
      aria-expanded={$navDrawerOpen}
      on:click={() => navDrawerOpen.update((o) => !o)}
    >
      ☰
    </button>
    <button
      type="button"
      class="drawer-backdrop"
      aria-label="Close menu"
      tabindex="-1"
      on:click={() => navDrawerOpen.set(false)}
    ></button>
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
    position: relative;
  }
  .main-content {
    flex: 1;
    margin-left: 220px;
    min-height: 100vh;
    min-width: 0;
  }
  .mobile-menu-btn {
    display: none;
    position: fixed;
    top: 10px;
    left: 10px;
    z-index: 300;
    width: 44px;
    height: 44px;
    padding: 0;
    align-items: center;
    justify-content: center;
    border: 1px solid var(--border);
    border-radius: 10px;
    background: var(--bg-secondary);
    color: var(--text-primary);
    font-size: 1.25rem;
    line-height: 1;
    cursor: pointer;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  }
  .drawer-backdrop {
    display: none;
    position: fixed;
    inset: 0;
    z-index: 199;
    border: none;
    padding: 0;
    margin: 0;
    background: rgba(0, 0, 0, 0.45);
    cursor: pointer;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.2s ease;
  }
  .app-layout.drawer-open .drawer-backdrop {
    opacity: 1;
    pointer-events: auto;
  }
  @media (max-width: 900px) {
    .mobile-menu-btn {
      display: flex;
    }
    .drawer-backdrop {
      display: block;
    }
    .main-content {
      margin-left: 0;
      padding-top: 3.5rem;
      padding-left: 0.75rem;
      padding-right: 0.75rem;
      padding-bottom: 1rem;
    }
  }
</style>
