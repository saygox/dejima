<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { ListVideoDevices, ListAudioDevices, GetConfig, SetDevice, SetCaptureResolution, SetAudioDevice, SetAudioSampleRate, StartVideo } from '../../../wailsjs/go/main/App';
  import { updateVideo, updateVideoDevice } from '../stores/connection';

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

  let videoDevices: VideoDevice[] = [];
  let audioDevices: AudioDeviceInfo[] = [];
  let selectedDeviceKey = '0:';
  let selectedResolution = '0x0';
  let selectedSampleRate = 48000;
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
    await Promise.all([refreshVideoDevices(), refreshAudioDevices()]);
    try {
      const cfg = await GetConfig();
      const match = videoDevices.find(d =>
        (cfg.device_path && d.path === cfg.device_path) ||
        (!cfg.device_path && d.index === (cfg.device_index || 0))
      );
      if (match) {
        selectedDeviceKey = `${match.index}:${match.path || ''}`;
        updateVideoDevice(match.name);
        if (!cfg.audio_device_id) {
          const audioDev = audioDevices.find(a =>
            a.name === match.name || a.name.includes(match.name) || match.name.includes(a.name)
          );
          if (audioDev) SetAudioDevice(audioDev.id);
        }
      } else if (videoDevices.length > 0) {
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
      selectedSampleRate = cfg.audio_sample_rate || 48000;
    } catch {
      // use default
    }
  });

  async function refreshVideoDevices() {
    try {
      videoDevices = await ListVideoDevices() || [];
      error = '';
    } catch (e) {
      error = `Failed to list video devices: ${e}`;
    }
  }

  async function refreshAudioDevices() {
    try {
      audioDevices = await ListAudioDevices() || [];
      error = '';
    } catch (e) {
      error = `Failed to list audio devices: ${e}`;
    }
  }

  function onDeviceSelect() {
    const [idxStr, ...pathParts] = selectedDeviceKey.split(':');
    SetDevice(parseInt(idxStr) || 0, pathParts.join(':'));
    const dev = videoDevices.find(d => `${d.index}:${d.path || ''}` === selectedDeviceKey);
    updateVideoDevice(dev?.name || '');
    if (dev) {
      const audioDev = audioDevices.find(a =>
        a.name === dev.name || a.name.includes(dev.name) || dev.name.includes(a.name)
      );
      if (audioDev) SetAudioDevice(audioDev.id);
    }
  }

  function onResolutionChange() {
    const [w, h] = selectedResolution.split('x').map(Number);
    SetCaptureResolution(w, h);
  }

  function onSampleRateChange() {
    SetAudioSampleRate(selectedSampleRate);
  }

  let starting = false;

  async function startAV() {
    starting = true;
    try {
      await StartVideo();
      const dev = videoDevices.find(d => `${d.index}:${d.path || ''}` === selectedDeviceKey);
      updateVideo(true);
      updateVideoDevice(dev?.name || '');
      dispatch('close');
    } catch (e) {
      error = `Failed to start: ${e}`;
      starting = false;
    }
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="overlay" on:click|self={close} role="presentation">
  <div class="modal">
    <div class="modal-header">
      <h3>HDMI Capture Device</h3>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
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
      <p class="hint">Resolution must not exceed the HDMI source output.</p>

      <label class="field-label" for="sample-rate">Audio Sample Rate</label>
      <div class="port-row">
        <select id="sample-rate" bind:value={selectedSampleRate} on:change={onSampleRateChange}>
          <option value={44100}>44100 Hz</option>
          <option value={48000}>48000 Hz</option>
        </select>
      </div>
      <p class="hint">Restart required after changing sample rate.</p>

      <div class="action-row">
        <button class="btn btn-primary" on:click={startAV} disabled={starting || videoDevices.length === 0}>
          {starting ? 'Starting...' : 'Start'}
        </button>
      </div>

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
    align-items: flex-end;
    justify-content: flex-start;
    z-index: 100;
    padding: 0 0 32px 12px;
  }

  .modal {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    width: 380px;
    max-width: 90vw;
    box-shadow: 0 4px 24px rgba(0, 0, 0, 0.5);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    border-bottom: 1px solid #334155;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 0.9em;
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
    padding: 12px 14px;
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

  .field-label {
    font-size: 0.8em;
    color: #94a3b8;
    margin-top: 8px;
    margin-bottom: 4px;
    display: block;
  }

  .hint {
    font-size: 0.75em;
    color: #64748b;
    margin-top: 4px;
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

  .action-row {
    display: flex;
    justify-content: flex-end;
    margin-top: 12px;
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
