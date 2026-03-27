<script lang="ts">
  import { currentPage, user, darkMode, suppressSingleServerAutoOpen, navDrawerOpen } from '../store';
  import { api } from '../api';

  function toggleTheme() {
    darkMode.update(d => {
      const next = !d;
      document.documentElement.setAttribute('data-theme', next ? 'dark' : 'light');
      localStorage.setItem('tracelog-theme', next ? 'dark' : 'light');
      return next;
    });
  }

  const navItems = [
    { id: 'overview', label: 'Overview', icon: '◉' },
    { id: 'processes', label: 'Processes', icon: '⊞' },
    { id: 'logs', label: 'Logs', icon: '☰' },
    { id: 'http', label: 'HTTP Analytics', icon: '⇄' },
    { id: 'uptime', label: 'Uptime', icon: '↑' },
    { id: 'settings', label: 'Settings', icon: '⚙' },
  ];

  async function handleLogout() {
    try {
      await api.logout();
    } catch {}
    try {
      sessionStorage.removeItem('tracelog-current-page');
    } catch {
      /* ignore */
    }
    window.location.reload();
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <h1>TraceLog</h1>
  </div>

  <nav>
    {#each navItems as item}
      <button
        class:active={$currentPage === item.id}
        on:click={() => {
          suppressSingleServerAutoOpen.set(true);
          currentPage.set(item.id);
        }}
      >
        <span class="icon">{item.icon}</span>
        <span>{item.label}</span>
      </button>
    {/each}
  </nav>

  <div class="sidebar-footer">
    <div class="user-info">
      <span class="user-avatar">{$user?.username?.charAt(0).toUpperCase()}</span>
      <span class="username">{$user?.username}</span>
      <button class="theme-toggle" on:click={toggleTheme} title="Toggle theme">
        {$darkMode ? '☀' : '◑'}
      </button>
    </div>
    <button class="logout" on:click={handleLogout}>Sign out</button>
  </div>
</aside>

<style>
  .sidebar {
    width: 220px;
    height: 100dvh;
    min-height: 100vh;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    position: fixed;
    left: 0;
    top: 0;
    z-index: 200;
    transition: transform 0.2s ease;
  }
  .sidebar-header {
    padding: 1.25rem 1rem;
    border-bottom: 1px solid var(--border);
  }
  .sidebar-header h1 {
    font-size: 1.2rem;
    margin: 0;
    background: linear-gradient(135deg, #58a6ff, #bc8cff);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }
  nav {
    flex: 1;
    padding: 0.75rem 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  nav button {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.55rem 0.75rem;
    background: none;
    border: none;
    color: var(--text-secondary);
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    text-align: left;
    transition: background 0.1s;
  }
  nav button:hover { background: var(--bg-hover); }
  nav button.active {
    background: var(--bg-hover);
    color: var(--text-primary);
    font-weight: 600;
  }
  .icon { font-size: 1rem; width: 20px; text-align: center; }
  .sidebar-footer {
    padding: 0.75rem;
    border-top: 1px solid var(--border);
  }
  .user-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0.25rem 0.5rem;
  }
  .user-avatar {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: #58a6ff33;
    color: #58a6ff;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 700;
  }
  .username { font-size: 0.8rem; color: var(--text-primary); flex: 1; }
  .theme-toggle {
    background: none; border: 1px solid var(--border); color: var(--text-muted);
    border-radius: 6px; width: 28px; height: 28px; cursor: pointer; font-size: 0.9rem;
    display: flex; align-items: center; justify-content: center; transition: all 0.15s;
  }
  .theme-toggle:hover { background: var(--bg-hover); color: var(--text-primary); }
  .logout {
    width: 100%;
    padding: 0.4rem;
    background: none;
    border: 1px solid var(--border);
    color: var(--text-muted);
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.75rem;
  }
  .logout:hover { background: var(--bg-hover); color: #f85149; border-color: #f8514944; }

  @media (max-width: 900px) {
    .sidebar {
      width: min(280px, 88vw);
      transform: translateX(-100%);
      box-shadow: 8px 0 32px rgba(0, 0, 0, 0.25);
    }
    :global(.app-layout.drawer-open) .sidebar {
      transform: translateX(0);
    }
    nav button {
      min-height: 44px;
      font-size: 0.9rem;
    }
    .logout {
      min-height: 44px;
    }
  }
</style>
