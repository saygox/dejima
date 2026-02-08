import { SendKeyEvent } from '../../../wailsjs/go/main/App';

let capturing = false;
// Track currently pressed keys so we can release them all on capture exit
const pressedKeys = new Set<string>();

function onKeyDown(e: KeyboardEvent) {
  // Always prevent default to suppress beep on macOS WebView
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  // Ignore key repeat events — only send the initial press
  if (e.repeat) return;
  pressedKeys.add(e.code);
  SendKeyEvent(e.code, true).catch(console.error);
}

function onKeyUp(e: KeyboardEvent) {
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  pressedKeys.delete(e.code);
  SendKeyEvent(e.code, false).catch(console.error);
}

/** Release all currently pressed keys on the remote side. */
export function releaseAllKeys() {
  for (const code of pressedKeys) {
    SendKeyEvent(code, false).catch(console.error);
  }
  pressedKeys.clear();
}

function onFocus() {
  capturing = true;
}

function onBlur() {
  // Release all held keys when element loses focus
  releaseAllKeys();
  capturing = false;
}

export function startKeyboardCapture(element: HTMLElement) {
  element.addEventListener('keydown', onKeyDown);
  element.addEventListener('keyup', onKeyUp);
  element.addEventListener('focus', onFocus);
  element.addEventListener('blur', onBlur);
  // If element already has focus, start capturing
  if (document.activeElement === element) {
    capturing = true;
  }
}

export function stopKeyboardCapture(element: HTMLElement) {
  releaseAllKeys();
  capturing = false;
  element.removeEventListener('keydown', onKeyDown);
  element.removeEventListener('keyup', onKeyUp);
  element.removeEventListener('focus', onFocus);
  element.removeEventListener('blur', onBlur);
}
