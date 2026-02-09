<script lang="ts">
  import { onDestroy } from 'svelte';
  import { connection } from '../stores/connection';
  import { pasteMode } from '../stores/imeMode';
  import { GetVideoFrameCount } from '../../../wailsjs/go/main/App';

  let frameCount = 0;
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let streamStartTime = 0;

  function startPolling() {
    stopPolling();
    frameCount = 0;
    streamStartTime = Date.now();
    pollTimer = setInterval(async () => {
      try {
        frameCount = await GetVideoFrameCount();
      } catch {
        // ignore
      }
    }, 2000);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
    frameCount = 0;
    streamStartTime = 0;
  }

  // React to streaming state changes
  $: if ($connection.videoStreaming) {
    startPolling();
  } else {
    stopPolling();
  }

  onDestroy(() => stopPolling());

  $: noFramesWarning = $connection.videoStreaming && frameCount === 0 && streamStartTime > 0 && (Date.now() - streamStartTime) > 5000;

  function videoLabel(streaming: boolean, fc: number, warn: boolean): string {
    if (!streaming) return 'Video: Off';
    if (warn) return `Video: Streaming (no frames!)`;
    if (fc > 0) return `Video: Streaming (${fc} frames)`;
    return 'Video: Streaming';
  }
</script>

<div class="status-bar">
  <div class="status-item" title={$connection.videoDeviceName || 'No device selected'}>
    <span class="dot" class:connected={$connection.videoStreaming} class:warning={noFramesWarning}></span>
    {videoLabel($connection.videoStreaming, frameCount, noFramesWarning)}
  </div>
  <div class="status-item">
    <span class="dot" class:connected={$connection.serialConnected}></span>
    Serial: {$connection.serialConnected ? $connection.serialPort : 'Disconnected'}
  </div>

  <div class="spacer"></div>

  <!-- DIP switch: type / paste mode -->
  <div class="dip-switch">
    <span class="dip-label" class:active={!$pasteMode}>TYPE</span>
    <button
      class="dip-toggle"
      class:on={$pasteMode}
      on:click={() => { $pasteMode = !$pasteMode; }}
      title={$pasteMode ? 'Paste mode (clipboard + Ctrl+V)' : 'Type mode (wtype/xdotool)'}
    >
      <span class="dip-knob" />
    </button>
    <span class="dip-label" class:active={$pasteMode}>PASTE</span>
  </div>
</div>

<style>
  .status-bar {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 4px 12px;
    background: #0f172a;
    border-top: 1px solid #1e293b;
    font-size: 0.75em;
    color: #94a3b8;
    height: 24px;
  }

  .status-item {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: default;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #ef4444;
  }

  .dot.connected {
    background: #22c55e;
  }

  .dot.warning {
    background: #f59e0b;
  }

  .spacer {
    flex: 1;
  }

  /* DIP switch */
  .dip-switch {
    display: flex;
    align-items: center;
    gap: 5px;
    user-select: none;
  }

  .dip-label {
    font-size: 0.9em;
    font-weight: 600;
    letter-spacing: 0.05em;
    color: #475569;
    transition: color 0.15s;
  }

  .dip-label.active {
    color: #e2e8f0;
  }

  .dip-toggle {
    position: relative;
    width: 24px;
    height: 12px;
    background: #334155;
    border: 1px solid #475569;
    border-radius: 6px;
    padding: 0;
    cursor: pointer;
    transition: background 0.15s;
  }

  .dip-toggle.on {
    background: #1e40af;
    border-color: #3b82f6;
  }

  .dip-knob {
    position: absolute;
    top: 1px;
    left: 1px;
    width: 8px;
    height: 8px;
    background: #94a3b8;
    border-radius: 50%;
    transition: transform 0.15s, background 0.15s;
  }

  .dip-toggle.on .dip-knob {
    transform: translateX(12px);
    background: #60a5fa;
  }
</style>
