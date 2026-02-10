<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { ListSerialPorts, ConnectSerial, DisconnectSerial, DetectFT232, GetSerialStatus } from '../../../wailsjs/go/main/App';
  import { updateSerial } from '../stores/connection';

  const dispatch = createEventDispatcher();

  let serialPorts: string[] = [];
  let selectedPort = '';
  let connectedPort = '';
  let error = '';

  onMount(async () => {
    await refreshPorts();
    connectedPort = await GetSerialStatus();
    if (connectedPort) {
      selectedPort = connectedPort;
      updateSerial(connectedPort);
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
      dispatch('close');
    } catch (e) {
      error = `Connection failed: ${e}`;
    }
  }

  async function disconnect() {
    await DisconnectSerial();
    connectedPort = '';
    updateSerial('');
  }

  function close() {
    dispatch('close');
  }
</script>

<div class="overlay" on:click|self={close} role="presentation">
  <div class="modal">
    <div class="modal-header">
      <h3>Serial Connection</h3>
      <button class="close-btn" on:click={close}>&times;</button>
    </div>

    <div class="modal-body">
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
    width: 480px;
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
