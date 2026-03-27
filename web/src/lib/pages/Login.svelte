<script lang="ts">
  import { api } from '../api';
  import { user, isAuthenticated } from '../store';

  let username = '';
  let password = '';
  let error = '';
  let loading = false;

  async function handleLogin() {
    error = '';
    loading = true;
    try {
      const res = await api.login(username, password);
      api.setCsrfToken(res.csrf_token);
      user.set(res.user);
      isAuthenticated.set(true);
    } catch (e: any) {
      error = e.message || 'Login failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-container">
  <div class="login-card">
    <div class="logo">
      <h1>TraceLog</h1>
      <p class="subtitle">Server Monitoring</p>
    </div>

    <form on:submit|preventDefault={handleLogin}>
      {#if error}
        <div class="error-msg">{error}</div>
      {/if}

      <div class="field">
        <label for="username">Username</label>
        <input id="username" type="text" bind:value={username} placeholder="admin" autocomplete="username" required />
      </div>

      <div class="field">
        <label for="password">Password</label>
        <input id="password" type="password" bind:value={password} placeholder="••••••••" autocomplete="current-password" required />
      </div>

      <button type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>
    <p class="rate-hint">Five failed sign-ins within one minute from your IP can block further attempts for about 15 minutes.</p>
  </div>
</div>

<style>
  .login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-primary);
  }
  .login-card {
    width: 100%;
    max-width: 380px;
    padding: 2.5rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 16px;
  }
  .logo {
    text-align: center;
    margin-bottom: 2rem;
  }
  .logo h1 {
    font-size: 1.8rem;
    background: linear-gradient(135deg, #58a6ff, #bc8cff);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    margin: 0;
  }
  .subtitle {
    color: var(--text-muted);
    font-size: 0.9rem;
    margin-top: 0.25rem;
  }
  .field {
    margin-bottom: 1rem;
  }
  label {
    display: block;
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin-bottom: 0.4rem;
  }
  input {
    width: 100%;
    padding: 0.65rem 0.75rem;
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 8px;
    color: var(--text-primary);
    font-size: 0.95rem;
    outline: none;
    transition: border-color 0.15s;
  }
  input:focus {
    border-color: #58a6ff;
  }
  button {
    width: 100%;
    padding: 0.7rem;
    margin-top: 0.5rem;
    background: #238636;
    color: #fff;
    border: none;
    border-radius: 8px;
    font-size: 0.95rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
  }
  button:hover:not(:disabled) {
    background: #2ea043;
  }
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  .error-msg {
    background: rgba(248, 81, 73, 0.1);
    border: 1px solid rgba(248, 81, 73, 0.4);
    color: #f85149;
    padding: 0.6rem 0.75rem;
    border-radius: 8px;
    font-size: 0.85rem;
    margin-bottom: 1rem;
  }
  .rate-hint {
    margin: 1rem 0 0 0;
    font-size: 0.72rem;
    color: var(--text-muted);
    line-height: 1.4;
  }
</style>
