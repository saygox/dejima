<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { startKeyboardCapture, stopKeyboardCapture, releaseAllKeys } from '../input/keyboard';
  import { startMouseCapture, stopMouseCapture, enterCapture, exitCapture } from '../input/mouse';
  import { connection } from '../stores/connection';
  import { GetStreamURL, SendText } from '../../../wailsjs/go/main/App';

  let videoContainer: HTMLElement;
  let imeInput: HTMLTextAreaElement;
  let captured = false;
  let streamURL = '';
  let composing = false;

  function onContainerClick(e: MouseEvent) {
    if (!captured) {
      captured = true;
      enterCapture(e.clientX, e.clientY);
    }
    // Focus the hidden textarea so it receives IME input
    imeInput?.focus();
  }

  function onKeyDown(e: KeyboardEvent) {
    if (e.code === 'Escape' && captured) {
      releaseCaptureState();
    }
  }

  /** Common cleanup when capture is released (Esc, click-outside, pointer lock exit) */
  function releaseCaptureState() {
    if (!captured) return;
    captured = false;
    exitCapture();
    releaseAllKeys();
  }

  // --- IME handling on the hidden textarea ---

  function onCompositionStart() {
    composing = true;
  }

  function onCompositionEnd(e: CompositionEvent) {
    composing = false;
    const text = e.data;
    if (text) {
      SendText(text).catch(console.error);
    }
    // Clear the textarea so it doesn't accumulate
    if (imeInput) imeInput.value = '';
  }

  function onImeInput() {
    // For non-IME input that ends up in the textarea (e.g. on some platforms),
    // clear it after a tick if we're not composing
    if (!composing) {
      setTimeout(() => {
        if (imeInput && !composing) {
          imeInput.value = '';
        }
      }, 0);
    }
  }

  // Keyboard events on the hidden textarea should bubble to keyboard.ts
  // via the parent container, but we need to handle Esc explicitly
  function onImeKeydown(e: KeyboardEvent) {
    if (e.isComposing) return; // let IME handle it
    // Re-dispatch to the parent so keyboard.ts picks it up
    // preventDefault + stopPropagation are handled by keyboard.ts
  }

  onMount(async () => {
    streamURL = await GetStreamURL();

    // keyboard.ts listens on the container; events from textarea bubble up
    startKeyboardCapture(videoContainer);
    startMouseCapture(videoContainer, () => {
      // Called by mouse.ts when capture exits (click outside, pointer lock exit)
      releaseCaptureState();
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
  class:captured
  bind:this={videoContainer}
  tabindex="0"
  on:click={onContainerClick}
  on:keydown={onKeyDown}
  on:contextmenu|preventDefault
  role="application"
>
  <!-- Hidden textarea for receiving IME composition input -->
  <textarea
    bind:this={imeInput}
    class="ime-input"
    on:compositionstart={onCompositionStart}
    on:compositionend={onCompositionEnd}
    on:input={onImeInput}
    on:keydown={onImeKeydown}
    autocomplete="off"
    autocorrect="off"
    autocapitalize="off"
    spellcheck="false"
  ></textarea>

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
  }

  .video-container:focus-within {
    outline: 2px solid #4a9eff;
  }

  .video-container.captured {
    cursor: none;
  }

  .ime-input {
    position: absolute;
    bottom: 40px;
    left: 50%;
    transform: translateX(-50%);
    width: 300px;
    padding: 4px 8px;
    background: rgba(0, 0, 0, 0.7);
    border: 1px solid #475569;
    border-radius: 4px;
    color: #e2e8f0;
    font-size: 0.9em;
    outline: none;
    resize: none;
    text-align: center;
    z-index: 10;
  }

  .ime-input:empty:not(:focus),
  .video-container:not(.captured) .ime-input {
    opacity: 0;
    pointer-events: none;
    width: 1px;
    height: 1px;
    padding: 0;
    border: none;
    overflow: hidden;
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
