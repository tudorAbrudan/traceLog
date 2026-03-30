<script lang="ts">
  import { api } from '../../api';

  export let servers: any[] = [];
  export let channels: any[] = [];

  // Alerts
  let alertRules: any[] = [];
  let alertHistory: any[] = [];
  let emailHistory: any[] = [];
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

  /** Client-side filter: show rules for this server only ('' = all). */
  let alertServerFilter = '';
  /** ID of the rule currently being edited ('' = none). */
  let editingRuleId = '';
  let editAlertMetric = 'cpu_percent';
  let editAlertOp = '>';
  let editAlertThreshold = 90;
  let editAlertDuration = 300;
  let editAlertChannel = '';
  let editAlertLogCooldown = 1800;
  let editAlertServerId = '';
  let editDockerContainer = '';

  $: filteredAlertRules = alertServerFilter
    ? alertRules.filter((r) => r.server_id === alertServerFilter)
    : alertRules;

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

  function isLogAlertMetric(m: string): boolean {
    return m === 'log_critical' || m === 'log_error' || m === 'log_warn';
  }

  function isDockerAlertMetric(m: string): boolean {
    return m === 'docker_mem_pct' || m === 'docker_cpu_percent';
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

  import { onMount } from 'svelte';

  onMount(async () => {
    alertRules = (await api.listAlertRules()) || [];
    logSilences = (await api.listLogAlertSilences()) || [];
    alertHistory = (await api.listAlertHistory(150)) || [];
    if (!newAlertServerId && servers.length > 0) {
      newAlertServerId = servers[0].id;
    }
  });

  function emailToForChannel(channelId: string): string {
    if (!channelId) return '';
    const ch = channels.find((c) => c.id === channelId);
    if (!ch || ch.type !== 'email') return '';
    try {
      const cfg = JSON.parse(ch.config || '{}');
      return typeof cfg.to === 'string' ? cfg.to : '';
    } catch {
      return '';
    }
  }

  function emailChannelName(channelId: string): string {
    if (!channelId) return '';
    const ch = channels.find((c) => c.id === channelId);
    return ch?.name || channelId;
  }

  $: emailHistory = alertHistory.filter((row) => {
    const ch = channels.find((c) => c.id === row.channel_id);
    return ch?.type === 'email';
  });

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

  function startEditRule(rule: any) {
    editingRuleId = rule.id;
    editAlertMetric = rule.metric;
    editAlertOp = rule.operator || '>';
    editAlertThreshold = rule.threshold ?? 90;
    editAlertDuration = rule.duration_seconds ?? 300;
    editAlertChannel = rule.channel_id || '';
    editAlertLogCooldown = rule.cooldown_seconds ?? 1800;
    editAlertServerId = rule.server_id || '';
    editDockerContainer = rule.docker_container || '';
  }

  async function saveEditRule() {
    if (!editingRuleId) return;
    const payload: Record<string, unknown> = {
      metric: editAlertMetric,
      operator: editAlertOp,
      threshold: editAlertThreshold,
      duration_seconds: isLogAlertMetric(editAlertMetric) ? 0 : editAlertDuration,
      cooldown_seconds: isLogAlertMetric(editAlertMetric) ? editAlertLogCooldown : editAlertDuration,
      channel_id: editAlertChannel,
      server_id: editAlertServerId,
      docker_container: editDockerContainer,
      enabled: true,
    };
    await api.updateAlertRule(editingRuleId, payload);
    editingRuleId = '';
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
</script>

<div class="section">
  <h3>Alert Rules</h3>
  <p class="hint">
    <strong>Metrics:</strong> if a value stays beyond the threshold for the <strong>duration</strong>, a notification is sent (then <strong>cooldown</strong> applies).
    <strong>Docker:</strong> uses each agent <code>docker stats</code> scrape; <code>docker_mem_pct</code> is memory used vs <em>container</em> cgroup limit (not host RAM). <code>docker_cpu_percent</code> is CPU share of the <em>host</em>. Optional container substring limits which containers are checked.
    <strong>Ingested logs:</strong> each line stored in TraceLog (files, Apache/nginx as plain, app logs, etc.) is classified; when level matches the rule, notify immediately (cooldown only — no duration).
    Container stderr/stdout only triggers log alerts if lines are <strong>ingested</strong> (e.g. json-file log + Log Source); UI <em>Load logs</em> on the server page does not store lines.
  </p>
  {#if servers.length > 1}
    <div class="alert-filter-row">
      <span class="hint-inline">Show:</span>
      <select bind:value={alertServerFilter} class="alert-server-filter">
        <option value="">All servers</option>
        {#each servers as s}
          <option value={s.id}>{s.name}</option>
        {/each}
      </select>
    </div>
  {/if}
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
      {#each filteredAlertRules as rule (rule.id)}
        <div class="item-row item-row-rule">
          <div class="rule-head">
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
                    ? `contains "${rule.docker_container}"`
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
            <div class="rule-actions">
              <button class="btn-edit" on:click={() => startEditRule(rule)}>Edit</button>
              <button class="btn-delete" on:click={() => removeAlert(rule.id)}>Delete</button>
            </div>
          </div>
          {#if editingRuleId === rule.id}
            <div class="edit-rule-form">
              <select bind:value={editAlertMetric} class="alert-metric-select">
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
              {#if !isLogAlertMetric(editAlertMetric)}
                <select bind:value={editAlertOp}>
                  <option value=">">{'>'}</option>
                  <option value=">=">{'>='}</option>
                  <option value="<">{'<'}</option>
                </select>
                <input type="number" bind:value={editAlertThreshold} min="0" max="100" style="width:80px" />
                <span class="hint-inline">for</span>
                <select bind:value={editAlertDuration}>
                  <option value={60}>1 min</option>
                  <option value={300}>5 min</option>
                  <option value={600}>10 min</option>
                </select>
              {:else}
                <span class="hint-inline">cooldown</span>
                <select bind:value={editAlertLogCooldown}>
                  <option value={300}>5 min</option>
                  <option value={900}>15 min</option>
                  <option value={1800}>30 min</option>
                  <option value={3600}>1 h</option>
                </select>
              {/if}
              <span class="hint-inline">notify</span>
              <select bind:value={editAlertChannel}>
                <option value="">None</option>
                {#each channels as ch}<option value={ch.id}>{ch.name}</option>{/each}
              </select>
              <button class="btn-save" on:click={saveEditRule}>Save</button>
              <button class="btn-cancel" on:click={() => (editingRuleId = '')}>Cancel</button>
            </div>
          {/if}
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

<div class="section email-history-section">
  <h3>Emails sent</h3>
  <p class="hint">
    Rows appear when the hub <strong>sends an email</strong> for a rule (newest first). This is not a full audit trail.
  </p>
  {#if emailHistory.length === 0}
    <p class="hint">None yet — trigger an alert or use Test on an email notification channel.</p>
  {:else}
    <div class="item-list">
      {#each emailHistory as row (row.id)}
        <div class="item-row alert-history-row">
          <div>
            <span class="item-detail">{row.ts}</span>
            <span class="item-detail">
              Rule <code>{row.rule_id}</code> · {silenceServerLabel(row.server_id)}
              {#if row.channel_id}
                · Email channel: {emailChannelName(row.channel_id)}
              {/if}
            </span>
            {#if emailToForChannel(row.channel_id)}
              <span class="item-detail alert-history-msg">To: {emailToForChannel(row.channel_id)}</span>
            {/if}
            <span class="item-detail alert-history-msg">{row.message}</span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .silence-section { margin-top: 1.75rem; padding-top: 1.25rem; border-top: 1px solid var(--border); }
  .email-history-section { margin-top: 1.75rem; padding-top: 1.25rem; border-top: 1px solid var(--border); }
  .silence-form .silence-pattern-input { min-width: 200px; flex: 1; max-width: 420px; }
  .silence-form .silence-select { min-width: 160px; }
  .silence-pattern { word-break: break-word; }
  .alert-metric-select { min-width: 200px; max-width: 100%; flex: 1 1 220px; }
  .alert-form-log { align-items: flex-end; }
  .alert-form-docker { align-items: flex-end; }
  .alert-server-select { min-width: 160px; flex: 1 1 140px; }
  .docker-filter-inp { min-width: 180px; flex: 2 1 200px; }
  .alert-history-row { align-items: flex-start; }
  .alert-history-msg { word-break: break-word; white-space: pre-wrap; color: var(--text-secondary); margin-top: 0.25rem; }
  .alert-filter-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  .alert-server-filter {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.3rem 0.5rem;
    font-size: 0.8rem;
  }
  .btn-edit {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-secondary);
    padding: 0.25rem 0.6rem;
    font-size: 0.75rem;
    cursor: pointer;
  }
  .btn-edit:hover { border-color: var(--accent); color: var(--accent); }
  .btn-cancel {
    background: transparent;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-muted);
    padding: 0.3rem 0.75rem;
    font-size: 0.78rem;
    cursor: pointer;
  }
  .item-row-rule { flex-direction: column; align-items: stretch; }
  .rule-head { display: flex; justify-content: space-between; align-items: center; width: 100%; }
  .rule-actions { display: flex; gap: 0.4rem; }
  .edit-rule-form {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    align-items: center;
    margin-top: 0.5rem;
    padding: 0.5rem;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
  }
</style>
