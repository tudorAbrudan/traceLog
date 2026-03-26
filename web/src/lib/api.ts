const BASE = '/api';

let csrfToken = '';

async function request(method: string, path: string, body?: unknown) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (csrfToken && method !== 'GET') {
    headers['X-CSRF-Token'] = csrfToken;
  }

  const res = await fetch(`${BASE}${path}`, {
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

  // Servers
  listServers: () => request('GET', '/servers'),
  getServer: (id: string) => request('GET', `/servers/${id}`),
  getMetrics: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/metrics?range=${range_}`),
  getDockerMetrics: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/docker?range=${range_}`),
  createServer: (name: string, host: string) => request('POST', '/servers', { name, host }),
  deleteServer: (id: string) => request('DELETE', `/servers/${id}`),

  // Logs
  getLogs: (serverId: string, opts: { source?: string; level?: string; search?: string; range?: string } = {}) => {
    const params = new URLSearchParams({ server_id: serverId });
    if (opts.source) params.set('source', opts.source);
    if (opts.level && opts.level !== 'all') params.set('level', opts.level);
    if (opts.search) params.set('search', opts.search);
    if (opts.range) params.set('range', opts.range);
    return request('GET', `/logs?${params}`);
  },

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

  // Notifications
  listNotificationChannels: () => request('GET', '/notifications'),
  createNotificationChannel: (data: any) => request('POST', '/notifications', data),
  deleteNotificationChannel: (id: string) => request('DELETE', `/notifications/${id}`),
  testNotificationChannel: (id: string) => request('POST', `/notifications/${id}/test`),

  // Processes
  getProcesses: (id: string, latest = false) => request('GET', `/servers/${id}/processes?latest=${latest}`),
  getProcessHistory: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/processes?range=${range_}`),

  // Access Logs / HTTP Analytics
  getAccessStats: (id: string, range_: string = '24h') => request('GET', `/servers/${id}/access-stats?range=${range_}`),
  getRecentAccessLogs: (id: string) => request('GET', `/servers/${id}/access-logs`),

  // Detection
  detect: () => request('GET', '/detect'),

  // Health
  health: () => request('GET', '/health'),
};
