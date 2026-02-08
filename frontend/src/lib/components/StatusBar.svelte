<script lang="ts">
  import { connection } from '../stores/connection';
</script>

<div class="status-bar">
  <div class="status-item">
    <span class="dot" class:connected={$connection.videoStreaming}></span>
    Video: {$connection.videoStreaming ? 'Streaming' : 'Off'}
  </div>
  <div class="status-item">
    <span class="dot" class:connected={$connection.serialConnected}></span>
    Serial: {$connection.serialConnected ? $connection.serialPort : 'Disconnected'}
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
</style>
