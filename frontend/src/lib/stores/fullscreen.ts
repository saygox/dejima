import { writable, derived } from 'svelte/store';

export const isFullscreen = writable(false);
export const platform = writable('');
export const isWindows = derived(platform, ($p) => $p === 'windows');
