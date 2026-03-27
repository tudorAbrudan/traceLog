<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api';

  let aboutVersion = '';

  onMount(async () => {
    try {
      const h = await api.health();
      aboutVersion = h?.version || 'unknown';
    } catch {
      aboutVersion = 'unknown';
    }
  });
</script>

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

<style>
  .about-grid { display: flex; flex-direction: column; gap: 0.75rem; }
  .about-item { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid var(--border); font-size: 0.85rem; }
  .about-item span { color: var(--text-muted); }
  .about-item strong, .about-item a { color: var(--text-primary); }
</style>
