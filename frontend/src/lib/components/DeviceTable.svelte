<script>
  import { onMount } from 'svelte';
  
  let devices = [];
  let error = null;
  let loading = true;

  async function loadDevices() {
    try {
      const res = await fetch('/api/dashboard');
      if (res.ok) {
        const data = await res.json();
        devices = data.devices || [];
        error = null;
      } else {
        error = 'Failed to load devices';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadDevices();
    setInterval(loadDevices, 5000);
  });

  function getFuelColor(percent) {
    if (percent < 10) return '#ef4444';
    if (percent < 20) return '#f59e0b';
    if (percent < 50) return '#eab308';
    return '#10b981';
  }

  function getConnectionClass(conn) {
    switch(conn) {
      case 'ONLINE': return 'status-online';
      case 'STALE': return 'status-stale';
      default: return 'status-offline';
    }
  }

  function getEquipClass(status) {
    switch(status) {
      case 'running': return 'equip-running';
      case 'idle': return 'equip-idle';
      case 'maintenance': return 'equip-maintenance';
      default: return '';
    }
  }
</script>

<div class="table-container">
  <div class="table-header">
    <h2>Device Status</h2>
  </div>
  
  {#if loading}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading devices...</span>
    </div>
  {:else if error}
    <div class="error-message" style="margin: 1rem;">&#9888; {error}</div>
  {:else if devices.length === 0}
    <div class="no-data">No devices found</div>
  {:else}
    <table class="device-table">
      <thead>
        <tr>
          <th>Device</th>
          <th>Site</th>
          <th>Fuel</th>
          <th>Temp</th>
          <th>Flow Rate</th>
          <th>Equipment</th>
          <th>Connection</th>
          <th>Last Seen</th>
        </tr>
      </thead>
      <tbody>
        {#each devices as device}
          <tr>
            <td><a href="/devices/{device.device_id}" class="device-link">{device.device_id}</a></td>
            <td>{device.site || '-'}</td>
            <td>
              <div class="fuel-container">
                <div class="fuel-bar-track">
                  <div class="fuel-bar-fill" style="width: {Math.max(device.fuel_percent, 2)}%; background: {getFuelColor(device.fuel_percent)}"></div>
                </div>
                <span class="fuel-value" style="color: {getFuelColor(device.fuel_percent)}">{Math.round(device.fuel_percent)}%</span>
              </div>
            </td>
            <td style={device.temperature > 85 ? 'color: #ef4444; font-weight: 600;' : ''}>{device.temperature.toFixed(1)}&deg;C</td>
            <td>{device.flow_rate.toFixed(1)} L/min</td>
            <td><span class="badge {getEquipClass(device.equipment_status)}">{device.equipment_status}</span></td>
            <td>
              <span class="badge {getConnectionClass(device.connection)}">
                <span class="badge-dot"></span>
                {device.connection}
              </span>
            </td>
            <td style="color: var(--text-secondary); font-size: 0.85rem;">{device.last_seen ? new Date(device.last_seen).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'}) : '-'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<style>
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }
  
  .loading-spinner {
    display: inline-block;
    width: 24px;
    height: 24px;
    border: 3px solid var(--border-light);
    border-top-color: var(--accent-blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 2rem;
    color: var(--text-muted);
    gap: 0.75rem;
  }
</style>
