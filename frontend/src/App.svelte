<script lang="ts">
  import { onMount } from 'svelte';
  import VideoDisplay from './lib/components/VideoDisplay.svelte';
  import Toolbar from './lib/components/Toolbar.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';
  import { GetConfig, ListVideoDevices, StartVideo, ListSerialPorts, ConnectSerial } from '../wailsjs/go/main/App';
  import { updateVideo, updateVideoDevice, updateSerial } from './lib/stores/connection';

  let showSettings = false;

  onMount(async () => {
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
</script>

<div class="app-layout">
  <Toolbar on:settings={() => showSettings = true} />
  <div class="main-area">
    <VideoDisplay />
  </div>
  <StatusBar />
</div>

{#if showSettings}
  <SettingsModal on:close={() => showSettings = false} />
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
    overflow: hidden;
  }
</style>
