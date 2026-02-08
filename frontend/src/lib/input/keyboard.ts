import { SendKeyEvent } from '../../../wailsjs/go/main/App';

let capturing = false;
// Track currently pressed keys so we can release them all on capture exit
const pressedKeys = new Set<string>();

const MODIFIER_CODES = new Set([
  'MetaLeft', 'MetaRight', 'ControlLeft', 'ControlRight',
  'AltLeft', 'AltRight', 'ShiftLeft', 'ShiftRight',
]);

function onKeyDown(e: KeyboardEvent) {
  // During IME composition, don't send raw key events
  if (e.isComposing) return;
  // Always prevent default to suppress beep on macOS WebView
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  // Ignore key repeat events — only send the initial press
  if (e.repeat) return;
  // Escape is used to exit capture — never send to remote
  if (e.code === 'Escape') return;
  pressedKeys.add(e.code);
  SendKeyEvent(e.code, true).catch(console.error);
}

function onKeyUp(e: KeyboardEvent) {
  if (e.isComposing) return;
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  pressedKeys.delete(e.code);
  SendKeyEvent(e.code, false).catch(console.error);

  // macOS: keyup for letter keys does NOT fire while Cmd is held.
  // When a modifier is released, flush any non-modifier keys still
  // stuck in pressedKeys — their keyup events were swallowed by the OS.
  if (MODIFIER_CODES.has(e.code)) {
    for (const code of pressedKeys) {
      if (!MODIFIER_CODES.has(code)) {
        SendKeyEvent(code, false).catch(console.error);
        pressedKeys.delete(code);
      }
    }
  }
}

/** Release all currently pressed keys on the remote side. */
export function releaseAllKeys() {
  for (const code of pressedKeys) {
    SendKeyEvent(code, false).catch(console.error);
  }
  pressedKeys.clear();
}

let captureElement: HTMLElement | null = null;

function isFocusWithin(): boolean {
  if (!captureElement) return false;
  return captureElement.contains(document.activeElement);
}

function onFocusIn() {
  capturing = true;
}

function onFocusOut(e: FocusEvent) {
  // Only release if focus truly left the container (not just moved to child textarea)
  const related = e.relatedTarget as Node | null;
  if (captureElement && related && captureElement.contains(related)) {
    return; // Focus moved within container — stay capturing
  }
  releaseAllKeys();
  capturing = false;
}

export function startKeyboardCapture(element: HTMLElement) {
  captureElement = element;
  element.addEventListener('keydown', onKeyDown);
  element.addEventListener('keyup', onKeyUp);
  element.addEventListener('focusin', onFocusIn);
  element.addEventListener('focusout', onFocusOut);
  if (isFocusWithin()) {
    capturing = true;
  }
}

export function stopKeyboardCapture(element: HTMLElement) {
  releaseAllKeys();
  capturing = false;
  captureElement = null;
  element.removeEventListener('keydown', onKeyDown);
  element.removeEventListener('keyup', onKeyUp);
  element.removeEventListener('focusin', onFocusIn);
  element.removeEventListener('focusout', onFocusOut);
}
