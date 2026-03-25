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
  login: (username: string, password: string) => request('POST', '/auth/login', { username, password }),
  logout: () => request('POST', '/auth/logout'),
  me: () => request('GET', '/auth/me'),

  // Servers
  listServers: () => request('GET', '/servers'),
  getServer: (id: string) => request('GET', `/servers/${id}`),
  getMetrics: (id: string, range_: string = '1h') => request('GET', `/servers/${id}/metrics?range=${range_}`),
  createServer: (name: string, host: string) => request('POST', '/servers', { name, host }),

  // Settings
  getSettings: () => request('GET', '/settings'),
  updateSettings: (data: unknown) => request('PUT', '/settings', data),
  getLogSources: () => request('GET', '/settings/log-sources'),

  // Health
  health: () => request('GET', '/health'),
};
