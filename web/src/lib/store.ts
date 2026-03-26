import { writable } from 'svelte/store';

interface User {
  id: string;
  username: string;
}

export const user = writable<User | null>(null);
export const isAuthenticated = writable(false);

const savedTheme = typeof localStorage !== 'undefined' ? localStorage.getItem('tracelog-theme') : null;
export const darkMode = writable(savedTheme !== 'light');

export const currentPage = writable('overview');
