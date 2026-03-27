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

  let aboutVersion = '';

  let exportPassword = '';
  let exportBusy = false;
  let exportErr = '';

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

  /** JSON template for Gmail SMTP (matches hub EmailConfig: host, port, username, password, from, to, use_tls). */
  const gmailConfigTemplate = `{
  "host": "smtp.gmail.com",
  "port": 587,
  "username": "you@gmail.com",
  "password": "xxxx xxxx xxxx xxxx",
  "from": "you@gmail.com",
  "to": "alerts@example.com",
  "use_tls": true
}`;

  function insertGmailTemplate() {
    newChType = 'email';
    newChConfig = gmailConfigTemplate.trim();
  }

  onMount(async () => {
    try {
      const s = await api.getSettings();
      retentionDays = parseInt(s.retention_days) || 30;
      collectionInterval = parseInt(s.collection_interval) || 10;
    } catch {}
  });

  async function loadTab(tab: string) {
    activeTab = tab;
    if (tab === 'about') {
      try {
        const h = await api.health();
        aboutVersion = h?.version || 'unknown';
      } catch {
        aboutVersion = 'unknown';
      }
    }
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
    const name = newLogName.trim();
    const path = newLogPath.trim();
    if (!name || !path) {
      alert('Enter a display name and an absolute file path (e.g. /var/log/nginx/access.log).');
      return;
    }
    try {
      await api.createLogSource({ name, path, format: newLogFormat, type: 'file', server_id: '', enabled: true });
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
    if (!newChName?.trim() || !newChConfig?.trim()) {
      alert('Enter a channel name and configuration JSON.');
      return;
    }
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
          <p class="hint">Applies to metrics, Docker stats, <strong>ingested log lines</strong>, HTTP access rows, uptime results, alert history, and process metrics. Older data is removed automatically about every hour — not the same as the <strong>Logs</strong> page “Purge”, which clears stored lines on demand.</p>
          <div class="field">
            <label for="retention">Keep data for</label>
            <div class="range-input">
              <input id="retention" type="range" min="1" max="30" bind:value={retentionDays} />
              <span>{retentionDays} days</span>
            </div>
          </div>
          <div class="field">
            <label for="interval">Collection interval</label>
            <p class="hint field-hint">How often agents send system (and related) metric samples to the hub. Lower values mean fresher charts and slightly more traffic.</p>
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
          <p class="hint">Scan checks this server for usual paths (nginx, apache, syslog, etc.) and adds only files that exist. Format is set per file type (e.g. nginx for access logs).</p>
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
          <p class="hint">Manual add: the file must exist on the machine running TraceLog. Nginx and apache formats are checked against the first lines of the file (access-log style). Use plain for error logs, app output, or syslog-style lines.</p>
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

          <div class="notify-help">
            <h4 class="notify-help-title">Gmail (SMTP) — example &amp; setup</h4>
            <p class="hint notify-help-lead">
              With <strong>2-Step Verification</strong> enabled, Google does not allow normal account passwords for SMTP. Use an
              <strong>App Password</strong> in the <code>password</code> field.
            </p>
            <ol class="notify-steps">
              <li>Open <a href="https://myaccount.google.com/security" target="_blank" rel="noopener noreferrer">Google Account security</a> and ensure <strong>2-Step Verification</strong> is on.</li>
              <li>Go to <a href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer">App passwords</a> (or search “App passwords” in account settings if the link is hidden).</li>
              <li>Create a new app password (e.g. app: Mail, device: Other → “TraceLog”), copy the 16 characters into JSON — with or without spaces as Google shows them.</li>
              <li><code>username</code> and <code>from</code> must be your full Gmail address. <code>to</code> is any address that should receive alerts.</li>
              <li>Keep <code>host</code> <code>smtp.gmail.com</code>, <code>port</code> <code>587</code>, <code>use_tls</code> <code>true</code> (STARTTLS).</li>
            </ol>
            <p class="hint">After saving the channel, use <strong>Test</strong> to verify delivery.</p>
            <div class="notify-example-row">
              <pre class="notify-pre"><code>{gmailConfigTemplate.trim()}</code></pre>
              <button type="button" class="btn-secondary notify-insert-btn" on:click={insertGmailTemplate}>Insert example into form</button>
            </div>
          </div>

          <p class="hint">Webhook: JSON body includes <code>subject</code>, <code>body</code>, and <code>time</code> (RFC3339). Optional <code>headers</code> map in config for auth tokens.</p>
          <div class="add-form">
            <input type="text" bind:value={newChName} placeholder="Channel name" />
            <select bind:value={newChType}>
              <option value="email">Email (SMTP)</option>
              <option value="webhook">Webhook</option>
            </select>
            <textarea bind:value={newChConfig} placeholder={newChType === 'email'
              ? 'JSON: host, port, username, password, from, to, use_tls — see Gmail example above'
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
          <p class="hint">Each server is an agent (or the local node in <code>serve</code> mode). The <strong>API key</strong> is used by <code>tracelog agent --hub … --key …</code>. Deleting a server removes its metrics and stored logs for that server ID from TraceLog’s database.</p>
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
          <p class="hint">Evaluated on incoming system metrics. If the metric stays beyond the threshold for the <strong>duration</strong>, a notification is sent to the chosen channel (after cooldown). Pick a channel from <strong>Notifications</strong> first.</p>
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

          <h4 class="subhead">Database backup</h4>
          <p class="hint">
            Download a SQLite snapshot of this hub’s data (same idea as <code>VACUUM INTO</code> / CLI backup). Enter your TraceLog password to confirm.
          </p>
          {#if exportErr}
            <p class="export-err">{exportErr}</p>
          {/if}
          <div class="export-row">
            <input
              type="password"
              class="export-pass"
              placeholder="Your TraceLog password"
              bind:value={exportPassword}
              autocomplete="current-password"
            />
            <button
              type="button"
              class="btn-secondary"
              disabled={exportBusy}
              on:click={async () => {
                exportErr = '';
                if (!exportPassword.trim()) {
                  exportErr = 'Enter your password to download a backup.';
                  return;
                }
                exportBusy = true;
                try {
                  await api.exportDatabase(exportPassword);
                  exportPassword = '';
                } catch (e: any) {
                  exportErr = e.message || 'Download failed';
                } finally {
                  exportBusy = false;
                }
              }}
            >
              {exportBusy ? 'Preparing…' : 'Download backup'}
            </button>
          </div>
        </div>

      {:else if activeTab === 'about'}
        <div class="section">
          <h3>About TraceLog</h3>
          <p class="hint">Released builds show the same version as <code>tracelog version</code> (set at compile time). <code>dev</code> means a local or non-release binary.</p>
          <div class="about-grid">
            <div class="about-item"><span>Version</span><strong>{aboutVersion || '…'}</strong></div>
            <div class="about-item"><span>License</span><strong>MIT</strong></div>
            <div class="about-item"><span>GitHub</span><a href="https://github.com/tudorAbrudan/tracelog" target="_blank" rel="noopener noreferrer">tudorAbrudan/tracelog</a></div>
            <div class="about-item"><span>Docs</span><a href="https://tudorAbrudan.github.io/traceLog/guide/logs-http-analytics" target="_blank" rel="noopener noreferrer">Logs &amp; HTTP analytics</a></div>
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
  .subhead { font-size: 0.9rem; margin: 1.25rem 0 0.5rem 0; color: var(--text-primary); font-weight: 600; }
  .export-row { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; margin-top: 0.5rem; }
  .export-pass {
    padding: 0.5rem 0.65rem; min-width: 220px; flex: 1; max-width: 320px;
    background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px;
    color: var(--text-primary); font-size: 0.85rem;
  }
  .export-err { color: #f85149; font-size: 0.85rem; margin: 0.5rem 0 0 0; }
  .field { margin-bottom: 1rem; }
  .field label { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.3rem; }
  .field-hint { margin-top: -0.2rem; margin-bottom: 0.4rem !important; }
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
  .notify-help { margin-bottom: 1.25rem; padding: 1rem 1.1rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 10px; }
  .notify-help-title { font-size: 0.95rem; margin: 0 0 0.5rem 0; color: var(--text-primary); }
  .notify-help-lead { margin-bottom: 0.75rem !important; }
  .notify-steps { margin: 0 0 0.75rem 0; padding-left: 1.25rem; font-size: 0.82rem; color: var(--text-secondary); line-height: 1.5; }
  .notify-steps li { margin-bottom: 0.4rem; }
  .notify-steps a { color: var(--accent); }
  .notify-example-row { display: flex; flex-wrap: wrap; gap: 0.75rem; align-items: flex-start; margin-top: 0.5rem; }
  .notify-pre { flex: 1; min-width: 220px; margin: 0; padding: 0.65rem 0.8rem; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 8px; overflow-x: auto; font-size: 0.75rem; line-height: 1.45; color: var(--text-primary); }
  .notify-pre code { background: none; padding: 0; font-size: inherit; color: inherit; }
  .notify-insert-btn { margin-bottom: 0 !important; align-self: center; white-space: nowrap; }
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
