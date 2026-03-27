function apiBase(): string {
  if (typeof document === 'undefined') return '/api';
  const raw = document.querySelector('meta[name="tracelog-url-prefix"]')?.getAttribute('content');
  if (!raw || raw === '__TRACELOG_URL_PREFIX__') return '/api';
  const prefix = raw.replace(/\/$/, '');
  return `${prefix}/api`;
}

let csrfToken = '';

async function request(method: string, path: string, body?: unknown) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (csrfToken && method !== 'GET') {
    headers['X-CSRF-Token'] = csrfToken;
  }

  const res = await fetch(`${apiBase()}${path}`, {
    method,
    headers,
    credentials: 'same-origin',
    body: body ? JSON.stringify(body) : undefined,
  });

  const data = await res.json();
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

export const api = {
  setCsrfToken(token: string) { csrfToken = token; },

  // Auth
  setup: (username: string, password: string) => request('POST', '/auth/setup', { username, password }),
  login: (username: string, password: string) => request('POST', '/auth/login', { username, password }),
  logout: () => request('POST', '/auth/logout'),
  me: () => request('GET', '/auth/me'),

  /** Re-enter hub login password; downloads a SQLite snapshot (VACUUM INTO) of TraceLog’s database. */
  exportDatabase: async (password: string) => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
    const res = await fetch(`${apiBase()}/database/export`, {
      method: 'POST',
      headers,
      credentials: 'same-origin',
      body: JSON.stringify({ password }),
    });
    const ct = res.headers.get('Content-Type') || '';
    if (!res.ok) {
      let msg = `HTTP ${res.status}`;
      if (ct.includes('application/json')) {
        try {
          const j = await res.json();
          if (j.error) msg = j.error;
        } catch {
          /* ignore */
        }
      }
      throw new Error(msg);
    }
    const blob = await res.blob();
    const disp = res.headers.get('Content-Disposition') || '';
    const m = /filename\*=UTF-8''([^;\n]+)|filename="([^"]+)"|filename=([^;\n]+)/i.exec(disp);
    let name = 'tracelog-backup.db';
    if (m) {
      const raw = (m[1] || m[2] || m[3] || '').trim();
      try {
        name = decodeURIComponent(raw);
      } catch {
        name = raw;
      }
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = name;
    a.click();
    URL.revokeObjectURL(url);
  },

  // Servers
  listServers: () => request('GET', '/servers'),
  getServer: (id: string) => request('GET', `/servers/${id}`),
  getMetrics: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/metrics?range=${range_}`),
  getDockerMetrics: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/docker?range=${range_}`),
  getDockerLogs: (id: string, container: string, tail = 500) =>
    request('GET', `/servers/${id}/docker/logs?container=${encodeURIComponent(container)}&tail=${tail}`),
  createServer: (name: string, host: string) => request('POST', '/servers', { name, host }),
  deleteServer: (id: string) => request('DELETE', `/servers/${id}`),

  // Logs
  getLogs: (
    serverId: string,
    opts: {
      source?: string;
      level?: string;
      /** Minimum severity: error|warn|info|debug (includes more severe levels). */
      severity_min?: string;
      search?: string;
      range?: string;
    } = {},
  ) => {
    const params = new URLSearchParams({ server_id: serverId });
    if (opts.source) params.set('source', opts.source);
    if (opts.level && opts.level !== 'all') params.set('level', opts.level);
    if (opts.severity_min && opts.severity_min !== 'all') params.set('severity_min', opts.severity_min);
    if (opts.search) params.set('search', opts.search);
    if (opts.range) params.set('range', opts.range);
    return request('GET', `/logs?${params}`);
  },
  /** Removes ingested log rows in TraceLog’s database (not files on the server disk). */
  purgeIngestedLogs: (body: { server_id: string; mode: 'all' | 'older_than'; range?: string; source?: string }) =>
    request('POST', '/logs/purge', body),

  // Log Sources
  listLogSources: () => request('GET', '/log-sources'),
  createLogSource: (data: any) => request('POST', '/log-sources', data),
  deleteLogSource: (id: string) => request('DELETE', `/log-sources/${id}`),

  // Settings
  getSettings: () => request('GET', '/settings'),
  updateSettings: (data: unknown) => request('PUT', '/settings', data),

  // Uptime
  listUptimeChecks: () => request('GET', '/uptime'),
  createUptimeCheck: (data: any) => request('POST', '/uptime', data),
  deleteUptimeCheck: (id: string) => request('DELETE', `/uptime/${id}`),
  getUptimeResults: (id: string, range_: string = '24h') => request('GET', `/uptime/${id}/results?range=${range_}`),

  // Alerts
  listAlertRules: () => request('GET', '/alerts'),
  createAlertRule: (data: any) => request('POST', '/alerts', data),
  deleteAlertRule: (id: string) => request('DELETE', `/alerts/${id}`),

  listLogAlertSilences: () => request('GET', '/log-alert-silences'),
  createLogAlertSilence: (data: { pattern: string; server_id?: string; rule_metric?: string }) =>
    request('POST', '/log-alert-silences', data),
  deleteLogAlertSilence: (id: string) => request('DELETE', `/log-alert-silences/${id}`),

  // Notifications
  listNotificationChannels: () => request('GET', '/notifications'),
  createNotificationChannel: (data: any) => request('POST', '/notifications', data),
  deleteNotificationChannel: (id: string) => request('DELETE', `/notifications/${id}`),
  testNotificationChannel: (id: string) => request('POST', `/notifications/${id}/test`),

  // Processes
  getProcesses: (id: string, latest = false) => request('GET', `/servers/${id}/processes?latest=${latest}`),
  getProcessHistory: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/processes?range=${range_}`),

  // Access Logs / HTTP Analytics
  getAccessStats: (id: string, range_: string = '24h', topN?: number) => {
    const q = new URLSearchParams({ range: range_ });
    if (topN != null && topN > 0) q.set('top_n', String(topN));
    return request('GET', `/servers/${id}/access-stats?${q}`);
  },
  getRecentAccessLogs: (id: string) => request('GET', `/servers/${id}/access-logs`),
  getAccessBadRequests: (id: string, opts: { range?: string; ip?: string; limit?: number } = {}) => {
    const q = new URLSearchParams();
    if (opts.range) q.set('range', opts.range);
    if (opts.ip) q.set('ip', opts.ip);
    if (opts.limit) q.set('limit', String(opts.limit));
    const suffix = q.toString() ? `?${q}` : '';
    return request('GET', `/servers/${id}/access-bad-requests${suffix}`);
  },
  getAccessIPPolicy: () => request('GET', '/access-ip-policy'),
  putAccessIPPolicy: (ips: string[]) => request('PUT', '/access-ip-policy', { ips }),

  // Detection
  detect: () => request('GET', '/detect'),

  // Health
  health: () => request('GET', '/health'),
};
