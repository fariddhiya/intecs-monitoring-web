<script context="module">
  export function load({ params }) {
    return { id: params.id };
  }
</script>

<script>
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  
  $: id = $page.params.id;
  let device = null;
  let telemetry = [];
  let loading = true;
  let error = null;

  async function loadDevice() {
    try {
      const res = await fetch(`/api/devices/${id}`);
      if (res.ok) {
        device = await res.json();
      } else {
        error = 'Device not found';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    } finally {
      loading = false;
    }
  }

  async function loadTelemetry() {
    try {
      const res = await fetch(`/api/devices/${id}/telemetry?limit=20`);
      if (res.ok) {
        telemetry = await res.json();
      }
    } catch (e) {
      console.error('Failed to load telemetry:', e);
    }
  }

  onMount(() => {
    loadDevice();
    loadTelemetry();
    setInterval(loadDevice, 5000);
    setInterval(loadTelemetry, 10000);
  });

  function getConnClass(conn) {
    switch(conn) {
      case 'ONLINE': return 'conn-online';
      case 'STALE': return 'conn-stale';
      default: return 'conn-offline';
    }
  }

  function getFuelColor(percent) {
    if (percent < 10) return '#f44336';
    if (percent < 20) return '#FF9800';
    return '#4CAF50';
  }
</script>

<div class="detail-page">
  {#if loading}
    <p>Loading device details...</p>
  {:else if error}
    <div class="error">{error}</div>
  {:else if device}
    <a href="/devices" class="back-link">&larr; Back to Devices</a>
    
    <h1>{device.device_id}</h1>
    
    <div class="info-grid">
      <div class="info-card">
        <h3>Site</h3>
        <p>{device.site || 'Unknown'}</p>
      </div>
      
      <div class="info-card">
        <h3>Connection</h3>
        <span class="badge {getConnClass(device.connection)}">{device.connection || 'UNKNOWN'}</span>
      </div>
      
      <div class="info-card">
        <h3>Equipment Status</h3>
        <span class="badge equip-{device.equipment_status || ''}">{device.equipment_status || '-'}</span>
      </div>
    </div>
    
    <div class="metrics-grid">
      <div class="metric-card">
        <h3>Fuel Level</h3>
        <div class="metric-value" style="color: {getFuelColor(device.fuel_percent)}">
          {Math.round(device.fuel_percent)}%
        </div>
        <div class="fuel-track">
          <div class="fuel-fill" style="width: {device.fuel_percent}%; background: {getFuelColor(device.fuel_percent)}"></div>
        </div>
      </div>
      
      <div class="metric-card">
        <h3>Temperature</h3>
        <div class="metric-value">{device.temperature.toFixed(1)}°C</div>
      </div>
      
      <div class="metric-card">
        <h3>Flow Rate</h3>
        <div class="metric-value">{device.flow_rate.toFixed(1)} L/min</div>
      </div>
      
      <div class="metric-card">
        <h3>Last Seen</h3>
        <div class="metric-value" style="font-size: 1rem;">
          {device.last_seen ? new Date(device.last_seen).toLocaleString() : '-'}
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .detail-page {
    max-width: 900px;
    margin: 0 auto;
    padding: 2rem 1rem;
  }
  
  .back-link {
    color: #2196F3;
    text-decoration: none;
    display: inline-block;
    margin-bottom: 1.5rem;
  }
  
  .back-link:hover {
    text-decoration: underline;
  }
  
  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
    margin: 1.5rem 0;
  }
  
  .info-card {
    background: white;
    border-radius: 8px;
    padding: 1rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }
  
  .info-card h3 {
    font-size: 0.8rem;
    color: #666;
    text-transform: uppercase;
    margin-bottom: 0.5rem;
  }
  
  .info-card p {
    font-size: 1.1rem;
    margin: 0;
  }
  
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
    margin: 1.5rem 0;
  }
  
  .metric-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }
  
  .metric-card h3 {
    font-size: 0.85rem;
    color: #666;
    text-transform: uppercase;
    margin-bottom: 0.75rem;
  }
  
  .metric-value {
    font-size: 2rem;
    font-weight: bold;
    color: #333;
  }
  
  .fuel-track {
    height: 8px;
    background: #e0e0e0;
    border-radius: 4px;
    margin-top: 0.75rem;
    overflow: hidden;
  }
  
  .fuel-fill {
    height: 100%;
    transition: width 0.3s ease;
  }
  
  .badge {
    padding: 0.25rem 0.75rem;
    border-radius: 12px;
    font-size: 0.9rem;
    font-weight: 500;
  }
  
  .conn-online { background: #e8f5e9; color: #2e7d32; }
  .conn-stale { background: #fff3e0; color: #ef6c00; }
  .conn-offline { background: #ffebee; color: #c62828; }
  
  .equip-running { background: #e3f2fd; color: #1565c0; }
  .equip-idle { background: #f3e5f5; color: #6a1b9a; }
  .equip-maintenance { background: #eceff1; color: #455a64; }
  
  .error {
    background: #ffebee;
    color: #c62828;
    padding: 1rem;
    border-radius: 8px;
  }
</style>
