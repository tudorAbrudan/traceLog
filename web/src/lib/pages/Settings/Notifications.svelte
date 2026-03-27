<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api } from '../../api';

  const dispatch = createEventDispatcher();

  let channels: any[] = [];
  let newChName = ''; let newChType = 'email'; let newChConfig = '';
  let editingChannelId = '';
  let editChName = '';
  let editChType = 'email';
  let editChConfig = '';

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

  import { onMount } from 'svelte';

  onMount(async () => {
    channels = (await api.listNotificationChannels()) || [];
  });

  async function addChannel() {
    if (!newChName?.trim() || !newChConfig?.trim()) {
      alert('Enter a channel name and configuration JSON.');
      return;
    }
    try {
      await api.createNotificationChannel({ name: newChName, type: newChType, config: newChConfig });
      newChName = ''; newChConfig = '';
      channels = (await api.listNotificationChannels()) || [];
      dispatch('channelsChanged');
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
      dispatch('channelsChanged');
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function removeChannel(id: string) {
    if (editingChannelId === id) cancelEditChannel();
    await api.deleteNotificationChannel(id);
    channels = (await api.listNotificationChannels()) || [];
    dispatch('channelsChanged');
  }

  async function testChannel(id: string) {
    try {
      await api.testNotificationChannel(id);
      alert('Test notification sent!');
    } catch (e: any) { alert('Test failed: ' + e.message); }
  }
</script>

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
      <li>Go to <a href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer">App passwords</a> (or search "App passwords" in account settings if the link is hidden).</li>
      <li>Create a new app password (e.g. app: Mail, device: Other → "TraceLog"), copy the 16 characters into JSON — with or without spaces as Google shows them.</li>
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

<style>
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
</style>
