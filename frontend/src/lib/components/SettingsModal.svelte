<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { ListSerialPorts, ConnectSerial, DisconnectSerial, DetectFT232, GetSerialStatus, GetConfig, SetDevice, SetCaptureResolution, ListVideoDevices, ListAudioDevices, SetAudioDevice } from '../../../wailsjs/go/main/App';
  import { updateSerial, updateVideoDevice } from '../stores/connection';

  const dispatch = createEventDispatcher();

  interface VideoDevice {
    index: number;
    name: string;
    path: string;
  }

  interface AudioDeviceInfo {
    id: string;
    name: string;
  }

  let serialPorts: string[] = [];
  let selectedPort = '';
  let connectedPort = '';
  let videoDevices: VideoDevice[] = [];
  let selectedDeviceKey = '0:';
  let selectedResolution = '0x0';
  let audioDevices: AudioDeviceInfo[] = [];
  let error = '';

  const resolutions = [
    { label: 'Auto (negotiate with source)', value: '0x0' },
    { label: '640x480', value: '640x480' },
    { label: '720x480', value: '720x480' },
    { label: '800x600', value: '800x600' },
    { label: '1024x768', value: '1024x768' },
    { label: '1280x720 (720p)', value: '1280x720' },
    { label: '1280x1024', value: '1280x1024' },
    { label: '1920x1080 (1080p)', value: '1920x1080' },
  ];

  onMount(async () => {
    await Promise.all([refreshPorts(), refreshVideoDevices(), refreshAudioDevices()]);
    connectedPort = await GetSerialStatus();
    if (connectedPort) {
      selectedPort = connectedPort;
      updateSerial(connectedPort);
    }
    try {
      const cfg = await GetConfig();

      // Match config to an actual device in the list.
      // On Windows, device_path is the reliable identifier; on macOS/Linux it's device_index.
      const match = videoDevices.find(d =>
        (cfg.device_path && d.path === cfg.device_path) ||
        (!cfg.device_path && d.index === (cfg.device_index || 0))
      );
      if (match) {
        selectedDeviceKey = `${match.index}:${match.path || ''}`;
        updateVideoDevice(match.name);
        // Auto-set matching audio device if not already configured
        if (!cfg.audio_device_id) {
          const audioDev = audioDevices.find(a =>
            a.name === match.name || a.name.includes(match.name) || match.name.includes(a.name)
          );
          if (audioDev) SetAudioDevice(audioDev.id);
        }
      } else if (videoDevices.length > 0) {
        // No match — auto-select (and save) the first device
        const first = videoDevices[0];
        selectedDeviceKey = `${first.index}:${first.path || ''}`;
        SetDevice(first.index, first.path || '');
        updateVideoDevice(first.name);
        const audioDev = audioDevices.find(a =>
          a.name === first.name || a.name.includes(first.name) || first.name.includes(a.name)
        );
        if (audioDev) SetAudioDevice(audioDev.id);
      }

      const w = cfg.capture_width || 0;
      const h = cfg.capture_height || 0;
      selectedResolution = (w && h) ? `${w}x${h}` : '0x0';
    } catch (e) {
      // use default
    }
  });

  async function refreshPorts() {
    try {
      serialPorts = await ListSerialPorts();
      error = '';
    } catch (e) {
      error = `Failed to list ports: ${e}`;
    }
  }

  async function autoDetect() {
    try {
      const port = await DetectFT232();
      selectedPort = port;
      error = '';
    } catch (e) {
      error = `Auto-detect failed: ${e}`;
    }
  }

  async function connect() {
    if (!selectedPort) return;
    try {
      await ConnectSerial(selectedPort);
      connectedPort = selectedPort;
      updateSerial(connectedPort);
      error = '';
    } catch (e) {
      error = `Connection failed: ${e}`;
    }
  }

  async function disconnect() {
    try {
      await DisconnectSerial();
      connectedPort = '';
      updateSerial('');
    } catch (e) {
      error = `Serial切断に失敗しました: ${e}`;
    }
  }

  async function refreshVideoDevices() {
    try {
      videoDevices = await ListVideoDevices() || [];
      error = '';
    } catch (e) {
      error = `Failed to list video devices: ${e}`;
    }
  }

  function onDeviceSelect() {
    const [idxStr, ...pathParts] = selectedDeviceKey.split(':');
    SetDevice(parseInt(idxStr) || 0, pathParts.join(':'));
    const dev = videoDevices.find(d => `${d.index}:${d.path || ''}` === selectedDeviceKey);
    updateVideoDevice(dev?.name || '');
    // Auto-set matching audio device by name
    if (dev) {
      const audioDev = audioDevices.find(a =>
        a.name === dev.name || a.name.includes(dev.name) || dev.name.includes(a.name)
      );
      if (audioDev) {
        SetAudioDevice(audioDev.id);
      }
    }
  }

  function onResolutionChange() {
    const [w, h] = selectedResolution.split('x').map(Number);
    SetCaptureResolution(w, h);
  }

  async function refreshAudioDevices() {
    try {
      audioDevices = await ListAudioDevices() || [];
      error = '';
    } catch (e) {
      error = `Failed to list audio devices: ${e}`;
    }
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="overlay" on:click|self={close} role="presentation">
  <div class="modal">
    <div class="modal-header">
      <h3>Settings</h3>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
      <section>
        <h4>HDMI Capture Device</h4>
        <div class="port-row">
          <select bind:value={selectedDeviceKey} on:change={onDeviceSelect}>
            {#if videoDevices.length === 0}
              <option value="0:">No devices found</option>
            {/if}
            {#each videoDevices as dev}
              <option value="{dev.index}:{dev.path || ''}">{dev.name}</option>
            {/each}
          </select>
          <button class="btn" on:click={() => { refreshVideoDevices(); refreshAudioDevices(); }}>Refresh</button>
        </div>
        <div class="port-row">
          <select bind:value={selectedResolution} on:change={onResolutionChange}>
            {#each resolutions as res}
              <option value={res.value}>{res.label}</option>
            {/each}
          </select>
        </div>
        <p class="hint">Resolution must not exceed the HDMI source output. Restart video after changing.</p>
      </section>

      <section>
        <h4>Serial Connection</h4>
        <div class="port-row">
          <select bind:value={selectedPort}>
            <option value="">Select port...</option>
            {#each serialPorts as port}
              <option value={port}>{port}</option>
            {/each}
          </select>
          <button class="btn" on:click={refreshPorts}>Refresh</button>
          <button class="btn" on:click={autoDetect}>Auto-detect</button>
        </div>
        <div class="port-row">
          {#if connectedPort}
            <span class="connected-label">Connected: {connectedPort}</span>
            <button class="btn btn-danger" on:click={disconnect}>Disconnect</button>
          {:else}
            <button class="btn btn-primary" on:click={connect} disabled={!selectedPort}>Connect</button>
          {/if}
        </div>
      </section>

      {#if error}
        <div class="error">{error}</div>
      {/if}
    </div>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .modal {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    width: 420px;
    max-width: 90vw;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid #334155;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1em;
    color: #e2e8f0;
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

  .modal-body {
    padding: 16px;
  }

  section {
    margin-bottom: 16px;
  }

  h4 {
    margin: 0 0 8px;
    font-size: 0.9em;
    color: #94a3b8;
  }

  select {
    flex: 1;
    padding: 4px 8px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 4px;
    color: #e2e8f0;
    font-size: 0.85em;
  }

  .port-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .hint {
    font-size: 0.75em;
    color: #64748b;
    margin-top: 4px;
  }

  .connected-label {
    color: #22c55e;
    font-size: 0.85em;
  }

  .btn {
    padding: 4px 12px;
    border: 1px solid #475569;
    border-radius: 4px;
    background: #334155;
    color: #e2e8f0;
    font-size: 0.8em;
    cursor: pointer;
    white-space: nowrap;
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

  .btn-danger {
    background: #dc2626;
    border-color: #ef4444;
  }

  .btn-danger:hover {
    background: #ef4444;
  }

  .error {
    color: #ef4444;
    font-size: 0.85em;
    margin-top: 8px;
    padding: 8px;
    background: rgba(239, 68, 68, 0.1);
    border-radius: 4px;
  }
</style>
