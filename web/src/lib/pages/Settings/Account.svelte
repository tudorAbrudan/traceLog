<script lang="ts">
  import { api } from '../../api';
  import { user } from '../../store';

  let exportPassword = '';
  let exportBusy = false;
  let exportErr = '';
</script>

<div class="section">
  <h3>Account</h3>
  <p class="hint">Logged in as: <strong>{$user?.username}</strong></p>
  <p class="hint">To change password: <code>tracelog user reset-password {$user?.username}</code></p>

  <h4 class="subhead">Database backup</h4>
  <p class="hint">
    Download a SQLite snapshot of <strong>TraceLog's own hub database</strong> (metrics, log copies, users — same idea as <code>VACUUM INTO</code> / CLI <code>tracelog backup</code>). Enter your <strong>TraceLog login</strong> password to confirm. This is not a dump of an external app database (MySQL, Postgres, etc.).
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

<style>
  .subhead { font-size: 0.9rem; margin: 1.25rem 0 0.5rem 0; color: var(--text-primary); font-weight: 600; }
  .export-row { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; margin-top: 0.5rem; }
  .export-pass {
    padding: 0.5rem 0.65rem; min-width: 220px; flex: 1; max-width: 320px;
    background: var(--bg-primary); border: 1px solid var(--border); border-radius: 6px;
    color: var(--text-primary); font-size: 0.85rem;
  }
  .export-err { color: #f85149; font-size: 0.85rem; margin: 0.5rem 0 0 0; }
</style>
