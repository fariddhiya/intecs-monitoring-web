<script>
  import { onMount } from 'svelte';
  
  let devices = [];
  let error = null;

  async function loadDevices() {
    try {
      const res = await fetch('/api/dashboard');
      if (res.ok) {
        const data = await res.json();
        devices = data.devices;
        error = null;
      } else {
        error = 'Failed to load devices';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    }
  }

  onMount(() => {
    loadDevices();
    setInterval(loadDevices, 5000);
  });

  function getFuelColor(percent) {
    if (percent < 10) return '#f44336';
    if (percent < 20) return '#FF9800';
    if (percent < 50) return '#FFC107';
    return '#4CAF50';
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
  <h2 style="margin-bottom: 1rem;">Device Status</h2>
  
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
          <td>{device.site}</td>
          <td>
            <span class="fuel-bar" style="width: {Math.max(device.fuel_percent, 5)}%">
              {Math.round(device.fuel_percent)}%
            </span>
            <span class="fuel-value" style="color: {getFuelColor(device.fuel_percent)}">
              {Math.round(device.fuel_percent)}%
            </span>
          </td>
          <td>{device.temperature.toFixed(1)}°C</td>
          <td>{device.flow_rate.toFixed(1)} L/min</td>
          <td>
            <span class="badge {getEquipClass(device.equipment_status)}">
              {device.equipment_status}
            </span>
          </td>
          <td>
            <span class="badge {getConnectionClass(device.connection)}">
              {device.connection}
            </span>
          </td>
          <td>{device.last_seen ? new Date(device.last_seen).toLocaleTimeString() : '-'}</td>
        </tr>
      {:else}
        <tr>
          <td colspan="8" class="no-data">No devices found</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .table-container {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    overflow-x: auto;
  }
  
  .device-table {
    width: 100%;
    border-collapse: collapse;
  }
  
  .device-table th {
    text-align: left;
    padding: 0.75rem;
    border-bottom: 2px solid #eee;
    color: #666;
    font-weight: 600;
    font-size: 0.85rem;
    text-transform: uppercase;
  }
  
  .device-table td {
    padding: 0.75rem;
    border-bottom: 1px solid #f5f5f5;
    vertical-align: middle;
  }
  
  .device-link {
    color: #2196F3;
    text-decoration: none;
    font-weight: 500;
  }
  
  .device-link:hover {
    text-decoration: underline;
  }
  
  .fuel-value {
    font-weight: bold;
    margin-right: 0.5rem;
  }
  
  .fuel-bar {
    display: inline-block;
    height: 6px;
    border-radius: 3px;
    background: #e0e0e0;
    margin-right: 0.5rem;
    vertical-align: middle;
    position: relative;
  }
  
  .badge {
    padding: 0.25rem 0.75rem;
    border-radius: 12px;
    font-size: 0.8rem;
    font-weight: 500;
    text-transform: capitalize;
  }
  
  .status-online {
    background: #e8f5e9;
    color: #2e7d32;
  }
  
  .status-stale {
    background: #fff3e0;
    color: #ef6c00;
  }
  
  .status-offline {
    background: #ffebee;
    color: #c62828;
  }
  
  .equip-running {
    background: #e3f2fd;
    color: #1565c0;
  }
  
  .equip-idle {
    background: #f3e5f5;
    color: #6a1b9a;
  }
  
  .equip-maintenance {
    background: #eceff1;
    color: #455a64;
  }
  
  .no-data {
    text-align: center;
    padding: 2rem !important;
    color: #999;
  }
</style>
