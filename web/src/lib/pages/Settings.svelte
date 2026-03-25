<script lang="ts">
  import { onMount } from 'svelte';
  import { user } from '../store';
  import { api } from '../api';

  let activeTab = 'general';
  let retentionDays = 30;
  let collectionInterval = 10;
  let saved = false;

  // Log sources
  let logSources: any[] = [];
  let newLogName = ''; let newLogPath = ''; let newLogFormat = 'plain';

  // Notifications
  let channels: any[] = [];
  let newChName = ''; let newChType = 'email'; let newChConfig = '';

  // Servers
  let servers: any[] = [];

  // Alerts
  let alertRules: any[] = [];
  let newAlertMetric = 'cpu_percent'; let newAlertOp = '>'; let newAlertThreshold = 90;
  let newAlertDuration = 300; let newAlertChannel = '';

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'logs', label: 'Log Sources' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'servers', label: 'Servers' },
    { id: 'alerts', label: 'Alerts' },
    { id: 'account', label: 'Account' },
    { id: 'about', label: 'About' },
  ];

  const metrics = ['cpu_percent', 'mem_percent', 'disk_percent', 'load_1', 'load_5', 'load_15'];

  onMount(async () => {
    try {
      const s = await api.getSettings();
      retentionDays = parseInt(s.retention_days) || 30;
      collectionInterval = parseInt(s.collection_interval) || 10;
    } catch {}
  });

  async function loadTab(tab: string) {
    activeTab = tab;
    try {
      if (tab === 'logs') logSources = (await api.listLogSources()) || [];
      if (tab === 'notifications') channels = (await api.listNotificationChannels()) || [];
      if (tab === 'servers') servers = (await api.listServers()) || [];
      if (tab === 'alerts') {
        alertRules = (await api.listAlertRules()) || [];
        channels = (await api.listNotificationChannels()) || [];
      }
    } catch {}
  }

  async function saveGeneral() {
    try {
      await api.updateSettings({ retention_days: String(retentionDays), collection_interval: String(collectionInterval) });
      saved = true; setTimeout(() => saved = false, 2000);
    } catch (e: any) { alert('Save failed: ' + e.message); }
  }

  // Log sources
  async function addLogSource() {
    if (!newLogName || !newLogPath) return;
    try {
      await api.createLogSource({ name: newLogName, path: newLogPath, format: newLogFormat, type: 'file', server_id: '', enabled: true });
      newLogName = ''; newLogPath = '';
      logSources = (await api.listLogSources()) || [];
    } catch (e: any) { alert('Failed: ' + e.message); }
  }
  async function removeLogSource(id: string) {
    await api.deleteLogSource(id);
    logSources = (await api.listLogSources()) || [];
  }
  async function scanLogs() {
    try {
      const d = await api.detect();
      if (d.log_files && d.log_files.length > 0) {
        for (const lf of d.log_files) {
          await api.createLogSource({ name: lf.name, path: lf.path, format: lf.format, type: lf.type, server_id: '', enabled: true });
        }
        logSources = (await api.listLogSources()) || [];
        alert(`Found and added ${d.log_files.length} log sources.`);
      } else {
        alert('No common log files found on this system.');
      }
    } catch (e: any) { alert('Scan failed: ' + e.message); }
  }

  // Notification channels
  async function addChannel() {
    if (!newChName || !newChConfig) return;
    try {
      await api.createNotificationChannel({ name: newChName, type: newChType, config: newChConfig });
      newChName = ''; newChConfig = '';
      channels = (await api.listNotificationChannels()) || [];
    } catch (e: any) { alert('Failed: ' + e.message); }
  }
  async function removeChannel(id: string) {
    await api.deleteNotificationChannel(id);
    channels = (await api.listNotificationChannels()) || [];
  }
  async function testChannel(id: string) {
    try {
      await api.testNotificationChannel(id);
      alert('Test notification sent!');
    } catch (e: any) { alert('Test failed: ' + e.message); }
  }

  // Servers
  async function removeServer(id: string) {
    if (!confirm('Delete this server and all its data?')) return;
    await api.deleteServer(id);
    servers = (await api.listServers()) || [];
  }

  // Alerts
  async function addAlert() {
    try {
      await api.createAlertRule({
        metric: newAlertMetric, operator: newAlertOp, threshold: newAlertThreshold,
        duration_seconds: newAlertDuration, cooldown_seconds: 1800, channel_id: newAlertChannel,
        server_id: '', enabled: true,
      });
      alertRules = (await api.listAlertRules()) || [];
    } catch (e: any) { alert('Failed: ' + e.message); }
  }
  async function removeAlert(id: string) {
    await api.deleteAlertRule(id);
    alertRules = (await api.listAlertRules()) || [];
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
          <button class="btn-save" on:click={saveGeneral}>{saved ? '✓ Saved' : 'Save Changes'}</button>
        </div>

      {:else if activeTab === 'logs'}
        <div class="section">
          <h3>Log Sources</h3>
          <button class="btn-secondary" on:click={scanLogs}>Scan for common log files</button>
          <div class="add-form">
            <input type="text" bind:value={newLogName} placeholder="Name" />
            <input type="text" bind:value={newLogPath} placeholder="/var/log/..." />
            <select bind:value={newLogFormat}>
              <option value="plain">Plain</option>
              <option value="nginx">Nginx</option>
              <option value="apache">Apache</option>
            </select>
            <button class="btn-save" on:click={addLogSource}>Add</button>
          </div>
          {#if logSources.length === 0}
            <p class="hint">No log sources configured. Click "Scan" or add manually above.</p>
          {:else}
            <div class="item-list">
              {#each logSources as ls (ls.id)}
                <div class="item-row">
                  <div>
                    <strong>{ls.name}</strong>
                    <span class="item-detail">{ls.path || ls.container} ({ls.format})</span>
                  </div>
                  <button class="btn-delete" on:click={() => removeLogSource(ls.id)}>Delete</button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'notifications'}
        <div class="section">
          <h3>Notification Channels</h3>
          <div class="add-form">
            <input type="text" bind:value={newChName} placeholder="Channel name" />
            <select bind:value={newChType}>
              <option value="email">Email (SMTP)</option>
              <option value="webhook">Webhook</option>
            </select>
            <textarea bind:value={newChConfig} placeholder={newChType === 'email'
              ? '{"host":"smtp.example.com","port":587,"username":"...","password":"...","from":"...","to":"...","use_tls":true}'
              : '{"url":"https://hooks.slack.com/...","method":"POST"}'}
            ></textarea>
            <button class="btn-save" on:click={addChannel}>Add Channel</button>
          </div>
          {#if channels.length === 0}
            <p class="hint">No notification channels configured.</p>
          {:else}
            <div class="item-list">
              {#each channels as ch (ch.id)}
                <div class="item-row">
                  <div>
                    <strong>{ch.name}</strong>
                    <span class="item-detail">{ch.type}</span>
                  </div>
                  <div class="item-actions">
                    <button class="btn-secondary" on:click={() => testChannel(ch.id)}>Test</button>
                    <button class="btn-delete" on:click={() => removeChannel(ch.id)}>Delete</button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'servers'}
        <div class="section">
          <h3>Connected Servers</h3>
          {#if servers.length === 0}
            <p class="hint">No servers registered.</p>
          {:else}
            <div class="item-list">
              {#each servers as srv (srv.id)}
                <div class="item-row">
                  <div>
                    <strong>{srv.name}</strong>
                    <span class="item-detail">{srv.host} — {srv.status}</span>
                    <span class="item-detail api-key">API Key: {srv.api_key}</span>
                  </div>
                  <button class="btn-delete" on:click={() => removeServer(srv.id)}>Delete</button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'alerts'}
        <div class="section">
          <h3>Alert Rules</h3>
          <div class="add-form alert-form">
            <select bind:value={newAlertMetric}>
              {#each metrics as m}<option value={m}>{m}</option>{/each}
            </select>
            <select bind:value={newAlertOp}>
              <option value=">">{'>'}</option>
              <option value=">=">{'>='}</option>
              <option value="<">{'<'}</option>
            </select>
            <input type="number" bind:value={newAlertThreshold} min="0" max="100" style="width:80px" />
            <span class="hint-inline">for</span>
            <select bind:value={newAlertDuration}>
              <option value={60}>1 min</option>
              <option value={300}>5 min</option>
              <option value={600}>10 min</option>
            </select>
            <span class="hint-inline">notify</span>
            <select bind:value={newAlertChannel}>
              <option value="">None</option>
              {#each channels as ch}<option value={ch.id}>{ch.name}</option>{/each}
            </select>
            <button class="btn-save" on:click={addAlert}>Add Rule</button>
          </div>
          {#if alertRules.length === 0}
            <p class="hint">No alert rules configured. Add one above.</p>
          {:else}
            <div class="item-list">
              {#each alertRules as rule (rule.id)}
                <div class="item-row">
                  <div>
                    <strong>{rule.metric} {rule.operator} {rule.threshold}</strong>
                    <span class="item-detail">Duration: {rule.duration_seconds}s | Cooldown: {rule.cooldown_seconds}s</span>
                  </div>
                  <button class="btn-delete" on:click={() => removeAlert(rule.id)}>Delete</button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'account'}
        <div class="section">
          <h3>Account</h3>
          <p class="hint">Logged in as: <strong>{$user?.username}</strong></p>
          <p class="hint">To change password: <code>tracelog user reset-password {$user?.username}</code></p>
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
  .settings-nav { display: flex; flex-direction: column; gap: 2px; min-width: 160px; }
  .settings-nav button { padding: 0.5rem 0.75rem; background: none; border: none; color: var(--text-secondary); text-align: left; border-radius: 6px; cursor: pointer; font-size: 0.85rem; }
  .settings-nav button:hover { background: var(--bg-secondary); }
  .settings-nav button.active { background: var(--bg-secondary); color: var(--text-primary); font-weight: 600; }
  .settings-content { flex: 1; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 12px; padding: 1.5rem; }
  .section h3 { font-size: 1rem; margin: 0 0 0.75rem 0; color: var(--text-primary); }
  .field { margin-bottom: 1rem; }
  .field label { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.3rem; }
  .range-input { display: flex; align-items: center; gap: 0.75rem; }
  .range-input input { flex: 1; }
  .range-input span { min-width: 60px; font-size: 0.85rem; color: var(--text-primary); }
  select, .add-form input, .add-form textarea { padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px; color: var(--text-primary); font-size: 0.85rem; }
  .add-form textarea { width: 100%; min-height: 60px; font-family: monospace; font-size: 0.8rem; resize: vertical; }
  .add-form { display: flex; gap: 0.5rem; margin-bottom: 1rem; flex-wrap: wrap; align-items: flex-end; }
  .alert-form { align-items: center; }
  .hint-inline { font-size: 0.8rem; color: var(--text-muted); }
  .btn-save { padding: 0.5rem 1.25rem; background: #238636; color: #fff; border: none; border-radius: 8px; cursor: pointer; font-weight: 600; font-size: 0.85rem; }
  .btn-secondary { padding: 0.5rem 1rem; background: var(--bg-primary); border: 1px solid var(--border); color: var(--text-primary); border-radius: 8px; cursor: pointer; font-size: 0.85rem; margin-bottom: 1rem; }
  .hint { color: var(--text-muted); font-size: 0.85rem; margin-bottom: 1rem; }
  code { background: var(--bg-primary); padding: 2px 6px; border-radius: 4px; font-size: 0.8rem; }
  .item-list { display: flex; flex-direction: column; gap: 0.5rem; }
  .item-row { display: flex; justify-content: space-between; align-items: center; padding: 0.75rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 8px; }
  .item-row strong { color: var(--text-primary); font-size: 0.85rem; }
  .item-detail { display: block; font-size: 0.75rem; color: var(--text-muted); }
  .api-key { font-family: monospace; font-size: 0.7rem; }
  .item-actions { display: flex; gap: 0.5rem; }
  .btn-delete { padding: 0.3rem 0.7rem; background: none; border: 1px solid var(--border); color: var(--text-muted); border-radius: 6px; cursor: pointer; font-size: 0.75rem; }
  .btn-delete:hover { border-color: #f85149; color: #f85149; }
  .about-grid { display: flex; flex-direction: column; gap: 0.75rem; }
  .about-item { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
  .about-item span { color: var(--text-muted); }
  .about-item strong, .about-item a { color: var(--text-primary); }
  @media (max-width: 768px) { .settings-layout { flex-direction: column; } .settings-nav { flex-direction: row; overflow-x: auto; min-width: auto; } }
</style>
