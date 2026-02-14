<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { startKeyboardCapture, stopKeyboardCapture, releaseAllKeys } from '../input/keyboard';
  import { startMouseCapture, stopMouseCapture, enterCapture, exitCapture } from '../input/mouse';
  import { connection, updateVideo } from '../stores/connection';
  import { GetStreamURL, SendText, StartVideo, StopVideo } from '../../../wailsjs/go/main/App';
  import { WindowSetTitle } from '../../../wailsjs/runtime';
  import { pasteMode } from '../stores/imeMode';

  let videoContainer: HTMLElement;
  let imeInput: HTMLTextAreaElement;
  const TITLE_DEFAULT = 'Dejima KVM';
  const TITLE_CAPTURED = 'Dejima KVM — Input captured (Esc to release)';

  let captured = false;

  $: WindowSetTitle(captured ? TITLE_CAPTURED : TITLE_DEFAULT);
  let streamURL = '';
  let composing = false;
  let justComposed = false;
  let lastContainerClickTime = 0;
  let singleClickTimer: ReturnType<typeof setTimeout> | null = null;

  function onContainerPointerDown(e: PointerEvent) {
    if (captured) return; // mouse.ts handles captured clicks

    const now = Date.now();
    if (now - lastContainerClickTime < 400) {
      // Double-click → toggle AV
      lastContainerClickTime = 0;
      if (singleClickTimer) {
        clearTimeout(singleClickTimer);
        singleClickTimer = null;
      }
      toggleAV();
    } else {
      // Possible single-click → delay 400ms to distinguish from double-click
      lastContainerClickTime = now;
      const clientX = e.clientX;
      const clientY = e.clientY;
      singleClickTimer = setTimeout(() => {
        singleClickTimer = null;
        captured = true;
        enterCapture(clientX, clientY);
        imeInput?.focus();
      }, 400);
    }
  }

  async function toggleAV() {
    console.log('[VideoDisplay] toggleAV, streaming=', $connection.videoStreaming);
    if ($connection.videoStreaming) {
      try {
        await StopVideo();
      } catch (e) {
        console.error('[VideoDisplay] StopVideo failed:', e);
      } finally {
        updateVideo(false);
      }
    } else {
      try {
        await StartVideo();
        updateVideo(true);
      } catch {
        // ignore — user can use status bar settings
      }
    }
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
    justComposed = true;
    const text = e.data;
    if (text) {
      SendText(text, $pasteMode).catch(console.error);
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
    if (e.isComposing || e.keyCode === 229) {
      e.stopPropagation();
      return; // let IME handle it
    }
    if (justComposed && e.code === 'Enter') {
      justComposed = false;
      e.preventDefault();
      e.stopPropagation();
      return; // swallow the Enter that confirmed the composition
    }
    justComposed = false;
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
    if (singleClickTimer) clearTimeout(singleClickTimer);
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
  on:pointerdown={onContainerPointerDown}
  on:keydown={onKeyDown}
  on:contextmenu|preventDefault
  role="application"
>
  <!-- Hidden textarea for receiving IME composition input -->
  <textarea
    bind:this={imeInput}
    class="ime-input"
    class:composing
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

  /* Default: invisible but focusable (for compositionstart detection) */
  .ime-input {
    position: absolute;
    bottom: 40px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 10;
    outline: none;
    resize: none;
    opacity: 0;
    pointer-events: none;
    width: 1px;
    height: 1px;
    padding: 0;
    border: none;
    overflow: hidden;
  }

  /* Visible only during IME composition */
  .ime-input.composing {
    opacity: 1;
    pointer-events: auto;
    width: 300px;
    height: calc(1.5em + 8px);
    padding: 4px 8px;
    background: rgba(0, 0, 0, 0.7);
    border: 1px solid #475569;
    border-radius: 4px;
    color: #e2e8f0;
    font-size: 1rem;
    line-height: 1.5;
    text-align: center;
  }

  /* Always hidden when not captured */
  .video-container:not(.captured) .ime-input {
    opacity: 0;
    pointer-events: none;
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

</style>
