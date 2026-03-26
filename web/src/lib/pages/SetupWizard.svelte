<script lang="ts">
  import { api } from '../api';

  let step = 1;
  let detection: any = null;
  let detecting = false;
  let saving = false;
  let error = '';

  let username = 'admin';
  let password = '';
  let passwordConfirm = '';

  let smtpHost = '';
  let smtpPort = '587';
  let smtpUser = '';
  let smtpPass = '';
  let smtpFrom = '';

  let selectedLogSources: Set<string> = new Set();

  async function runDetection() {
    detecting = true;
    try {
      detection = await api.detect();
    } catch (e: any) {
      error = e.message;
    } finally {
      detecting = false;
    }
  }

  function toggleLogSource(path: string) {
    if (selectedLogSources.has(path)) {
      selectedLogSources.delete(path);
    } else {
      selectedLogSources.add(path);
    }
    selectedLogSources = new Set(selectedLogSources);
  }

  async function createAccount() {
    error = '';
    if (password.length < 8) {
      error = 'Password must be at least 8 characters.';
      return;
    }
    if (password !== passwordConfirm) {
      error = 'Passwords do not match.';
      return;
    }
    saving = true;
    try {
      const res = await api.setup(username, password);
      api.setCsrfToken(res.csrf_token);
      step = 2;
    } catch (e: any) {
      error = e.message;
    } finally {
      saving = false;
    }
  }

  async function finishSetup() {
    saving = true;
    error = '';
    try {
      for (const path of selectedLogSources) {
        const name = path.split('/').pop() || path;
        await api.createLogSource({
          name,
          type: 'file',
          path,
          format: 'auto',
          server_id: 'local',
          enabled: true,
        });
      }

      if (smtpHost) {
        await api.createNotificationChannel({
          name: 'Email',
          type: 'email',
          config: JSON.stringify({
            host: smtpHost,
            port: parseInt(smtpPort),
            username: smtpUser,
            password: smtpPass,
            from: smtpFrom,
          }),
        });
      }

      window.location.reload();
    } catch (e: any) {
      error = e.message;
    } finally {
      saving = false;
    }
  }
</script>

<div class="wizard-overlay">
  <div class="wizard">
    <div class="wizard-header">
      <h1>Welcome to TraceLog</h1>
      <p>Let's set up monitoring for your server.</p>
    </div>

    <div class="steps">
      <div class="step-indicator">
        {#each [1, 2, 3, 4] as s}
          <div class="step-dot" class:active={step >= s} class:current={step === s}>{s}</div>
          {#if s < 4}<div class="step-line" class:active={step > s}></div>{/if}
        {/each}
      </div>
    </div>

    {#if step === 1}
      <div class="step-content">
        <h2>Create Admin Account</h2>
        <p>Set up the administrator account for TraceLog.</p>

        <div class="form-grid">
          <label class="full">
            <span>Username</span>
            <input type="text" bind:value={username} placeholder="admin" />
          </label>
          <label>
            <span>Password</span>
            <input type="password" bind:value={password} placeholder="Min 8 characters" />
          </label>
          <label>
            <span>Confirm Password</span>
            <input type="password" bind:value={passwordConfirm} />
          </label>
        </div>

        {#if error}
          <div class="error-msg">{error}</div>
        {/if}

        <button class="btn primary" on:click={createAccount} disabled={saving || !username || !password}>
          {saving ? 'Creating...' : 'Create Account'}
        </button>
      </div>

    {:else if step === 2}
      <div class="step-content">
        <h2>Auto-Detection</h2>
        <p>TraceLog will scan your server for services, log files, and Docker containers.</p>

        {#if !detection}
          <button class="btn primary" on:click={runDetection} disabled={detecting}>
            {detecting ? 'Scanning...' : 'Scan Server'}
          </button>
        {:else}
          <div class="detection-results">
            {#if detection.docker}
              <div class="detect-item success">Docker detected</div>
            {:else}
              <div class="detect-item muted">Docker not found</div>
            {/if}

            {#if detection.web_server}
              <div class="detect-item success">Web server: {detection.web_server}</div>
            {/if}

            {#if detection.log_files?.length > 0}
              <h3>Discovered Log Files</h3>
              <div class="log-list">
                {#each detection.log_files as lf}
                  <label class="log-item">
                    <input type="checkbox" checked={selectedLogSources.has(lf)} on:change={() => toggleLogSource(lf)} />
                    <span>{lf}</span>
                  </label>
                {/each}
              </div>
            {/if}

            {#if detection.processes?.length > 0}
              <h3>Detected Services</h3>
              <div class="process-list">
                {#each detection.processes as p}
                  <span class="process-tag">{p}</span>
                {/each}
              </div>
            {/if}
          </div>

          <button class="btn primary" on:click={() => step = 3}>Next</button>
        {/if}
      </div>

    {:else if step === 3}
      <div class="step-content">
        <h2>Email Notifications (Optional)</h2>
        <p>Configure SMTP to receive alert emails. You can skip this and set it up later.</p>

        <div class="form-grid">
          <label>
            <span>SMTP Host</span>
            <input type="text" bind:value={smtpHost} placeholder="smtp.gmail.com" />
          </label>
          <label>
            <span>Port</span>
            <input type="text" bind:value={smtpPort} placeholder="587" />
          </label>
          <label>
            <span>Username</span>
            <input type="text" bind:value={smtpUser} placeholder="user@example.com" />
          </label>
          <label>
            <span>Password</span>
            <input type="password" bind:value={smtpPass} />
          </label>
          <label class="full">
            <span>From Address</span>
            <input type="email" bind:value={smtpFrom} placeholder="tracelog@example.com" />
          </label>
        </div>

        <div class="btn-row">
          <button class="btn secondary" on:click={() => step = 2}>Back</button>
          <button class="btn secondary" on:click={() => { smtpHost = ''; step = 4; }}>Skip</button>
          <button class="btn primary" on:click={() => step = 4}>Next</button>
        </div>
      </div>

    {:else if step === 4}
      <div class="step-content">
        <h2>Ready to Go</h2>
        <p>TraceLog will start collecting system metrics immediately. Here's what's configured:</p>

        <div class="summary">
          <div class="summary-item">System metrics: <strong>CPU, Memory, Disk, Network, Load</strong></div>
          <div class="summary-item">Docker monitoring: <strong>{detection?.docker ? 'Enabled' : 'Disabled'}</strong></div>
          <div class="summary-item">Process monitoring: <strong>Enabled</strong></div>
          <div class="summary-item">Log sources: <strong>{selectedLogSources.size || 'None configured'}</strong></div>
          <div class="summary-item">Email notifications: <strong>{smtpHost ? 'Configured' : 'Not configured'}</strong></div>
          <div class="summary-item">Data retention: <strong>30 days</strong></div>
        </div>

        {#if error}
          <div class="error-msg">{error}</div>
        {/if}

        <div class="btn-row">
          <button class="btn secondary" on:click={() => step = 3}>Back</button>
          <button class="btn primary" on:click={finishSetup} disabled={saving}>
            {saving ? 'Setting up...' : 'Finish Setup'}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .wizard-overlay {
    position: fixed; inset: 0; background: var(--bg-primary);
    display: flex; align-items: center; justify-content: center;
    padding: 1rem; z-index: 1000;
  }
  .wizard {
    background: var(--bg-secondary); border: 1px solid var(--border);
    border-radius: 16px; padding: 2rem; max-width: 560px; width: 100%;
  }
  .wizard-header { text-align: center; margin-bottom: 1.5rem; }
  .wizard-header h1 {
    font-size: 1.5rem; margin: 0 0 0.35rem;
    background: linear-gradient(135deg, #58a6ff, #bc8cff);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  }
  .wizard-header p { color: var(--text-muted); font-size: 0.9rem; margin: 0; }

  .steps { margin-bottom: 1.5rem; }
  .step-indicator { display: flex; align-items: center; justify-content: center; gap: 0; }
  .step-dot {
    width: 28px; height: 28px; border-radius: 50%;
    border: 2px solid var(--border); color: var(--text-muted);
    display: flex; align-items: center; justify-content: center;
    font-size: 0.75rem; font-weight: 600;
  }
  .step-dot.active { border-color: var(--accent); color: var(--accent); }
  .step-dot.current { background: var(--accent); color: #fff; border-color: var(--accent); }
  .step-line { width: 40px; height: 2px; background: var(--border); }
  .step-line.active { background: var(--accent); }

  h2 { font-size: 1.1rem; margin: 0 0 0.35rem; color: var(--text-primary); }
  h3 { font-size: 0.85rem; margin: 1rem 0 0.5rem; color: var(--text-secondary); }
  .step-content p { color: var(--text-muted); font-size: 0.85rem; margin: 0 0 1rem; }

  .btn {
    padding: 0.55rem 1.2rem; border-radius: 8px; border: none;
    font-size: 0.85rem; font-weight: 600; cursor: pointer; transition: all 0.15s;
  }
  .btn.primary { background: var(--accent); color: #fff; }
  .btn.primary:hover { opacity: 0.9; }
  .btn.primary:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn.secondary { background: var(--bg-hover); color: var(--text-secondary); border: 1px solid var(--border); }
  .btn.secondary:hover { color: var(--text-primary); }
  .btn-row { display: flex; gap: 0.5rem; margin-top: 1.5rem; justify-content: flex-end; }

  .detection-results { margin-bottom: 1rem; }
  .detect-item {
    padding: 0.4rem 0.6rem; border-radius: 6px; margin-bottom: 0.25rem;
    font-size: 0.8rem; background: var(--bg-hover);
  }
  .detect-item.success { color: var(--success); }
  .detect-item.muted { color: var(--text-muted); }

  .log-list { display: flex; flex-direction: column; gap: 0.25rem; }
  .log-item {
    display: flex; align-items: center; gap: 0.5rem;
    font-size: 0.8rem; color: var(--text-secondary); padding: 0.25rem 0;
  }
  .log-item input { accent-color: var(--accent); }

  .process-list { display: flex; flex-wrap: wrap; gap: 0.35rem; }
  .process-tag {
    padding: 0.2rem 0.5rem; border-radius: 4px;
    background: #58a6ff22; color: var(--accent); font-size: 0.75rem;
  }

  .form-grid {
    display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;
  }
  .form-grid label { display: flex; flex-direction: column; gap: 0.25rem; }
  .form-grid label.full { grid-column: 1 / -1; }
  .form-grid span { font-size: 0.75rem; color: var(--text-muted); font-weight: 500; }
  .form-grid input {
    padding: 0.45rem 0.6rem; background: var(--bg-primary); border: 1px solid var(--border);
    border-radius: 6px; color: var(--text-primary); font-size: 0.85rem;
  }
  .form-grid input:focus { outline: none; border-color: var(--accent); }

  .summary { margin-bottom: 0.5rem; }
  .summary-item {
    padding: 0.35rem 0; font-size: 0.85rem; color: var(--text-secondary);
    border-bottom: 1px solid var(--border);
  }
  .summary-item:last-child { border-bottom: none; }
  .summary-item strong { color: var(--text-primary); }

  .error-msg {
    padding: 0.5rem 0.75rem; background: #f8514922; color: var(--danger);
    border-radius: 6px; font-size: 0.8rem; margin-top: 0.75rem;
  }
</style>
