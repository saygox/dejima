<script lang="ts">
  import VideoDisplay from './lib/components/VideoDisplay.svelte';
  import Toolbar from './lib/components/Toolbar.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import SettingsModal from './lib/components/SettingsModal.svelte';

  let showSettings = false;
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
