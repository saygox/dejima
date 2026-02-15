<script lang="ts">
  import { EventsEmit, WindowMinimise, WindowFullscreen, WindowUnfullscreen, Quit } from '../../../wailsjs/runtime/runtime';
  import { isFullscreen } from '../stores/fullscreen';

  let menuOpen = false;
  let sendKeyOpen = false;
  let visible = true;
  let hideTimer: ReturnType<typeof setTimeout> | null = null;

  $: if ($isFullscreen) {
    // When entering fullscreen, hide after a short delay
    scheduleHide();
  } else {
    // When leaving fullscreen, always show
    cancelHide();
    visible = true;
  }

  function scheduleHide() {
    cancelHide();
    hideTimer = setTimeout(() => {
      visible = false;
      hideTimer = null;
    }, 600);
  }

  function cancelHide() {
    if (hideTimer) {
      clearTimeout(hideTimer);
      hideTimer = null;
    }
  }

  function onHotZoneEnter() {
    if (!$isFullscreen) return;
    cancelHide();
    visible = true;
  }

  function onBarLeave() {
    if (!$isFullscreen) return;
    menuOpen = false;
    sendKeyOpen = false;
    scheduleHide();
  }

  function toggleMenu() {
    sendKeyOpen = false;
    menuOpen = !menuOpen;
  }

  function toggleSendKey() {
    sendKeyOpen = !sendKeyOpen;
  }

  function menuAction(event: string, ...args: any[]) {
    EventsEmit(event, ...args);
    menuOpen = false;
    sendKeyOpen = false;
  }

  function doMinimise() {
    WindowMinimise();
  }

  function doToggleFullscreen() {
    if ($isFullscreen) {
      WindowUnfullscreen();
      $isFullscreen = false;
    } else {
      WindowFullscreen();
      $isFullscreen = true;
    }
  }

  function doClose() {
    Quit();
  }

  function onWindowClick(e: MouseEvent) {
    // Close dropdown if clicking outside
    const target = e.target as HTMLElement;
    if (!target.closest('.tools-menu-wrapper')) {
      menuOpen = false;
      sendKeyOpen = false;
    }
  }

  const SEND_KEYS = [
    { label: 'Escape', id: 'escape' },
    { label: 'Ctrl+Alt+Delete', id: 'ctrl-alt-del' },
    { label: 'Alt+Tab', id: 'alt-tab' },
    { label: 'Alt+F4', id: 'alt-f4' },
    { label: 'PrintScreen', id: 'printscreen' },
    { label: 'Insert', id: 'insert' },
    { label: 'ScrollLock', id: 'scrolllock' },
    { label: 'Pause/Break', id: 'pause' },
  ];
</script>

<svelte:window on:click={onWindowClick} />

{#if $isFullscreen}
  <!-- Invisible hot zone at top of screen to trigger title bar reveal -->
  <!-- svelte-ignore a11y-no-static-element-interactions -->
  <div class="hot-zone" on:mouseenter={onHotZoneEnter}></div>
{/if}

<!-- svelte-ignore a11y-no-static-element-interactions -->
<div
  class="title-bar"
  class:hidden={$isFullscreen && !visible}
  class:fullscreen={$isFullscreen}
  on:mouseleave={onBarLeave}
  on:mouseenter={cancelHide}
  style="--wails-draggable: drag"
>
  <!-- Left: Tools menu -->
  <div class="title-section left" style="--wails-draggable: no-drag">
    <div class="tools-menu-wrapper">
      <button class="tools-btn" on:click={toggleMenu}>
        Tools
        <span class="arrow">{menuOpen ? '\u25B4' : '\u25BE'}</span>
      </button>
      {#if menuOpen}
        <div class="dropdown">
          <button class="dropdown-item" on:click={() => menuAction('menu:typeText')}>Type Text...</button>
          <button class="dropdown-item" on:click={() => menuAction('menu:getClipboard')}>Get Clipboard</button>
          <div class="dropdown-separator"></div>
          <button
            class="dropdown-item has-submenu"
            on:click={toggleSendKey}
          >
            Send Key
            <span class="submenu-arrow">{'\u25B8'}</span>
          </button>
          {#if sendKeyOpen}
            <div class="submenu">
              {#each SEND_KEYS as key}
                <button class="dropdown-item" on:click={() => menuAction('menu:sendKey', key.id)}>{key.label}</button>
              {/each}
            </div>
          {/if}
          <div class="dropdown-separator"></div>
          <button class="dropdown-item" on:click={() => menuAction('menu:testClipboard')}>Test Clipboard Pipeline...</button>
          <button class="dropdown-item" on:click={() => menuAction('menu:videoDiag')}>Video Diagnostics...</button>
          <button class="dropdown-item" on:click={() => menuAction('menu:rpiDiag')}>RPi Diagnostics...</button>
        </div>
      {/if}
    </div>
  </div>

  <!-- Center: Title -->
  <div class="title-section center">
    <span class="title-text">Dejima KVM</span>
  </div>

  <!-- Right: Window controls -->
  <div class="title-section right" style="--wails-draggable: no-drag">
    <button class="win-btn minimize" on:click={doMinimise} title="Minimize">
      <svg width="10" height="1" viewBox="0 0 10 1"><rect width="10" height="1" fill="currentColor"/></svg>
    </button>
    <button class="win-btn maximize" on:click={doToggleFullscreen} title={$isFullscreen ? 'Restore' : 'Maximize'}>
      {#if $isFullscreen}
        <svg width="10" height="10" viewBox="0 0 10 10">
          <path d="M2 0h8v8h-2v2H0V2h2V0zm1 1v1h5v5h1V1H3zm-2 2v6h6V3H1z" fill="currentColor"/>
        </svg>
      {:else}
        <svg width="10" height="10" viewBox="0 0 10 10"><rect x="0" y="0" width="10" height="10" fill="none" stroke="currentColor" stroke-width="1"/></svg>
      {/if}
    </button>
    <button class="win-btn close" on:click={doClose} title="Close">
      <svg width="10" height="10" viewBox="0 0 10 10">
        <path d="M1 0l4 4 4-4 1 1-4 4 4 4-1 1-4-4-4 4-1-1 4-4-4-4z" fill="currentColor"/>
      </svg>
    </button>
  </div>
</div>

<style>
  .hot-zone {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 4px;
    z-index: 1001;
  }

  .title-bar {
    display: flex;
    align-items: center;
    height: 32px;
    min-height: 32px;
    background: #0f172a;
    color: #e2e8f0;
    font-size: 12px;
    position: relative;
    z-index: 1000;
    user-select: none;
    transition: transform 0.2s ease;
  }

  .title-bar.fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 1000;
  }

  .title-bar.hidden {
    transform: translateY(-100%);
  }

  .title-section {
    display: flex;
    align-items: center;
    height: 100%;
  }

  .title-section.left {
    flex: 1;
    justify-content: flex-start;
  }

  .title-section.center {
    flex: 0 0 auto;
  }

  .title-section.right {
    flex: 1;
    justify-content: flex-end;
  }

  .title-text {
    color: #94a3b8;
    font-size: 12px;
  }

  /* Tools button */
  .tools-menu-wrapper {
    position: relative;
  }

  .tools-btn {
    background: none;
    border: none;
    color: #e2e8f0;
    font-size: 12px;
    padding: 0 12px;
    height: 32px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .tools-btn:hover {
    background: #1e293b;
  }

  .arrow {
    font-size: 8px;
  }

  /* Dropdown */
  .dropdown {
    position: absolute;
    top: 32px;
    left: 0;
    min-width: 200px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 4px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.5);
    z-index: 1010;
    padding: 4px 0;
  }

  .dropdown-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 6px 12px;
    background: none;
    border: none;
    color: #e2e8f0;
    font-size: 12px;
    cursor: pointer;
    text-align: left;
  }

  .dropdown-item:hover {
    background: #334155;
  }

  .dropdown-separator {
    height: 1px;
    background: #334155;
    margin: 4px 0;
  }

  .submenu-arrow {
    font-size: 8px;
    margin-left: 8px;
  }

  .submenu {
    padding: 2px 0 2px 12px;
    border-left: 2px solid #334155;
    margin-left: 10px;
  }

  .submenu .dropdown-item {
    padding: 4px 12px;
    font-size: 11px;
  }

  /* Window control buttons */
  .win-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 32px;
    background: none;
    border: none;
    color: #94a3b8;
    cursor: pointer;
  }

  .win-btn:hover {
    background: #1e293b;
    color: #e2e8f0;
  }

  .win-btn.close:hover {
    background: #dc2626;
    color: #ffffff;
  }
</style>
