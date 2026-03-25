import { writable } from 'svelte/store';

interface User {
  id: string;
  username: string;
}

export const user = writable<User | null>(null);
export const isAuthenticated = writable(false);
export const darkMode = writable(true);
export const currentPage = writable('overview');
