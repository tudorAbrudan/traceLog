<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api';

  let retentionDays = 30;
  let collectionInterval = 10;
  let saved = false;
  /** Substrings matched case-insensitively in User-Agent; excluded from HTTP Analytics aggregates (not from raw "Recent requests"). */
  let excludeUAText = '';
  /** ipinfo.io API key for IP geolocation + threat scoring. */
  let ipinfoApiKey = '';

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
      ipinfoApiKey = s.ipinfo_io_api_key || '';
    } catch {}
  });

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
        ipinfo_io_api_key: ipinfoApiKey.trim(),
      });
      saved = true; setTimeout(() => saved = false, 2000);
    } catch (e: any) { alert('Save failed: ' + e.message); }
  }
</script>

<div class="section">
  <h3>Data Retention</h3>
  <p class="hint">Applies to metrics, Docker stats, <strong>ingested log lines</strong>, HTTP access rows, uptime results, alert history, and process metrics. Older data is removed automatically about every hour — not the same as the <strong>Logs</strong> page "Purge", which clears stored lines on demand.</p>
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
      Default includes TraceLog's uptime probe. Raw "Recent requests" on HTTP Analytics is unchanged. Add other lines if your UI or bots use a fixed User-Agent.
    </p>
    <textarea
      id="exclude-ua"
      class="ua-exclude-ta"
      bind:value={excludeUAText}
      rows="3"
      placeholder="TraceLog/1.0 Uptime Monitor"
    ></textarea>
  </div>
  <div class="field">
    <label for="ipinfo-key">ipinfo.io API key (optional)</label>
    <p class="hint field-hint">
      Bearer token for <strong>ipinfo.io/lite</strong> endpoint; enables automatic IP geolocation lookups on the HTTP Analytics "Recommended to block" panel.
      Get your free API key at <a href="https://ipinfo.io" target="_blank" rel="noopener noreferrer">ipinfo.io</a>.
      Stored securely in TraceLog's database; only the hub server sends requests to ipinfo.io.
    </p>
    <input
      id="ipinfo-key"
      type="password"
      bind:value={ipinfoApiKey}
      placeholder="Paste your ipinfo.io bearer token"
      class="ipinfo-input"
    />
  </div>
  <button class="btn-save" on:click={saveGeneral}>{saved ? '✓ Saved' : 'Save Changes'}</button>
</div>

<style>
  .field { margin-bottom: 1rem; }
  .field label { display: block; font-size: 0.85rem; color: var(--text-secondary); margin-bottom: 0.3rem; }
  .field-hint { margin-top: -0.2rem; margin-bottom: 0.4rem !important; }
  .range-input { display: flex; align-items: center; gap: 0.75rem; }
  .range-input input { flex: 1; }
  .range-input span { min-width: 60px; font-size: 0.85rem; color: var(--text-primary); }
  .ua-exclude-ta, .ipinfo-input {
    width: 100%; font-family: monospace; font-size: 0.78rem;
    padding: 0.5rem; background: var(--bg-primary); border: 1px solid var(--border); border-radius: 8px;
    color: var(--text-primary);
  }
  .ua-exclude-ta { min-height: 4rem; resize: vertical; }
  .ipinfo-input { max-width: 400px; }
  .hint a { color: var(--accent); text-decoration: none; }
  .hint a:hover { text-decoration: underline; }
</style>
