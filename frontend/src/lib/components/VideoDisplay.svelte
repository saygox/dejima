<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { startKeyboardCapture, stopKeyboardCapture } from '../input/keyboard';
  import { startMouseCapture, stopMouseCapture, requestPointerLock, exitPointerLock } from '../input/mouse';
  import { connection } from '../stores/connection';
  import { GetStreamURL } from '../../../wailsjs/go/main/App';

  let videoContainer: HTMLElement;
  let captured = false;
  let streamURL = '';

  function onContainerClick() {
    if (!captured) {
      // Only enter pointer lock on click; exit is via Esc (browser default)
      requestPointerLock(videoContainer);
    }
    // During pointer lock, clicks are handled by mouse.ts as button events
  }

  onMount(async () => {
    streamURL = await GetStreamURL();

    startKeyboardCapture(videoContainer);
    startMouseCapture(videoContainer);

    document.addEventListener('pointerlockchange', () => {
      captured = document.pointerLockElement === videoContainer;
      // Ensure the element keeps focus while captured
      if (captured) {
        videoContainer.focus();
      }
    });
  });

  onDestroy(() => {
    stopKeyboardCapture(videoContainer);
    stopMouseCapture(videoContainer);
  });
</script>

<!-- svelte-ignore a11y-no-noninteractive-tabindex a11y-click-events-have-key-events -->
<div
  class="video-container"
  bind:this={videoContainer}
  tabindex="0"
  on:click={onContainerClick}
  on:contextmenu|preventDefault
  role="application"
>
  {#if $connection.videoStreaming}
    <img src={streamURL} alt="Video stream" class="video-stream" draggable="false" />
  {:else}
    <div class="no-video">
      <p>No video stream</p>
      <p class="hint">Start the video capture to view the remote display</p>
    </div>
  {/if}

  {#if captured}
    <div class="capture-indicator">Input captured (Esc to release)</div>
  {:else}
    <div class="capture-hint">Click to capture input</div>
  {/if}
</div>

<style>
  .video-container {
    position: relative;
    width: 100%;
    height: 100%;
    background: #000;
    outline: none;
    overflow: hidden;
    cursor: pointer;
  }

  .video-container:focus {
    outline: 2px solid #4a9eff;
  }

  .video-stream {
    width: 100%;
    height: 100%;
    object-fit: contain;
    pointer-events: none;
  }

  .no-video {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #888;
  }

  .no-video .hint {
    font-size: 0.85em;
    color: #555;
  }

  .capture-indicator {
    position: absolute;
    top: 8px;
    right: 8px;
    background: rgba(74, 158, 255, 0.8);
    color: white;
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 0.8em;
    pointer-events: none;
  }

  .capture-hint {
    position: absolute;
    bottom: 8px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(0, 0, 0, 0.6);
    color: #aaa;
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 0.8em;
    pointer-events: none;
  }
</style>
