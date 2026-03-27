<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { api } from '../../api';

  export let servers: any[] = [];

  const dispatch = createEventDispatcher();

  /** Editable name / host / notes per server (Settings → Servers); notes appear in alert emails. */
  let serverDrafts: Record<string, { name: string; host: string; notes: string }> = {};

  $: {
    const next = { ...serverDrafts };
    let changed = false;
    for (const s of servers) {
      if (!(s.id in next)) {
        next[s.id] = { name: s.name, host: s.host || '', notes: s.notes || '' };
        changed = true;
      }
    }
    if (changed) serverDrafts = next;
  }

  let saved = false;

  async function saveServerRow(id: string) {
    const d = serverDrafts[id];
    if (!d || !String(d.name || '').trim()) {
      alert('Name is required');
      return;
    }
    try {
      const updated = await api.updateServer(id, {
        name: d.name.trim(),
        host: (d.host || '').trim(),
        notes: (d.notes || '').trim(),
      });
      servers = (servers || []).map((x: any) => (x.id === id ? updated : x));
      serverDrafts = {
        ...serverDrafts,
        [id]: { name: updated.name, host: updated.host || '', notes: updated.notes || '' },
      };
      saved = true;
      setTimeout(() => {
        saved = false;
      }, 2000);
      dispatch('serversChanged');
    } catch (e: any) {
      alert('Failed: ' + e.message);
    }
  }

  async function removeServer(id: string) {
    if (!confirm('Delete this server and all its data?')) return;
    await api.deleteServer(id);
    const next = { ...serverDrafts };
    delete next[id];
    serverDrafts = next;
    servers = (await api.listServers()) || [];
    dispatch('serversChanged');
  }
</script>

<div class="section">
  <h3>Connected Servers</h3>
  <p class="hint">Each server is an agent (or the local node in <code>serve</code> mode). The <strong>API key</strong> is used by <code>tracelog agent --hub … --key …</code>. Deleting a server removes its metrics and stored logs for that server ID from TraceLog's database.</p>
  <p class="hint"><strong>Registered host</strong> and <strong>note</strong> are free text: use a public hostname, IP, or label you recognize in <strong>alert emails</strong> (e.g. avoid leaving every host as <code>localhost</code> when you monitor many machines).</p>
  <p class="hint">Most tabs in Settings are <strong>hub-wide</strong>. <strong>Log Sources</strong> can target the local <code>serve</code> agent or a <strong>remote</strong> server row (see Log Sources tab); remote agents pull their file list from the hub periodically.</p>
  {#if servers.length === 0}
    <p class="hint">No servers registered.</p>
  {:else}
    <div class="item-list server-edit-list">
      {#each servers as srv (srv.id)}
        {#if serverDrafts[srv.id]}
          <div class="item-row server-edit-row">
            <div class="server-edit-fields">
              <label class="server-field"
                >Name <input type="text" bind:value={serverDrafts[srv.id].name} /></label
              >
              <label class="server-field"
                >Registered host
                <input
                  type="text"
                  bind:value={serverDrafts[srv.id].host}
                  placeholder="e.g. prod.example.com or 10.0.1.5"
                /></label
              >
              <label class="server-field server-field-notes"
                >Note (in alert emails)
                <textarea
                  bind:value={serverDrafts[srv.id].notes}
                  rows="2"
                  placeholder="e.g. Hetzner NBG1 — cadourile.ro stack"
                ></textarea></label
              >
              <span class="item-detail">{srv.status} — API Key: <code class="api-key">{srv.api_key}</code></span>
            </div>
            <div class="server-edit-actions">
              <button type="button" class="btn-primary" on:click={() => saveServerRow(srv.id)}>Save</button>
              <button type="button" class="btn-delete" on:click={() => removeServer(srv.id)}>Delete</button>
            </div>
          </div>
        {/if}
      {/each}
    </div>
  {/if}
</div>

<style>
  .server-edit-list { gap: 0.75rem; }
  .server-edit-row { align-items: flex-start; flex-wrap: wrap; gap: 0.75rem; }
  .server-edit-fields { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 0.45rem; }
  .server-field { display: flex; flex-direction: column; gap: 0.25rem; font-size: 0.75rem; color: var(--text-secondary); }
  .server-field input,
  .server-field textarea {
    padding: 0.45rem 0.55rem; background: var(--bg-secondary); border: 1px solid var(--border); border-radius: 6px;
    color: var(--text-primary); font-size: 0.85rem; width: 100%; box-sizing: border-box;
  }
  .server-field-notes textarea { font-size: 0.82rem; resize: vertical; min-height: 2.5rem; }
  .server-edit-actions { display: flex; flex-direction: column; gap: 0.4rem; align-items: stretch; }
  .btn-primary {
    padding: 0.45rem 0.9rem; background: #238636; color: #fff; border: none; border-radius: 8px;
    cursor: pointer; font-weight: 600; font-size: 0.8rem;
  }
  .btn-primary:hover { filter: brightness(1.08); }
  .api-key { font-family: monospace; font-size: 0.7rem; }
</style>
