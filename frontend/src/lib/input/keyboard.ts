import { SendKeyEvent, SendText, GetRemoteClipboard } from '../../../wailsjs/go/main/App';
import { ClipboardGetText, ClipboardSetText } from '../../../wailsjs/runtime/runtime';

let capturing = false;
// Track currently pressed keys so we can release them all on capture exit
const pressedKeys = new Set<string>();

const MODIFIER_CODES = new Set([
  'MetaLeft', 'MetaRight', 'ControlLeft', 'ControlRight',
  'AltLeft', 'AltRight', 'ShiftLeft', 'ShiftRight',
]);

async function releaseHeldModifiers(): Promise<void> {
  for (const code of pressedKeys) {
    // Skip Meta — we never send it to remote (suppressed in onKeyDown)
    if (MODIFIER_CODES.has(code) && code !== 'MetaLeft' && code !== 'MetaRight') {
      await SendKeyEvent(code, false);
    }
  }
}

async function handleCtrlV(): Promise<void> {
  try {
    const text = await ClipboardGetText();
    if (text) {
      await releaseHeldModifiers();
      await SendText(text, true);
    } else {
      // Empty clipboard → pass through Ctrl+V to remote
      await releaseHeldModifiers();
      await SendKeyEvent('ControlLeft', true);
      await SendKeyEvent('KeyV', true);
      await SendKeyEvent('KeyV', false);
      await SendKeyEvent('ControlLeft', false);
    }
  } catch {
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('KeyV', true);
    await SendKeyEvent('KeyV', false);
    await SendKeyEvent('ControlLeft', false);
  }
}

async function handleCtrlC(): Promise<void> {
  try {
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('KeyC', true);
    await SendKeyEvent('KeyC', false);
    await SendKeyEvent('ControlLeft', false);
  } catch { /* best effort */ }
  scheduleClipboardSync();
}

let clipSyncTimer: ReturnType<typeof setTimeout> | null = null;

function scheduleClipboardSync() {
  if (clipSyncTimer) clearTimeout(clipSyncTimer);
  clipSyncTimer = setTimeout(async () => {
    try {
      const text = await GetRemoteClipboard();
      if (text) await ClipboardSetText(text);
    } catch { /* silent */ }
    clipSyncTimer = null;
  }, 300);
}

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

  // Cmd+V / Ctrl+V: sync host clipboard → remote paste
  if (e.code === 'KeyV' && (e.metaKey || e.ctrlKey)) {
    handleCtrlV();
    return;
  }

  // Cmd+C / Ctrl+C: send Ctrl+C to remote, then sync remote clipboard → host
  if (e.code === 'KeyC' && (e.metaKey || e.ctrlKey)) {
    handleCtrlC();
    return;
  }

  // Don't send Meta (Cmd) to remote — it maps to Super/Win on Linux and
  // fires immediately before we know if Cmd+V/C will follow.
  // Use Tools > Send Key for Super if needed.
  if (e.code === 'MetaLeft' || e.code === 'MetaRight') return;

  // Cmd+key (not C/V, already handled above) → translate to Ctrl+key for remote
  if (e.metaKey) {
    SendKeyEvent('ControlLeft', true)
      .then(() => SendKeyEvent(e.code, true))
      .then(() => SendKeyEvent(e.code, false))
      .then(() => SendKeyEvent('ControlLeft', false))
      .catch(console.error);
    return;
  }

  SendKeyEvent(e.code, true).catch(console.error);
}

function onKeyUp(e: KeyboardEvent) {
  if (e.isComposing) return;
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  pressedKeys.delete(e.code);

  // Don't send Meta release — we suppressed the press too
  if (e.code !== 'MetaLeft' && e.code !== 'MetaRight') {
    SendKeyEvent(e.code, false).catch(console.error);
  }

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
