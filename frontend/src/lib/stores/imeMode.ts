import { writable } from 'svelte/store';

/** false = type (wtype/xdotool), true = paste (clipboard + Ctrl+V) */
export const pasteMode = writable(true);
