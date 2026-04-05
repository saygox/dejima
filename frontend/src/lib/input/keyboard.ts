import { SendKeyEvent, SendText, GetRemoteClipboard, ResolveClipboardForPaste, MarkSentToRemote, WriteRemoteClipToHost } from '../../../wailsjs/go/main/App';

let capturing = false;
// Track currently pressed keys so we can release them all on capture exit
const pressedKeys = new Set<string>();

const MODIFIER_CODES = new Set([
  'MetaLeft', 'MetaRight', 'ControlLeft', 'ControlRight',
  'AltLeft', 'AltRight', 'ShiftLeft', 'ShiftRight',
]);

// Ctrl+Tab → Alt+Tab window-switcher state.
// While Ctrl is held, we keep AltLeft pressed on the remote so the user can
// press Tab repeatedly to cycle through windows.  AltLeft is released when
// Ctrl is released (handled in onKeyUp).
let altTabActive = false;

async function releaseHeldModifiers(): Promise<void> {
  for (const code of pressedKeys) {
    // Skip Meta — we never send it to remote (suppressed in onKeyDown)
    if (MODIFIER_CODES.has(code) && code !== 'MetaLeft' && code !== 'MetaRight') {
      await SendKeyEvent(code, false);
    }
  }
}

async function handleCtrlV(): Promise<void> {
  console.log('[keyboard] handleCtrlV called');
  try {
    const text = await ResolveClipboardForPaste();
    console.log('[keyboard] ResolveClipboardForPaste returned:', JSON.stringify(text?.substring(0, 50)), 'len=', text?.length);
    if (text) {
      await releaseHeldModifiers();
      console.log('[keyboard] calling SendText, len=', text.length);
      await SendText(text, true);
      console.log('[keyboard] SendText succeeded');
      await MarkSentToRemote(text);
    } else {
      console.log('[keyboard] text empty, sending raw Ctrl+V');
      await releaseHeldModifiers();
      await SendKeyEvent('ControlLeft', true);
      await SendKeyEvent('KeyV', true);
      await SendKeyEvent('KeyV', false);
      await SendKeyEvent('ControlLeft', false);
    }
  } catch (err) {
    console.error('[keyboard] handleCtrlV error:', err);
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('KeyV', true);
    await SendKeyEvent('KeyV', false);
    await SendKeyEvent('ControlLeft', false);
  }
}

async function handleCtrlShiftV(): Promise<void> {
  try {
    const text = await ResolveClipboardForPaste();
    if (text) {
      await releaseHeldModifiers();
      await SendText(text, true);
      await MarkSentToRemote(text);
    } else {
      await releaseHeldModifiers();
      await SendKeyEvent('ControlLeft', true);
      await SendKeyEvent('ShiftLeft', true);
      await SendKeyEvent('KeyV', true);
      await SendKeyEvent('KeyV', false);
      await SendKeyEvent('ShiftLeft', false);
      await SendKeyEvent('ControlLeft', false);
    }
  } catch (err) {
    console.error('[keyboard] handleCtrlShiftV error:', err);
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('ShiftLeft', true);
    await SendKeyEvent('KeyV', true);
    await SendKeyEvent('KeyV', false);
    await SendKeyEvent('ShiftLeft', false);
    await SendKeyEvent('ControlLeft', false);
  }
}

async function handleCtrlShiftC(): Promise<void> {
  try {
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('ShiftLeft', true);
    await SendKeyEvent('KeyC', true);
    await SendKeyEvent('KeyC', false);
    await SendKeyEvent('ShiftLeft', false);
    await SendKeyEvent('ControlLeft', false);
  } catch (err) { console.error('[keyboard] handleCtrlShiftC error:', err); }
  scheduleClipboardSync();
}

async function handleCtrlC(): Promise<void> {
  try {
    await releaseHeldModifiers();
    await SendKeyEvent('ControlLeft', true);
    await SendKeyEvent('KeyC', true);
    await SendKeyEvent('KeyC', false);
    await SendKeyEvent('ControlLeft', false);
  } catch (err) { console.error('[keyboard] handleCtrlC error:', err); }
  scheduleClipboardSync();
}

let clipSyncTimer: ReturnType<typeof setTimeout> | null = null;

function scheduleClipboardSync() {
  if (clipSyncTimer) clearTimeout(clipSyncTimer);
  clipSyncTimer = setTimeout(async () => {
    try {
      const text = await GetRemoteClipboard();
      if (text) await WriteRemoteClipToHost(text);
    } catch (err) { console.error('[keyboard] scheduleClipboardSync error:', err); }
    clipSyncTimer = null;
  }, 300);
}

function onKeyDown(e: KeyboardEvent) {
  // During IME composition, don't send raw key events
  if (e.isComposing || e.keyCode === 229) return;
  // F11 is handled by App.svelte global handler — never forward to remote
  if (e.code === 'F11') return;
  // Always prevent default to suppress beep on macOS WebView
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  // Ignore key repeat events — only send the initial press
  if (e.repeat) return;
  // Shift+Escape is used to exit capture — never send to remote
  if (e.code === 'Escape' && e.shiftKey) return;
  pressedKeys.add(e.code);

  // Cmd+Shift+V: sync host clipboard → remote terminal paste (Ctrl+Shift+V)
  if (e.code === 'KeyV' && e.shiftKey && (e.metaKey || e.ctrlKey)) {
    handleCtrlShiftV();
    return;
  }

  // Cmd+Shift+C: send Ctrl+Shift+C to remote terminal, then sync clipboard
  if (e.code === 'KeyC' && e.shiftKey && (e.metaKey || e.ctrlKey)) {
    handleCtrlShiftC();
    return;
  }

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

  // Ctrl+Tab / Ctrl+Shift+Tab → Alt+Tab / Alt+Shift+Tab (window switch on remote Linux)
  // Hold AltLeft while Ctrl remains held so user can press Tab repeatedly to cycle windows.
  // AltLeft is released when Ctrl is released (see onKeyUp).
  if (e.code === 'Tab' && e.ctrlKey && !e.metaKey) {
    (async () => {
      if (!altTabActive) {
        await releaseHeldModifiers();
        await SendKeyEvent('AltLeft', true);
        altTabActive = true;
      }
      if (e.shiftKey) await SendKeyEvent('ShiftLeft', true);
      await SendKeyEvent('Tab', true);
      await SendKeyEvent('Tab', false);
      if (e.shiftKey) await SendKeyEvent('ShiftLeft', false);
    })().catch(console.error);
    return;
  }

  // Cmd+Option+Arrow Left/Right → Ctrl+Shift+Tab / Ctrl+Tab (browser tab switch on Linux)
  if (e.metaKey && e.altKey && (e.code === 'ArrowLeft' || e.code === 'ArrowRight')) {
    const combo = e.code === 'ArrowLeft'
      ? [['ControlLeft', true], ['ShiftLeft', true], ['Tab', true], ['Tab', false], ['ShiftLeft', false], ['ControlLeft', false]] as const
      : [['ControlLeft', true], ['Tab', true], ['Tab', false], ['ControlLeft', false]] as const;
    (async () => {
      await releaseHeldModifiers();
      for (const [code, down] of combo) await SendKeyEvent(code, down);
    })().catch(console.error);
    return;
  }

  // Cmd+Arrow Left/Right → Alt+Arrow (browser back/forward on Linux)
  if (e.metaKey && !e.altKey && (e.code === 'ArrowLeft' || e.code === 'ArrowRight')) {
    SendKeyEvent('AltLeft', true)
      .then(() => SendKeyEvent(e.code, true))
      .then(() => SendKeyEvent(e.code, false))
      .then(() => SendKeyEvent('AltLeft', false))
      .catch(console.error);
    return;
  }

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
  if (e.isComposing || e.keyCode === 229) return;
  // F11 is handled by App.svelte global handler — never forward to remote
  if (e.code === 'F11') return;
  e.preventDefault();
  e.stopPropagation();
  if (!capturing) return;
  pressedKeys.delete(e.code);

  // Don't send Meta release — we suppressed the press too
  if (e.code !== 'MetaLeft' && e.code !== 'MetaRight') {
    // Ctrl release while Alt+Tab window-switcher is active → release AltLeft on remote
    if ((e.code === 'ControlLeft' || e.code === 'ControlRight') && altTabActive) {
      altTabActive = false;
      SendKeyEvent('AltLeft', false).catch(console.error);
      // Don't send Ctrl release — we never sent Ctrl down (we sent Alt instead)
    } else {
      SendKeyEvent(e.code, false).catch(console.error);
    }
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
  if (altTabActive) {
    altTabActive = false;
    SendKeyEvent('AltLeft', false).catch(console.error);
  }
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
  if (clipSyncTimer) {
    clearTimeout(clipSyncTimer);
    clipSyncTimer = null;
  }
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
