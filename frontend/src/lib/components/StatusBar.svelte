<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  import { connection, updateVideo, updateSerial } from '../stores/connection';
  import { pasteMode } from '../stores/imeMode';
  import { GetVideoFrameCount, GetVideoStatus, StartVideo, StopVideo, GetConfig, ConnectSerial, DisconnectSerial, DetectSerialPort, GetAudioVolume, GetAudioMuted, SetAudioVolume, SetAudioMuted } from '../../../wailsjs/go/main/App';

  const dispatch = createEventDispatcher();

  let avLongPress: ReturnType<typeof setTimeout> | null = null;
  let serialLongPress: ReturnType<typeof setTimeout> | null = null;
  let lastAVClickTime = 0;
  let lastSerialClickTime = 0;

  function onAVPointerDown() {
    console.log('[StatusBar] onAVPointerDown');
    // Double-click detection (before streaming check)
    const now = Date.now();
    if (now - lastAVClickTime < 400) {
      lastAVClickTime = 0;
      if (avLongPress) { clearTimeout(avLongPress); avLongPress = null; }
      onAVDblClick();
      return;
    }
    lastAVClickTime = now;

    // Long-press: OFF → open settings, ON → stop
    avLongPress = setTimeout(() => {
      avLongPress = null;
      console.log('[StatusBar] long-press fired, streaming=', $connection.videoStreaming);
      if ($connection.videoStreaming) {
        onAVDblClick();
      } else {
        dispatch('av-settings');
      }
    }, 500);
  }

  function onAVPointerUp() {
    if (avLongPress) {
      clearTimeout(avLongPress);
      avLongPress = null;
    }
  }

  function onSerialPointerDown() {
    // Double-click detection (before connected check)
    const now = Date.now();
    if (now - lastSerialClickTime < 400) {
      lastSerialClickTime = 0;
      if (serialLongPress) { clearTimeout(serialLongPress); serialLongPress = null; }
      onSerialDblClick();
      return;
    }
    lastSerialClickTime = now;

    // Long-press: disconnected → open settings, connected → disconnect
    serialLongPress = setTimeout(() => {
      serialLongPress = null;
      if ($connection.serialConnected) {
        onSerialDblClick();
      } else {
        dispatch('serial-settings');
      }
    }, 500);
  }

  function onSerialPointerUp() {
    if (serialLongPress) {
      clearTimeout(serialLongPress);
      serialLongPress = null;
    }
  }

  let frameCount = 0;
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let streamStartTime = 0;
  let statusError = '';
  let errorTimer: ReturnType<typeof setTimeout> | null = null;

  let audioVolume = 80;
  let audioMuted = false;

  function showError(msg: string) {
    statusError = msg;
    if (errorTimer) clearTimeout(errorTimer);
    errorTimer = setTimeout(() => { statusError = ''; }, 3000);
  }

  onMount(async () => {
    try {
      const cfg = await GetConfig();
      audioVolume = cfg.audio_volume ?? 80;
      audioMuted = cfg.audio_muted ?? false;
    } catch {
      // use defaults
    }
  });

  function onVolumeChange() {
    SetAudioVolume(audioVolume);
  }

  function toggleMute() {
    audioMuted = !audioMuted;
    SetAudioMuted(audioMuted);
  }

  async function onAVDblClick() {
    console.log('[StatusBar] onAVDblClick, streaming=', $connection.videoStreaming);
    if ($connection.videoStreaming) {
      try {
        await StopVideo();
      } catch (e) {
        console.error('[StatusBar] StopVideo failed:', e);
      } finally {
        updateVideo(false);
      }
    } else {
      try {
        await StartVideo();
        updateVideo(true);
      } catch {
        dispatch('av-settings');
      }
    }
  }

  async function onSerialDblClick() {
    if ($connection.serialConnected) {
      try {
        await DisconnectSerial();
        updateSerial('');
      } catch (e) {
        console.error('[StatusBar] DisconnectSerial failed:', e);
        showError('Serial切断に失敗しました。再試行してください');
      }
    } else {
      try {
        const cfg = await GetConfig();
        let port = cfg.serial_port;
        if (!port) {
          port = await DetectSerialPort();
        }
        await ConnectSerial(port);
        updateSerial(port);
      } catch {
        dispatch('serial-settings');
      }
    }
  }

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
      // Sync frontend state with backend reality
      try {
        const backendRunning = await GetVideoStatus();
        if ($connection.videoStreaming && !backendRunning) {
          console.log('[StatusBar] backend pipeline stopped — syncing UI');
          updateVideo(false);
        }
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

  onDestroy(() => {
    stopPolling();
    if (errorTimer) clearTimeout(errorTimer);
    if (avLongPress) clearTimeout(avLongPress);
    if (serialLongPress) clearTimeout(serialLongPress);
  });

  $: noFramesWarning = $connection.videoStreaming && frameCount === 0 && streamStartTime > 0 && (Date.now() - streamStartTime) > 5000;

  function videoLabel(streaming: boolean, warn: boolean, deviceName: string): string {
    if (!streaming) return 'AV: Off';
    if (warn) return 'AV: No signal';
    if (deviceName) return `AV: ${deviceName}`;
    return 'AV: Streaming';
  }
</script>

<div class="status-bar">
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="status-item clickable" on:pointerdown={onAVPointerDown} on:pointerup={onAVPointerUp} on:pointerleave={onAVPointerUp} on:contextmenu|preventDefault title="{$connection.videoStreaming ? 'ダブルクリック/長押しで停止' : 'ダブルクリック/長押しで設定'}" role="button" tabindex="0">
    <span class="dot" class:connected={$connection.videoStreaming} class:warning={noFramesWarning}></span>
    {videoLabel($connection.videoStreaming, noFramesWarning, $connection.videoDeviceName)}
  </div>
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div class="status-item clickable" on:pointerdown={onSerialPointerDown} on:pointerup={onSerialPointerUp} on:pointerleave={onSerialPointerUp} on:contextmenu|preventDefault title="{$connection.serialConnected ? 'ダブルクリック/長押しで切断' : 'ダブルクリック/長押しで設定'}" role="button" tabindex="0">
    <span class="dot" class:connected={$connection.serialConnected}></span>
    Serial: {$connection.serialConnected ? $connection.serialPort : 'Disconnected'}
  </div>
  {#if statusError}
    <span class="status-error">{statusError}</span>
  {/if}

  <div class="spacer"></div>

  <!-- Volume control -->
  <div class="volume-control">
    <button class="vol-btn" class:muted={audioMuted} on:click={toggleMute} title={audioMuted ? 'Unmute' : 'Mute'}>
      {audioMuted ? 'MUTE' : 'VOL'}
    </button>
    <input
      type="range"
      class="vol-slider"
      min="0"
      max="100"
      bind:value={audioVolume}
      on:input={onVolumeChange}
      title="Volume: {audioVolume}%"
    />
  </div>

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
    min-width: 200px;
  }

  .status-item.clickable {
    cursor: pointer;
    padding: 1px 4px;
    border-radius: 3px;
    transition: background 0.15s;
  }

  .status-item.clickable:hover {
    background: #1e293b;
  }

  .status-error {
    font-size: 0.75em;
    color: #ef4444;
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

  /* Volume control */
  .volume-control {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .vol-btn {
    background: none;
    border: 1px solid #334155;
    border-radius: 3px;
    color: #94a3b8;
    font-size: 0.8em;
    font-weight: 600;
    letter-spacing: 0.05em;
    padding: 1px 5px;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s;
    line-height: 1.2;
  }

  .vol-btn:hover {
    color: #e2e8f0;
    border-color: #475569;
  }

  .vol-btn.muted {
    color: #ef4444;
    border-color: #7f1d1d;
  }

  .vol-slider {
    -webkit-appearance: none;
    appearance: none;
    width: 64px;
    height: 4px;
    background: #334155;
    border-radius: 2px;
    outline: none;
    cursor: pointer;
  }

  .vol-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 10px;
    height: 10px;
    background: #94a3b8;
    border-radius: 50%;
    cursor: pointer;
    transition: background 0.15s;
  }

  .vol-slider::-webkit-slider-thumb:hover {
    background: #e2e8f0;
  }

  .vol-slider::-moz-range-thumb {
    width: 10px;
    height: 10px;
    background: #94a3b8;
    border: none;
    border-radius: 50%;
    cursor: pointer;
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
