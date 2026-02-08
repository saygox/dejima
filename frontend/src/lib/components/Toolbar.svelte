<script lang="ts">
  import { StartVideo, StopVideo, GetVideoStatus } from '../../../wailsjs/go/main/App';
  import { connection, updateVideo } from '../stores/connection';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let videoRunning = false;

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
</script>

<div class="toolbar">
  <div class="toolbar-left">
    <span class="title">KVM-Like</span>
  </div>
  <div class="toolbar-center">
    <button class="btn" on:click={toggleVideo}>
      {videoRunning ? 'Stop Video' : 'Start Video'}
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
</style>
