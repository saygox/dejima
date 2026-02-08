<script lang="ts">
  import { StartVideo, StopVideo, GetVideoStatus, SendText, GetRemoteClipboard, GetRemoteDiag } from '../../../wailsjs/go/main/App';
  import { connection, updateVideo } from '../stores/connection';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let videoRunning = false;
  let showTextInput = false;
  let textToSend = '';
  let textInput: HTMLTextAreaElement;
  let sending = false;

  async function toggleVideo() {
    if (videoRunning) {
      await StopVideo();
      videoRunning = false;
    } else {
      try {
        await StartVideo();
        videoRunning = true;
      } catch (e) {
        console.error('Failed to start video:', e);
      }
    }
    updateVideo(videoRunning);
  }

  function openSettings() {
    dispatch('settings');
  }

  function toggleFullscreen() {
    if (document.fullscreenElement) {
      document.exitFullscreen();
    } else {
      document.documentElement.requestFullscreen();
    }
  }

  function openTextInput() {
    showTextInput = true;
    textToSend = '';
    // Focus after DOM update
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

  let clipboardStatus = '';

  async function getRemoteClipboard() {
    clipboardStatus = 'Fetching...';
    try {
      const text = await GetRemoteClipboard();
      if (text) {
        await navigator.clipboard.writeText(text);
        clipboardStatus = 'Copied!';
      } else {
        clipboardStatus = 'Empty';
      }
    } catch (e) {
      clipboardStatus = `Error: ${e}`;
    }
    setTimeout(() => { clipboardStatus = ''; }, 2000);
  }

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

  function closeDiag() {
    showDiag = false;
    diagText = '';
  }

  function onDiagKeydown(e: KeyboardEvent) {
    if (e.code === 'Escape') closeDiag();
    e.stopPropagation();
  }

  function onTextKeydown(e: KeyboardEvent) {
    // Cmd/Ctrl+Shift+Enter = Paste mode (browser)
    if (e.code === 'Enter' && (e.metaKey || e.ctrlKey) && e.shiftKey) {
      e.preventDefault();
      sendText(true);
      return;
    }
    // Cmd/Ctrl+Enter = Type mode (terminal)
    if (e.code === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      sendText(false);
      return;
    }
    if (e.code === 'Escape') {
      closeTextInput();
    }
    // Stop propagation so keyboard.ts doesn't capture these
    e.stopPropagation();
  }
</script>

<div class="toolbar">
  <div class="toolbar-left">
    <span class="title">KVM-Like</span>
  </div>
  <div class="toolbar-center">
    <button class="btn" on:click={toggleVideo}>
      {videoRunning ? 'Stop Video' : 'Start Video'}
    </button>
    <button class="btn" on:click={openTextInput} title="Type text (Japanese etc.)">
      Type Text
    </button>
    <button class="btn" on:click={getRemoteClipboard} title="Get clipboard from remote RPi">
      Get Clipboard
    </button>
    {#if clipboardStatus}
      <span class="clipboard-status">{clipboardStatus}</span>
    {/if}
  </div>
  <div class="toolbar-right">
    <button class="btn" on:click={toggleFullscreen} title="Toggle fullscreen">
      Fullscreen
    </button>
    <button class="btn" on:click={runDiagnostics} title="RPi diagnostics">
      Diag
    </button>
    <button class="btn" on:click={openSettings} title="Settings">
      Settings
    </button>
  </div>
</div>

{#if showTextInput}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="text-overlay" on:click|self={closeTextInput} role="presentation">
    <div class="text-dialog">
      <div class="text-header">
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
      <div class="text-footer">
        <span class="hint">Cmd+Enter: Type / Cmd+Shift+Enter: Paste</span>
        <div class="text-actions">
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

{#if showDiag}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="text-overlay" on:click|self={closeDiag} on:keydown={onDiagKeydown} role="presentation">
    <div class="text-dialog diag-dialog">
      <div class="text-header">
        <span>RPi Diagnostics</span>
        <button class="close-btn" on:click={closeDiag}>&times;</button>
      </div>
      <pre class="diag-output">{diagLoading ? 'Fetching diagnostics...' : diagText}</pre>
      <div class="text-footer">
        <span class="hint">Esc to close</span>
        <button class="btn" on:click={runDiagnostics} disabled={diagLoading}>Refresh</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    background: #1e293b;
    border-bottom: 1px solid #334155;
    height: 40px;
    user-select: none;
    --wails-draggable: drag;
  }

  .toolbar-left, .toolbar-center, .toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .title {
    font-weight: 600;
    font-size: 0.9em;
    color: #e2e8f0;
  }

  .btn {
    padding: 4px 12px;
    border: 1px solid #475569;
    border-radius: 4px;
    background: #334155;
    color: #e2e8f0;
    font-size: 0.8em;
    cursor: pointer;
    --wails-draggable: no-drag;
  }

  .btn:hover {
    background: #475569;
  }

  .clipboard-status {
    font-size: 0.75em;
    color: #94a3b8;
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

  .text-actions {
    display: flex;
    gap: 6px;
  }

  .text-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
  }

  .text-dialog {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    width: 400px;
    max-width: 90vw;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
  }

  .text-header {
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

  .text-footer {
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
</style>
