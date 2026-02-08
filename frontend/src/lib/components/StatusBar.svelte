<script lang="ts">
  import { onDestroy } from 'svelte';
  import { connection } from '../stores/connection';
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
</style>
