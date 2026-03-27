import { writable } from 'svelte/store';

const CONTEXT_SERVER_KEY = 'tracelog-context-server-id';

function readContextServerId(): string {
  if (typeof sessionStorage === 'undefined') return '';
  try {
    return sessionStorage.getItem(CONTEXT_SERVER_KEY) || '';
  } catch {
    return '';
  }
}

interface User {
  id: string;
  username: string;
}

export const user = writable<User | null>(null);
export const isAuthenticated = writable(false);

const savedTheme = typeof localStorage !== 'undefined' ? localStorage.getItem('tracelog-theme') : null;
export const darkMode = writable(savedTheme !== 'light');

export const currentPage = writable('overview');

/** Last server focused from Overview / Server detail; preselects Logs / Processes / HTTP Analytics dropdowns. */
export const contextServerId = writable(readContextServerId());

if (typeof sessionStorage !== 'undefined') {
  contextServerId.subscribe((v) => {
    try {
      if (v) sessionStorage.setItem(CONTEXT_SERVER_KEY, v);
      else sessionStorage.removeItem(CONTEXT_SERVER_KEY);
    } catch {
      /* ignore */
    }
  });
}


/** When true, Overview shows the server grid even if there is only one server (user opened Overview or pressed Back). */
export const suppressSingleServerAutoOpen = writable(false);

/** Mobile nav drawer open (sidebar off-canvas). */
export const navDrawerOpen = writable(false);
