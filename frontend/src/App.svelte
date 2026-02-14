<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import VideoDisplay from './lib/components/VideoDisplay.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import AVSettingsModal from './lib/components/AVSettingsModal.svelte';
  import SerialSettingsModal from './lib/components/SerialSettingsModal.svelte';
  import { GetConfig, ListVideoDevices, StartVideo, ListSerialPorts, ConnectSerial, SendText, SendKeyEvent, GetRemoteClipboard, GetRemoteDiag, GetVideoDiag, WriteRemoteClipToHost, TestClipboardPipeline } from '../wailsjs/go/main/App';
  import { updateVideo, updateVideoDevice, updateSerial } from './lib/stores/connection';

  let showAVSettings = false;
  let showSerialSettings = false;

  // --- Type Text dialog ---
  let showTextInput = false;
  let textToSend = '';
  let textInput: HTMLTextAreaElement;
  let sending = false;

  function openTextInput() {
    showTextInput = true;
    textToSend = '';
    setTimeout(() => textInput?.focus(), 50);
  }

  function closeTextInput() {
    showTextInput = false;
    textToSend = '';
  }

  async function sendText(paste: boolean) {
    if (!textToSend || sending) return;
    sending = true;
    try {
      await SendText(textToSend, paste);
      textToSend = '';
      showTextInput = false;
    } catch (e) {
      console.error('Failed to send text:', e);
    }
    sending = false;
  }

  function onTextKeydown(e: KeyboardEvent) {
    if (e.code === 'Enter' && (e.metaKey || e.ctrlKey) && e.shiftKey) {
      e.preventDefault();
      sendText(true);
      return;
    }
    if (e.code === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      sendText(false);
      return;
    }
    if (e.code === 'Escape') {
      closeTextInput();
    }
    e.stopPropagation();
  }

  // --- Get Clipboard ---
  let clipboardStatus = '';

  async function getRemoteClipboard() {
    clipboardStatus = 'Fetching...';
    try {
      const text = await GetRemoteClipboard();
      if (text) {
        await WriteRemoteClipToHost(text);
        clipboardStatus = 'Copied!';
      } else {
        clipboardStatus = 'Empty';
      }
    } catch (e) {
      clipboardStatus = `Error: ${e}`;
    }
    setTimeout(() => { clipboardStatus = ''; }, 2000);
  }

  // --- RPi Diagnostics dialog ---
  let showDiag = false;
  let diagText = '';
  let diagLoading = false;

  async function runDiagnostics() {
    diagLoading = true;
    diagText = '';
    showDiag = true;
    try {
      diagText = await GetRemoteDiag();
    } catch (e) {
      diagText = `Error: ${e}`;
    }
    diagLoading = false;
  }

  async function runClipboardTest() {
    diagLoading = true;
    diagText = '';
    showDiag = true;
    try {
      diagText = await TestClipboardPipeline();
    } catch (e) {
      diagText = `Error: ${e}`;
    }
    diagLoading = false;
  }

  function closeDiag() {
    showDiag = false;
    diagText = '';
  }

  function onDiagKeydown(e: KeyboardEvent) {
    if (e.code === 'Escape') closeDiag();
    e.stopPropagation();
  }

  // --- Video Diagnostics dialog ---
  let showVideoDiag = false;
  let videoDiagText = '';
  let videoDiagLoading = false;

  async function runVideoDiag() {
    videoDiagLoading = true;
    videoDiagText = '';
    showVideoDiag = true;
    try {
      videoDiagText = await GetVideoDiag();
    } catch (e) {
      videoDiagText = `Error: ${e}`;
    }
    videoDiagLoading = false;
  }

  function closeVideoDiag() {
    showVideoDiag = false;
    videoDiagText = '';
  }

  function onVideoDiagKeydown(e: KeyboardEvent) {
    if (e.code === 'Escape') closeVideoDiag();
    e.stopPropagation();
  }

  // --- Send Key (menu) ---
  const SEND_KEY_MAP: Record<string, string[]> = {
    'escape':       ['Escape'],
    'ctrl-alt-del': ['ControlLeft', 'AltLeft', 'Delete'],
    'alt-tab':      ['AltLeft', 'Tab'],
    'alt-f4':       ['AltLeft', 'F4'],
    'printscreen':  ['PrintScreen'],
    'insert':       ['Insert'],
    'scrolllock':   ['ScrollLock'],
    'pause':        ['Pause'],
  };

  async function sendKeyCombo(codes: string[]) {
    for (const code of codes) await SendKeyEvent(code, true);
    for (const code of [...codes].reverse()) await SendKeyEvent(code, false);
  }

  function handleSendKey(keyId: string) {
    const codes = SEND_KEY_MAP[keyId];
    if (codes) sendKeyCombo(codes);
  }

  // --- Menu event listeners ---
  const cancelFns: (() => void)[] = [];

  onMount(async () => {
    cancelFns.push(EventsOn('menu:typeText', openTextInput));
    cancelFns.push(EventsOn('menu:getClipboard', getRemoteClipboard));
    cancelFns.push(EventsOn('menu:testClipboard', runClipboardTest));
    cancelFns.push(EventsOn('menu:videoDiag', runVideoDiag));
    cancelFns.push(EventsOn('menu:rpiDiag', runDiagnostics));
    cancelFns.push(EventsOn('menu:sendKey', handleSendKey));

    const cfg = await GetConfig();

    // Video: match saved device against available devices, auto-start if found
    const devices = await ListVideoDevices() || [];
    const match = devices.find(d =>
      (cfg.device_path && d.path === cfg.device_path) ||
      (!cfg.device_path && d.index === (cfg.device_index || 0))
    );
    if (match) {
      try {
        await StartVideo();
        updateVideo(true);
        updateVideoDevice(match.name);
      } catch (e) { console.error('Auto-start video:', e); }
    }

    // Serial: match saved port against available ports, auto-connect if found
    if (cfg.serial_port) {
      const ports = await ListSerialPorts() || [];
      if (ports.includes(cfg.serial_port)) {
        try {
          await ConnectSerial(cfg.serial_port);
          updateSerial(cfg.serial_port);
        } catch (e) { console.error('Auto-connect serial:', e); }
      }
    }
  });

  onDestroy(() => {
    cancelFns.forEach(fn => fn());
  });
</script>

<div class="app-layout">
  <div class="main-area">
    <VideoDisplay />
  </div>
  <StatusBar on:av-settings={() => showAVSettings = true} on:serial-settings={() => showSerialSettings = true} />
</div>

{#if showAVSettings}
  <AVSettingsModal on:close={() => showAVSettings = false} />
{/if}

{#if showSerialSettings}
  <SerialSettingsModal on:close={() => showSerialSettings = false} />
{/if}

{#if showTextInput}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="overlay" on:click|self={closeTextInput} role="presentation">
    <div class="dialog">
      <div class="dialog-header">
        <span>Type Text (IME OK)</span>
        <button class="close-btn" on:click={closeTextInput}>&times;</button>
      </div>
      <textarea
        bind:this={textInput}
        bind:value={textToSend}
        on:keydown={onTextKeydown}
        class="text-area"
        placeholder="ここに日本語などを入力..."
        rows="3"
      ></textarea>
      <div class="dialog-footer">
        <span class="hint">Cmd+Enter: Type / Cmd+Shift+Enter: Paste</span>
        <div class="dialog-actions">
          <button class="btn btn-primary" on:click={() => sendText(false)} disabled={!textToSend || sending} title="wtype/xdotool (for terminals)">
            Type
          </button>
          <button class="btn btn-secondary" on:click={() => sendText(true)} disabled={!textToSend || sending} title="wl-copy + Ctrl+V (for browsers)">
            Paste
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

{#if clipboardStatus}
  <div class="clipboard-toast">{clipboardStatus}</div>
{/if}

{#if showDiag}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="overlay" on:click|self={closeDiag} on:keydown={onDiagKeydown} role="presentation">
    <div class="dialog diag-dialog">
      <div class="dialog-header">
        <span>RPi Diagnostics</span>
        <button class="close-btn" on:click={closeDiag}>&times;</button>
      </div>
      <pre class="diag-output">{diagLoading ? 'Fetching diagnostics...' : diagText}</pre>
      <div class="dialog-footer">
        <span class="hint">Esc to close</span>
        <button class="btn" on:click={runDiagnostics} disabled={diagLoading}>Refresh</button>
      </div>
    </div>
  </div>
{/if}

{#if showVideoDiag}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="overlay" on:click|self={closeVideoDiag} on:keydown={onVideoDiagKeydown} role="presentation">
    <div class="dialog diag-dialog">
      <div class="dialog-header">
        <span>Video Pipeline Diagnostics</span>
        <button class="close-btn" on:click={closeVideoDiag}>&times;</button>
      </div>
      <pre class="diag-output">{videoDiagLoading ? 'Fetching video diagnostics...' : videoDiagText}</pre>
      <div class="dialog-footer">
        <span class="hint">Esc to close</span>
        <button class="btn" on:click={runVideoDiag} disabled={videoDiagLoading}>Refresh</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .app-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
    overflow: hidden;
  }

  .main-area {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  /* --- Dialog styles (migrated from Toolbar) --- */

  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
  }

  .dialog {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    width: 400px;
    max-width: 90vw;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
  }

  .dialog-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    border-bottom: 1px solid #334155;
    color: #e2e8f0;
    font-size: 0.9em;
  }

  .close-btn {
    background: none;
    border: none;
    color: #94a3b8;
    font-size: 1.4em;
    cursor: pointer;
    padding: 0 4px;
  }

  .close-btn:hover {
    color: #e2e8f0;
  }

  .text-area {
    width: 100%;
    box-sizing: border-box;
    padding: 10px 14px;
    background: #0f172a;
    border: none;
    color: #e2e8f0;
    font-size: 0.95em;
    resize: vertical;
    outline: none;
    font-family: inherit;
  }

  .dialog-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 14px;
    border-top: 1px solid #334155;
  }

  .hint {
    font-size: 0.75em;
    color: #64748b;
  }

  .btn {
    padding: 4px 12px;
    border: 1px solid #475569;
    border-radius: 4px;
    background: #334155;
    color: #e2e8f0;
    font-size: 0.8em;
    cursor: pointer;
  }

  .btn:hover {
    background: #475569;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: #2563eb;
    border-color: #3b82f6;
  }

  .btn-primary:hover {
    background: #3b82f6;
  }

  .btn-secondary {
    background: #475569;
    border-color: #64748b;
  }

  .btn-secondary:hover {
    background: #64748b;
  }

  .dialog-actions {
    display: flex;
    gap: 6px;
  }

  .diag-dialog {
    width: 600px;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
  }

  .diag-output {
    flex: 1;
    overflow: auto;
    padding: 10px 14px;
    margin: 0;
    background: #0f172a;
    color: #a5f3fc;
    font-size: 0.8em;
    font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 60vh;
    border: none;
  }

  .clipboard-toast {
    position: fixed;
    top: 16px;
    left: 50%;
    transform: translateX(-50%);
    background: #334155;
    color: #e2e8f0;
    padding: 6px 16px;
    border-radius: 6px;
    font-size: 0.85em;
    z-index: 300;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.4);
  }
</style>
