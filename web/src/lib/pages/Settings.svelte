<script lang="ts">
  import { onMount } from 'svelte';
  import { user } from '../store';
  import { api } from '../api';

  let activeTab = 'general';
  let retentionDays = 30;
  let collectionInterval = 10;
  let saved = false;
  /** Substrings matched case-insensitively in User-Agent; excluded from HTTP Analytics aggregates (not from raw “Recent requests”). */
  let excludeUAText = '';

  // Log sources
  let logSources: any[] = [];
  let newLogName = ''; let newLogPath = ''; let newLogFormat = 'plain';
  /** Empty = local hub agent; set to a server id for remote `tracelog agent` tail (path must exist on that host). */
  let newLogServerId = '';

  // Notifications
  let channels: any[] = [];
  let newChName = ''; let newChType = 'email'; let newChConfig = '';
  let editingChannelId = '';
  let editChName = '';
  let editChType = 'email';
  let editChConfig = '';

  let aboutVersion = '';

  let exportPassword = '';
  let exportBusy = false;
  let exportErr = '';

  // Servers
  let servers: any[] = [];

  // Alerts
  let alertRules: any[] = [];
  let alertHistory: any[] = [];
  /** Substring silences for ingested log alert notifications (Settings → Alerts). */
  let logSilences: any[] = [];
  let newSilencePattern = '';
  let newSilenceServerId = '';
  let newSilenceRuleMetric = '';
  let newAlertMetric = 'cpu_percent'; let newAlertOp = '>'; let newAlertThreshold = 90;
  let newAlertDuration = 300; let newAlertChannel = '';
  /** Cooldown for log-based alerts (seconds). */
  let newAlertLogCooldown = 1800;
  /** Target server for Docker metric rules (agent that scrapes docker stats). */
  let newAlertServerId = '';
  /** Substring filter on container name; empty = all containers. */
  let newDockerContainer = '';

  const metricAlerts = ['cpu_percent', 'mem_percent', 'disk_percent', 'load_1', 'load_5', 'load_15'];
  const dockerAlertMetrics = [
    { id: 'docker_mem_pct', label: 'Docker · memory % of container limit' },
    { id: 'docker_cpu_percent', label: 'Docker · CPU % (host share, docker stats)' },
  ];
  const logAlertMetrics = [
    { id: 'log_critical', label: 'Ingested log · critical only' },
    { id: 'log_error', label: 'Ingested log · error or critical' },
    { id: 'log_warn', label: 'Ingested log · warn, error, or critical' },
  ];

  const ingestLevelOpts = ['critical', 'error', 'warn', 'info', 'debug', 'deprecated'] as const;
  /** Per log source id: which severities to store (empty = all). */
  let ingestPick: Record<string, Record<string, boolean>> = {};

  const tabs = [
    { id: 'general', label: 'General' },
    { id: 'logs', label: 'Log Sources' },
    { id: 'notifications', label: 'Notifications' },
    { id: 'servers', label: 'Servers' },
    { id: 'alerts', label: 'Alerts' },
    { id: 'account', label: 'Account' },
    { id: 'about', label: 'About' },
  ];

  function isLogAlertMetric(m: string): boolean {
    return m === 'log_critical' || m === 'log_error' || m === 'log_warn';
  }

  function isDockerAlertMetric(m: string): boolean {
    return m === 'docker_mem_pct' || m === 'docker_cpu_percent';
  }


  /** JSON template for Gmail SMTP (hub EmailConfig: use_tls = implicit TLS/465; starttls = STARTTLS on 587). */
  const gmailConfigTemplate = `{
  "host": "smtp.gmail.com",
  "port": 587,
  "username": "myemail@gmail.com",
  "password": "xxxxxxxxxxxxxxxx",
  "from": "myemail@gmail.com",
  "to": "supervisor@example.com",
  "use_tls": false,
  "starttls": true
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
      try {
        const raw = s.access_stats_exclude_ua_substrings;
        if (raw) {
          const arr = JSON.parse(raw);
          excludeUAText = Array.isArray(arr) ? arr.join('\n') : '';
        }
      } catch {
        excludeUAText = '';
      }
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
      if (tab === 'logs') {
        logSources = (await api.listLogSources()) || [];
        syncIngestPickFromSources();
        if (servers.length === 0) {
          servers = (await api.listServers()) || [];
        }
      }
      if (tab === 'notifications') channels = (await api.listNotificationChannels()) || [];
      if (tab === 'servers') servers = (await api.listServers()) || [];
      if (tab === 'alerts') {
        alertRules = (await api.listAlertRules()) || [];
        channels = (await api.listNotificationChannels()) || [];
        logSilences = (await api.listLogAlertSilences()) || [];
        servers = (await api.listServers()) || [];
        alertHistory = (await api.listAlertHistory(150)) || [];
        if (!newAlertServerId && servers.length > 0) {
          newAlertServerId = servers[0].id;
        }
      }
    } catch {}
  }

  function syncIngestPickFromSources() {
    const next: Record<string, Record<string, boolean>> = {};
    for (const ls of logSources) {
      const cur = ls.ingest_levels;
      const row: Record<string, boolean> = {};
      for (const lv of ingestLevelOpts) {
        row[lv] = Array.isArray(cur) && cur.includes(lv);
      }
      next[ls.id] = row;
    }
    ingestPick = next;
  }

  async function saveGeneral() {
    try {
      const uaLines = excludeUAText
        .split('\n')
        .map((x) => x.trim())
        .filter(Boolean);
      await api.updateSettings({
        retention_days: String(retentionDays),
        collection_interval: String(collectionInterval),
        access_stats_exclude_ua_substrings: JSON.stringify(uaLines),
      });
      saved = true; setTimeout(() => saved = false, 2000);
    } catch (e: any) { alert('Save failed: ' + e.message); }
  }

  async function saveIngestLevels(lsId: string) {
    const row = ingestPick[lsId] || {};
    const levels = ingestLevelOpts.filter((lv) => row[lv]);
    try {
      await api.updateLogSource(lsId, { ingest_levels: levels });
      logSources = (await api.listLogSources()) || [];
      syncIngestPickFromSources();
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
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
      await api.createLogSource({
        name,
        path,
        format: newLogFormat,
        type: 'file',
        server_id: newLogServerId.trim(),
        enabled: true,
      });
      newLogName = ''; newLogPath = '';
      logSources = (await api.listLogSources()) || [];
      syncIngestPickFromSources();
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
        syncIngestPickFromSources();
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
  function startEditChannel(ch: { id: string; name?: string; type?: string; config?: string }) {
    editingChannelId = ch.id;
    editChName = ch.name || '';
    editChType = ch.type === 'webhook' ? 'webhook' : 'email';
    editChConfig = ch.config || '';
  }

  function cancelEditChannel() {
    editingChannelId = '';
  }

  async function saveEditedChannel() {
    if (!editingChannelId) return;
    if (!editChName?.trim() || !editChConfig?.trim()) {
      alert('Enter a channel name and configuration JSON.');
      return;
    }
    try {
      await api.updateNotificationChannel(editingChannelId, {
        name: editChName.trim(),
        type: editChType,
        config: editChConfig,
      });
      cancelEditChannel();
      channels = (await api.listNotificationChannels()) || [];
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function removeChannel(id: string) {
    if (editingChannelId === id) cancelEditChannel();
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
      const log = isLogAlertMetric(newAlertMetric);
      const dock = isDockerAlertMetric(newAlertMetric);
      if (dock && !newAlertServerId?.trim()) {
        alert('Choose a server for the Docker alert (the agent that runs docker stats).');
        return;
      }
      await api.createAlertRule({
        metric: newAlertMetric,
        operator: log ? '>' : newAlertOp,
        threshold: log ? 0 : newAlertThreshold,
        duration_seconds: log ? 0 : newAlertDuration,
        cooldown_seconds: log ? newAlertLogCooldown : 1800,
        channel_id: newAlertChannel,
        server_id: dock ? newAlertServerId.trim() : '',
        docker_container: dock ? newDockerContainer.trim() : '',
        enabled: true,
      });
      alertRules = (await api.listAlertRules()) || [];
    } catch (e: any) { alert('Failed: ' + e.message); }
  }
  async function removeAlert(id: string) {
    await api.deleteAlertRule(id);
    alertRules = (await api.listAlertRules()) || [];
  }

  async function addLogSilence() {
    const pattern = newSilencePattern.trim();
    if (!pattern) {
      alert('Enter a text pattern (case-insensitive substring of the log message).');
      return;
    }
    try {
      await api.createLogAlertSilence({
        pattern,
        server_id: newSilenceServerId.trim() || undefined,
        rule_metric: newSilenceRuleMetric.trim() || undefined,
      });
      newSilencePattern = '';
      newSilenceServerId = '';
      newSilenceRuleMetric = '';
      logSilences = (await api.listLogAlertSilences()) || [];
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function removeLogSilence(id: string) {
    await api.deleteLogAlertSilence(id);
    logSilences = (await api.listLogAlertSilences()) || [];
  }

  function silenceServerLabel(sid: string): string {
    if (!sid) return 'All servers';
    const s = servers.find((x) => x.id === sid);
    return s ? `${s.name} (${sid.slice(0, 8)}…)` : sid;
  }

  function silenceRuleLabel(metric: string): string {
    if (!metric) return 'All log alert rules';
    return logAlertMetrics.find((x) => x.id === metric)?.label ?? metric;
  }

  function logSourceAgentLabel(sid: string): string {
    if (!sid?.trim()) return 'This hub (local agent)';
    const s = servers.find((x) => x.id === sid);
    return s ? `Remote: ${s.name}` : sid.slice(0, 12);
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
          <div class="field">
            <label for="exclude-ua">HTTP analytics — ignore User-Agent (one substring per line)</label>
            <p class="hint field-hint">
              Rows whose User-Agent contains any of these (case-insensitive) are <strong>excluded from</strong> Top URL paths, Top method + path, Top IPs, and summary counts.
              Default includes TraceLog’s uptime probe. Raw “Recent requests” on HTTP Analytics is unchanged. Add other lines if your UI or bots use a fixed User-Agent.
            </p>
            <textarea
              id="exclude-ua"
              class="ua-exclude-ta"
              bind:value={excludeUAText}
              rows="3"
              placeholder="TraceLog/1.0 Uptime Monitor"
            ></textarea>
          </div>
          <button class="btn-save" on:click={saveGeneral}>{saved ? '✓ Saved' : 'Save Changes'}</button>
        </div>

      {:else if activeTab === 'logs'}
        <div class="section">
          <h3>Log Sources</h3>
          <p class="hint">
            <strong>Local hub (serve mode):</strong> sources with agent <em>This hub</em> are loaded when TraceLog starts on this machine.
            <strong>Restart TraceLog</strong> after add/remove (e.g. <code>sudo systemctl restart tracelog</code>).
            <strong>Remote agent:</strong> choose a monitored server — the file path must exist <em>on that host</em>; <code>tracelog agent</code> polls the hub about every <strong>2 minutes</strong> and starts tailing without a restart.
            The agent <strong>tails from the end of each file</strong> — only new lines after tail starts appear in Logs.
          </p>
          <p class="hint">Scan checks <strong>this</strong> machine for usual paths and adds local sources only.</p>
          <button class="btn-secondary" on:click={scanLogs}>Scan for common log files</button>
          <div class="add-form add-form-logs">
            <input type="text" bind:value={newLogName} placeholder="Name" />
            <input type="text" bind:value={newLogPath} placeholder="/var/log/..." />
            <select bind:value={newLogFormat} class="log-format-select">
              <option value="plain">Plain</option>
              <option value="nginx">Nginx</option>
              <option value="apache">Apache</option>
            </select>
            <select bind:value={newLogServerId} class="log-agent-select" title="Which agent tails this path">
              <option value="">This hub (local agent)</option>
              {#each servers as srv}
                <option value={srv.id}>Remote: {srv.name}</option>
              {/each}
            </select>
            <button class="btn-save" on:click={addLogSource}>Add</button>
          </div>
          <p class="hint">For <strong>This hub</strong>, the file must exist here and nginx/apache formats are validated from file samples. For <strong>Remote</strong>, only name/path/format are checked on the hub — ensure the path is correct on the agent host.</p>
          <p class="hint">
            <strong>Store only chosen severities</strong> (per source): unchecked = ingest nothing for that level. If <em>none</em> are checked, <strong>all</strong> levels are stored. After changing filters, <strong>restart TraceLog</strong> so the agent reloads config. The hub also drops non-matching lines as a safeguard.
          </p>
          {#if logSources.length === 0}
            <p class="hint">No log sources configured. Click "Scan" or add manually above.</p>
          {:else}
            <div class="item-list">
              {#each logSources as ls (ls.id)}
                <div class="item-row ingest-row">
                  <div class="ingest-main">
                    <strong>{ls.name}</strong>
                    <span class="item-detail">{logSourceAgentLabel(ls.server_id)} · {ls.path || ls.container} ({ls.format})</span>
                    <div class="ingest-levels">
                      {#each ingestLevelOpts as lv}
                        <label class="ingest-cb"
                          ><input type="checkbox" bind:checked={ingestPick[ls.id][lv]} /> {lv}</label
                        >
                      {/each}
                    </div>
                    <button type="button" class="btn-secondary ingest-save" on:click={() => saveIngestLevels(ls.id)}
                      >Save levels</button
                    >
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
              <li>Port <code>587</code>: set <code>starttls</code> to <code>true</code> and <code>use_tls</code> to <code>false</code> (plain connect, then STARTTLS). Port <code>465</code> uses implicit TLS — set <code>use_tls</code> to <code>true</code> instead.</li>
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
              ? 'JSON: host, port, username, password, from, to, use_tls, starttls — see Gmail example above'
              : '{"url":"https://hooks.slack.com/...","method":"POST"}'}
            ></textarea>
            <button class="btn-save" on:click={addChannel}>Add Channel</button>
          </div>
          {#if channels.length === 0}
            <p class="hint">No notification channels configured.</p>
          {:else}
            <div class="item-list">
              {#each channels as ch (ch.id)}
                <div class="item-row" class:channel-edit-row={editingChannelId === ch.id}>
                  {#if editingChannelId === ch.id}
                    <div class="channel-edit-fields">
                      <input type="text" bind:value={editChName} placeholder="Channel name" />
                      <select bind:value={editChType}>
                        <option value="email">Email (SMTP)</option>
                        <option value="webhook">Webhook</option>
                      </select>
                      <textarea
                        bind:value={editChConfig}
                        placeholder={editChType === 'email'
                          ? 'JSON: host, port, username, password, from, to, use_tls, starttls'
                          : '{"url":"https://…","method":"POST"}'}
                        rows="6"
                      ></textarea>
                      <div class="item-actions channel-edit-actions">
                        <button type="button" class="btn-save" on:click={saveEditedChannel}>Save</button>
                        <button type="button" class="btn-secondary" on:click={cancelEditChannel}>Cancel</button>
                      </div>
                    </div>
                  {:else}
                    <div>
                      <strong>{ch.name}</strong>
                      <span class="item-detail">{ch.type}</span>
                    </div>
                    <div class="item-actions">
                      <button type="button" class="btn-secondary" on:click={() => startEditChannel(ch)}>Edit</button>
                      <button type="button" class="btn-secondary" on:click={() => testChannel(ch.id)}>Test</button>
                      <button type="button" class="btn-delete" on:click={() => removeChannel(ch.id)}>Delete</button>
                    </div>
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>

      {:else if activeTab === 'servers'}
        <div class="section">
          <h3>Connected Servers</h3>
          <p class="hint">Each server is an agent (or the local node in <code>serve</code> mode). The <strong>API key</strong> is used by <code>tracelog agent --hub … --key …</code>. Deleting a server removes its metrics and stored logs for that server ID from TraceLog’s database.</p>
          <p class="hint">Most tabs in Settings are <strong>hub-wide</strong>. <strong>Log Sources</strong> can target the local <code>serve</code> agent or a <strong>remote</strong> server row (see Log Sources tab); remote agents pull their file list from the hub periodically.</p>
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
          <p class="hint">
            <strong>Metrics:</strong> if a value stays beyond the threshold for the <strong>duration</strong>, a notification is sent (then <strong>cooldown</strong> applies).
            <strong>Docker:</strong> uses each agent <code>docker stats</code> scrape; <code>docker_mem_pct</code> is memory used vs <em>container</em> cgroup limit (not host RAM). <code>docker_cpu_percent</code> is CPU share of the <em>host</em>. Optional container substring limits which containers are checked.
            <strong>Ingested logs:</strong> each line stored in TraceLog (files, Apache/nginx as plain, app logs, etc.) is classified; when level matches the rule, notify immediately (cooldown only — no duration).
            Container stderr/stdout only triggers log alerts if lines are <strong>ingested</strong> (e.g. json-file log + Log Source); UI <em>Load logs</em> on the server page does not store lines.
          </p>
          <div
            class="add-form alert-form"
            class:alert-form-log={isLogAlertMetric(newAlertMetric)}
            class:alert-form-docker={isDockerAlertMetric(newAlertMetric)}
          >
            <select bind:value={newAlertMetric} class="alert-metric-select">
              <optgroup label="System metrics">
                {#each metricAlerts as m}<option value={m}>{m}</option>{/each}
              </optgroup>
              <optgroup label="Docker containers">
                {#each dockerAlertMetrics as dm}<option value={dm.id}>{dm.label}</option>{/each}
              </optgroup>
              <optgroup label="Ingested log level">
                {#each logAlertMetrics as lm}<option value={lm.id}>{lm.label}</option>{/each}
              </optgroup>
            </select>
            {#if isDockerAlertMetric(newAlertMetric)}
              <select bind:value={newAlertServerId} class="alert-server-select" title="Agent host that runs Docker">
                {#each servers as s}<option value={s.id}>{s.name} ({s.host || s.id.slice(0, 8)}…)</option>{/each}
              </select>
              <input
                type="text"
                class="docker-filter-inp"
                bind:value={newDockerContainer}
                placeholder="Container name contains (empty = all)"
                title="Case-insensitive substring; leave empty to evaluate every container"
              />
            {/if}
            {#if !isLogAlertMetric(newAlertMetric)}
              <select bind:value={newAlertOp}>
                <option value=">">{'>'}</option>
                <option value=">=">{'>='}</option>
                <option value="<">{'<'}</option>
              </select>
              <input
                type="number"
                bind:value={newAlertThreshold}
                min="0"
                max={isDockerAlertMetric(newAlertMetric) && newAlertMetric === 'docker_cpu_percent' ? 5000 : 100}
                style="width:80px"
                title={isDockerAlertMetric(newAlertMetric) ? 'Percent for docker_mem_pct; docker_cpu can exceed 100% on multi-core' : ''}
              />
              <span class="hint-inline">for</span>
              <select bind:value={newAlertDuration}>
                <option value={60}>1 min</option>
                <option value={300}>5 min</option>
                <option value={600}>10 min</option>
              </select>
            {:else}
              <span class="hint-inline">cooldown</span>
              <select bind:value={newAlertLogCooldown}>
                <option value={300}>5 min</option>
                <option value={900}>15 min</option>
                <option value={1800}>30 min</option>
                <option value={3600}>1 h</option>
              </select>
            {/if}
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
                    <strong
                      >{isLogAlertMetric(rule.metric)
                        ? (logAlertMetrics.find((x) => x.id === rule.metric)?.label ?? rule.metric)
                        : isDockerAlertMetric(rule.metric)
                          ? `${dockerAlertMetrics.find((x) => x.id === rule.metric)?.label ?? rule.metric} ${rule.operator} ${rule.threshold}`
                          : `${rule.metric} ${rule.operator} ${rule.threshold}`}</strong
                    >
                    {#if isDockerAlertMetric(rule.metric)}
                      <span class="item-detail"
                        >Server: {silenceServerLabel(rule.server_id)} · containers: {rule.docker_container?.trim()
                          ? `contains “${rule.docker_container}”`
                          : 'all'}</span
                      >
                    {/if}
                    <span class="item-detail">
                      {#if isLogAlertMetric(rule.metric)}
                        Cooldown: {Math.round((rule.cooldown_seconds ?? 0) / 60)} min between notifications
                      {:else}
                        Must hold {rule.duration_seconds}s · Cooldown {Math.round((rule.cooldown_seconds ?? 0) / 60)} min
                      {/if}
                    </span>
                  </div>
                  <button class="btn-delete" on:click={() => removeAlert(rule.id)}>Delete</button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <div class="section">
          <h3>Recent alert notifications</h3>
          <p class="hint">
            Rows appear when the hub <strong>sends</strong> a notification (email/webhook) for a rule. Newest first. This is not a full audit trail.
          </p>
          {#if alertHistory.length === 0}
            <p class="hint">None yet — trigger an alert or use Test on a notification channel.</p>
          {:else}
            <div class="item-list">
              {#each alertHistory as row (row.id)}
                <div class="item-row alert-history-row">
                  <div>
                    <span class="item-detail">{row.ts}</span>
                    <span class="item-detail">Rule <code>{row.rule_id}</code> · {silenceServerLabel(row.server_id)}</span>
                    <span class="item-detail alert-history-msg">{row.message}</span>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <div class="section silence-section">
          <h3>Log alert silences</h3>
          <p class="hint">
            When an ingested line would trigger a <strong>log-based</strong> alert, matching silences skip the notification (line is still stored). Match is case-insensitive substring on the message. Leave server or rule empty to apply to all.
          </p>
          <div class="add-form silence-form">
            <input type="text" bind:value={newSilencePattern} placeholder="Substring e.g. document not found" class="silence-pattern-input" />
            <select bind:value={newSilenceServerId} class="silence-select">
              <option value="">All servers</option>
              {#each servers as srv (srv.id)}
                <option value={srv.id}>{srv.name}</option>
              {/each}
            </select>
            <select bind:value={newSilenceRuleMetric} class="silence-select">
              <option value="">All log rules</option>
              {#each logAlertMetrics as lm}<option value={lm.id}>{lm.label}</option>{/each}
            </select>
            <button type="button" class="btn-save" on:click={addLogSilence}>Add silence</button>
          </div>
          {#if logSilences.length === 0}
            <p class="hint">No silences. Noisy recurring lines can be muted here without turning off alerts entirely.</p>
          {:else}
            <div class="item-list">
              {#each logSilences as s (s.id)}
                <div class="item-row">
                  <div>
                    <strong class="silence-pattern">{s.pattern}</strong>
                    <span class="item-detail">{silenceServerLabel(s.server_id)} · {silenceRuleLabel(s.rule_metric)}</span>
                  </div>
                  <button type="button" class="btn-delete" on:click={() => removeLogSilence(s.id)}>Delete</button>
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
            Download a SQLite snapshot of <strong>TraceLog’s own hub database</strong> (metrics, log copies, users — same idea as <code>VACUUM INTO</code> / CLI <code>tracelog backup</code>). Enter your <strong>TraceLog login</strong> password to confirm. This is not a dump of an external app database (MySQL, Postgres, etc.).
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
  .silence-section { margin-top: 1.75rem; padding-top: 1.25rem; border-top: 1px solid var(--border); }
  .silence-form .silence-pattern-input { min-width: 200px; flex: 1; max-width: 420px; }
  .silence-form .silence-select { min-width: 160px; }
  .silence-pattern { word-break: break-word; }
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
  .item-actions { display: flex; gap: 0.5rem; flex-wrap: wrap; }
  .channel-edit-row { align-items: stretch; }
  .channel-edit-fields {
    display: flex; flex-direction: column; gap: 0.5rem; width: 100%; min-width: 0;
  }
  .channel-edit-fields input,
  .channel-edit-fields select,
  .channel-edit-fields textarea {
    padding: 0.5rem; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 6px;
    color: var(--text-primary); font-size: 0.85rem; box-sizing: border-box; width: 100%;
  }
  .channel-edit-fields textarea { font-family: monospace; font-size: 0.8rem; resize: vertical; min-height: 120px; }
  .channel-edit-actions { justify-content: flex-end; }
  .btn-delete { padding: 0.3rem 0.7rem; background: none; border: 1px solid var(--border); color: var(--text-muted); border-radius: 6px; cursor: pointer; font-size: 0.75rem; }
  .btn-delete:hover { border-color: #f85149; color: #f85149; }
  .about-grid { display: flex; flex-direction: column; gap: 0.75rem; }
  .about-item { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
  .about-item span { color: var(--text-muted); }
  .about-item strong, .about-item a { color: var(--text-primary); }
  .alert-metric-select { min-width: 200px; max-width: 100%; flex: 1 1 220px; }
  .alert-form-log { align-items: flex-end; }
  .alert-form-docker { align-items: flex-end; }
  .alert-server-select { min-width: 160px; flex: 1 1 140px; }
  .docker-filter-inp { min-width: 180px; flex: 2 1 200px; }
  .ua-exclude-ta {
    width: 100%; min-height: 4rem; font-family: monospace; font-size: 0.78rem;
    padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 8px;
    color: var(--text-primary); resize: vertical;
  }
  .ingest-row { align-items: flex-start; }
  .ingest-main { flex: 1; min-width: 0; }
  .ingest-levels { display: flex; flex-wrap: wrap; gap: 0.35rem 0.9rem; margin: 0.45rem 0; }
  .ingest-cb { font-size: 0.72rem; color: var(--text-secondary); cursor: pointer; user-select: none; }
  .ingest-save { margin-bottom: 0 !important; margin-top: 0.35rem; }
  .add-form-logs { align-items: flex-end; }
  .log-agent-select { min-width: 180px; flex: 1 1 160px; }
  .alert-history-row { align-items: flex-start; }
  .alert-history-msg { word-break: break-word; white-space: pre-wrap; color: var(--text-secondary); margin-top: 0.25rem; }
  @media (max-width: 900px) {
    .settings { padding: 1rem 0.5rem; }
    .settings-layout { flex-direction: column; gap: 0.75rem; }
    .settings-nav { flex-direction: row; flex-wrap: wrap; overflow-x: auto; min-width: auto; }
    .settings-nav button { font-size: 0.78rem; padding: 0.45rem 0.55rem; }
    .settings-content { padding: 1rem; }
    .item-row { flex-direction: column; align-items: flex-start; gap: 0.5rem; }
  }
</style>
