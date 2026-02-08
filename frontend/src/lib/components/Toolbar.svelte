<script lang="ts">
  import { StartVideo, StopVideo, GetVideoStatus, SendText } from '../../../wailsjs/go/main/App';
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

  async function sendText() {
    if (!textToSend || sending) return;
    sending = true;
    try {
      await SendText(textToSend);
      textToSend = '';
      showTextInput = false;
    } catch (e) {
      console.error('Failed to send text:', e);
    }
    sending = false;
  }

  function onTextKeydown(e: KeyboardEvent) {
    // Ctrl/Cmd+Enter to send
    if (e.code === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      sendText();
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
  </div>
  <div class="toolbar-right">
    <button class="btn" on:click={toggleFullscreen} title="Toggle fullscreen">
      Fullscreen
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
        <span class="hint">Cmd+Enter で送信 / Esc でキャンセル</span>
        <button class="btn btn-primary" on:click={sendText} disabled={!textToSend || sending}>
          Send
        </button>
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
</style>
