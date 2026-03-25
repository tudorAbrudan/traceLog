<script lang="ts">
  import { user } from '../store';
  import { api } from '../api';

  let activeTab = 'general';
  let retentionDays = 30;
  let collectionInterval = 10;
  let saved = false;

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'logs', label: 'Log Sources' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'servers', label: 'Servers' },
    { id: 'alerts', label: 'Alerts' },
    { id: 'account', label: 'Account' },
    { id: 'about', label: 'About' },
  ];

  async function saveGeneral() {
    try {
      await api.updateSettings({ retention_days: retentionDays, collection_interval: collectionInterval });
      saved = true;
      setTimeout(() => saved = false, 2000);
    } catch (e: any) {
      alert('Save failed: ' + e.message);
    }
  }
</script>

<div class="settings">
  <h2>Settings</h2>

  <div class="settings-layout">
    <nav class="settings-nav">
      {#each tabs as tab}
        <button class:active={activeTab === tab.id} on:click={() => activeTab = tab.id}>
          {tab.label}
        </button>
      {/each}
    </nav>

    <div class="settings-content">
      {#if activeTab === 'general'}
        <div class="section">
          <h3>Data Retention</h3>
          <div class="field">
            <label for="retention">Keep data for</label>
            <div class="range-input">
              <input id="retention" type="range" min="1" max="30" bind:value={retentionDays} />
              <span>{retentionDays} days</span>
            </div>
          </div>
          <div class="field">
            <label for="interval">Collection interval</label>
            <select id="interval" bind:value={collectionInterval}>
              <option value={5}>5 seconds</option>
              <option value={10}>10 seconds</option>
              <option value={30}>30 seconds</option>
              <option value={60}>60 seconds</option>
            </select>
          </div>
          <button class="btn-save" on:click={saveGeneral}>
            {saved ? '✓ Saved' : 'Save Changes'}
          </button>
        </div>

      {:else if activeTab === 'logs'}
        <div class="section">
          <h3>Log Sources</h3>
          <p class="hint">Configure which log files and Docker containers to monitor.</p>
          <button class="btn-scan">Scan for common log files</button>
          <div class="placeholder">Log source management UI coming soon.</div>
        </div>

      {:else if activeTab === 'notifications'}
        <div class="section">
          <h3>Notification Channels</h3>
          <p class="hint">Configure email (SMTP) and webhook notifications.</p>
          <div class="placeholder">Notification channel setup coming in Phase 3.</div>
        </div>

      {:else if activeTab === 'servers'}
        <div class="section">
          <h3>Connected Servers</h3>
          <p class="hint">Manage remote agents and generate API keys.</p>
          <div class="placeholder">Server management UI coming soon.</div>
        </div>

      {:else if activeTab === 'alerts'}
        <div class="section">
          <h3>Alert Rules</h3>
          <p class="hint">Default rules: CPU &gt; 90%, RAM &gt; 90%, Disk &gt; 90% (disabled by default).</p>
          <div class="placeholder">Alert rules configuration coming in Phase 3.</div>
        </div>

      {:else if activeTab === 'account'}
        <div class="section">
          <h3>Account</h3>
          <p class="hint">Logged in as: <strong>{$user?.username}</strong></p>
          <p class="hint">To change password, run: <code>tracelog user reset-password {$user?.username}</code></p>
        </div>

      {:else if activeTab === 'about'}
        <div class="section">
          <h3>About TraceLog</h3>
          <div class="about-grid">
            <div class="about-item"><span>Version</span><strong>dev</strong></div>
            <div class="about-item"><span>License</span><strong>MIT</strong></div>
            <div class="about-item"><span>GitHub</span><a href="https://github.com/tudorAbrudan/tracelog" target="_blank">tudorAbrudan/tracelog</a></div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .settings { padding: 1.5rem; }
  h2 { font-size: 1.4rem; color: var(--text-primary); margin: 0 0 1.5rem 0; }
  .settings-layout { display: flex; gap: 1.5rem; }
  .settings-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 160px;
  }
  .settings-nav button {
    padding: 0.5rem 0.75rem;
    background: none;
    border: none;
    color: var(--text-secondary);
    text-align: left;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.85rem;
  }
  .settings-nav button:hover { background: var(--bg-secondary); }
  .settings-nav button.active {
    background: var(--bg-secondary);
    color: var(--text-primary);
    font-weight: 600;
  }
  .settings-content {
    flex: 1;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.5rem;
  }
  .section h3 { font-size: 1rem; margin: 0 0 0.75rem 0; color: var(--text-primary); }
  .field { margin-bottom: 1rem; }
  .field label { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.3rem; }
  .range-input { display: flex; align-items: center; gap: 0.75rem; }
  .range-input input { flex: 1; }
  .range-input span { min-width: 60px; font-size: 0.85rem; color: var(--text-primary); }
  select {
    padding: 0.5rem;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 0.85rem;
  }
  .btn-save {
    padding: 0.5rem 1.25rem;
    background: #238636;
    color: #fff;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-weight: 600;
    font-size: 0.85rem;
  }
  .btn-scan {
    padding: 0.5rem 1rem;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }
  .hint { color: var(--text-muted); font-size: 0.85rem; margin-bottom: 1rem; }
  code { background: var(--bg-primary); padding: 2px 6px; border-radius: 4px; font-size: 0.8rem; }
  .placeholder { color: var(--text-muted); font-style: italic; font-size: 0.85rem; padding: 2rem; text-align: center; }
  .about-grid { display: flex; flex-direction: column; gap: 0.75rem; }
  .about-item { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
  .about-item span { color: var(--text-muted); }
  .about-item strong, .about-item a { color: var(--text-primary); }

  @media (max-width: 768px) {
    .settings-layout { flex-direction: column; }
    .settings-nav { flex-direction: row; overflow-x: auto; min-width: auto; }
  }
</style>
