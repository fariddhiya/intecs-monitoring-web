<script>
  import { page } from '$app/stores';
  import { onMount } from 'svelte';

  $: deviceId = $page.params.id;
  
  let device = null;
  let telemetry = [];
  let loading = true;
  let error = null;

  async function refreshDevice() {
    try {
      const res = await fetch(`/api/devices/${deviceId}`);
      if (res.ok) {
        device = await res.json();
        error = null;
      } else {
        if (!device) {
          error = 'Device not found';
        }
      }
    } catch (e) {
      if (!device) {
        error = e.message;
      }
    } finally {
      loading = false;
    }
  }

  async function refreshTelemetry() {
    try {
      const res = await fetch(`/api/devices/${deviceId}/telemetry?limit=30`);
      if (res.ok) {
        telemetry = await res.json();
      }
    } catch (e) {
      console.error(e);
    }
  }

  onMount(() => {
    refreshDevice();
    refreshTelemetry();
    setInterval(refreshDevice, 5000);
    setInterval(refreshTelemetry, 10000);
  });

  $: fuelColor = device ? getFuelColor(device.fuel_percent) : '#94a3b8';
  $: tempWarning = device && device.temperature > 85;
  $: connClass = device ? getConnClass(device.connection) : '';
  $: equipStatus = device?.equipment_status || '-';
  
  $: fuelHistory = telemetry.slice(-20);
  $: tempHistory = telemetry.slice(-20);
  $: maxTemp = tempHistory.length > 0 ? Math.max(...tempHistory.map(t => t.temperature), 100) : 100;

  function getConnClass(c) {
    switch(c) {
      case 'ONLINE': return 'conn-online';
      case 'STALE': return 'conn-stale';
      default: return 'conn-offline';
    }
  }

  function getFuelColor(p) {
    if (p < 10) return '#ef4444';
    if (p < 20) return '#f59e0b';
    if (p < 50) return '#eab308';
    return '#10b981';
  }

  function formatTime(ts) {
    return ts ? new Date(ts).toLocaleString([], {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    }) : '-';
  }

  function getTimeLabel(ts) {
    const d = new Date(ts);
    const now = new Date();
    const diffSec = Math.floor((now - d) / 1000);
    return diffSec < 60 ? `${diffSec}s ago` : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
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

<div class="detail-page">
  <a href="/devices" class="back-link">&#8592; Back to Devices</a>

  {#if error}
    <div class="error-state">
      <span class="error-icon">&#9888;</span>
      <h2>{error}</h2>
      <button class="retry-btn" on:click={refreshDevice}>Retry</button>
    </div>
  {:else if !device}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading device data...</span>
    </div>
  {:else}
    <div class="detail-header">
      <div>
        <h1 class="device-id">{device.device_id}</h1>
        <p class="device-subtitle">{device.site || 'Unknown Site'} &bull; {device.name || 'Equipment Monitor'}</p>
      </div>
      <span class="badge {connClass}">
        <span class="badge-dot"></span>
        {device.connection || 'UNKNOWN'}
      </span>
    </div>

    <div class="info-grid">
      <div class="mini-card">
        <span class="label">Connection</span>
        <span class="value"><span class="badge {connClass}" style="font-size: 0.75rem;"><span class="badge-dot"></span> {device.connection || '-'}</span></span>
      </div>
      <div class="mini-card">
        <span class="label">Equipment</span>
        <span class="value"><span class="badge {getEquipClass(equipStatus)}" style="font-size: 0.75rem;">{equipStatus}</span></span>
      </div>
      <div class="mini-card">
        <span class="label">Last Seen</span>
        <span class="value mono">{formatTime(device.last_seen)}</span>
      </div>
    </div>

    <div class="metrics-section">
      <div class="metric-card">
        <div class="metric-top">
          <span class="metric-icon">&#9881;</span>
          <span class="metric-label">Fuel Level</span>
        </div>
        <div class="metric-value" style="color: {fuelColor}">{Math.round(device.fuel_percent)}%</div>
        <div class="fuel-track">
          <div class="fuel-fill" style="width: {device.fuel_percent}%; background: {fuelColor}"></div>
        </div>
        <span class="metric-sub">{Math.round(device.fuel_level)} Liters</span>
      </div>

      <div class="metric-card">
        <div class="metric-top">
          <span class="metric-icon">&#127776;</span>
          <span class="metric-label">Temperature</span>
        </div>
        <div class="metric-value" style={tempWarning ? 'color: #ef4444;' : ''}>{device.temperature.toFixed(1)}&deg;C</div>
        <span class="metric-sub" style={tempWarning ? 'color: #ef4444;' : ''}>{tempWarning ? 'High Temperature' : 'Normal Range'}</span>
      </div>

      <div class="metric-card">
        <div class="metric-top">
          <span class="metric-icon">&#128167;</span>
          <span class="metric-label">Flow Rate</span>
        </div>
        <div class="metric-value" style="color: #0ea5e9">{device.flow_rate.toFixed(1)}</div>
        <span class="metric-sub">Liters/min</span>
      </div>
    </div>

    {#if fuelHistory.length > 0}
      <div class="chart-card">
        <div class="chart-header">
          <h2>Fuel Percentage History</h2>
          <span class="chart-legend">20 data points</span>
        </div>
        <div class="bars-container">
          {#each fuelHistory as point}
            <div class="bar-wrapper">
              <div class="bar" 
                   style="height: {point.fuel_percentage}%; background: {getFuelColor(point.fuel_percentage)}" 
                   title="{point.fuel_percentage.toFixed(1)}%">
              </div>
              {#if fuelHistory.indexOf(point) % Math.ceil(fuelHistory.length / 5) === 0}
                <span class="bar-time">{getTimeLabel(point.timestamp)}</span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if tempHistory.length > 0}
      <div class="chart-card">
        <div class="chart-header">
          <h2>Temperature History</h2>
          <span class="chart-legend">20 data points</span>
        </div>
        <div class="bars-container">
          {#each tempHistory as point}
            <div class="bar-wrapper">
              <div class="bar temp-bar" 
                   style="height: {(point.temperature / maxTemp) * 100}%; background: {point.temperature > 85 ? '#ef4444' : '#3b82f6'}" 
                   title="{point.temperature.toFixed(1)}&deg;C">
              </div>
              {#if tempHistory.indexOf(point) % Math.ceil(tempHistory.length / 5) === 0}
                <span class="bar-time">{getTimeLabel(point.timestamp)}</span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  .detail-page {
    max-width: 1200px;
    margin: 0 auto;
  }
  
  .back-link {
    color: var(--accent-blue);
    text-decoration: none;
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
    font-weight: 500;
    font-size: 0.9rem;
    padding: 0.5rem 0;
  }
  
  .back-link:hover {
    text-decoration: underline;
  }
  
  /* Header */
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1.5rem;
    gap: 1rem;
  }
  
  .device-id {
    font-size: 2rem;
    font-weight: 700;
    color: var(--text-primary);
    letter-spacing: -0.025em;
    margin-bottom: 0.25rem;
  }
  
  .device-subtitle {
    color: var(--text-secondary);
    font-size: 1rem;
  }
  
  /* Info Grid */
  .info-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }
  
  .mini-card {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: 1.25rem;
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  /* Metrics */
  .metrics-section {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }
  
  .metric-card {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: 1.5rem;
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  
  .metric-top {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .metric-icon {
    font-size: 1.25rem;
  }
  
  .metric-label {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  .metric-value {
    font-size: 2.25rem;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.2;
  }
  
  .metric-sub {
    font-size: 0.875rem;
    color: var(--text-muted);
  }
  
  .fuel-track {
    height: 8px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    overflow: hidden;
  }
  
  .fuel-fill {
    height: 100%;
    transition: width 0.5s ease;
    border-radius: 4px;
  }
  
  /* Charts */
  .chart-card {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    padding: 1.5rem;
    margin-bottom: 2rem;
  }
  
  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.25rem;
  }
  
  .chart-header h2 {
    font-size: 1rem;
    color: var(--text-secondary);
    font-weight: 600;
  }
  
  .chart-legend {
    font-size: 0.8rem;
    color: var(--text-muted);
    background: var(--bg-tertiary);
    padding: 0.25rem 0.75rem;
    border-radius: 100px;
  }
  
  .bars-container {
    display: flex;
    align-items: flex-end;
    gap: 4px;
    height: 180px;
    overflow-x: auto;
    padding-top: 0.5rem;
    scrollbar-width: thin;
  }
  
  .bar-wrapper {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    flex-shrink: 0;
  }
  
  .bar {
    width: 18px;
    border-radius: 4px 4px 0 0;
    transition: height 0.5s ease;
    min-height: 2px;
    cursor: default;
  }
  
  .temp-bar {
    width: 14px;
  }
  
  .bar-time {
    font-size: 0.65rem;
    color: var(--text-muted);
    white-space: nowrap;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .loading-spinner {
    display: inline-block;
    width: 32px;
    height: 32px;
    border: 3px solid var(--border-light);
    border-top-color: var(--accent-blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 5rem 2rem;
    color: var(--text-muted);
    gap: 1rem;
  }
  
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 5rem 2rem;
    text-align: center;
    gap: 1rem;
  }
  
  .error-icon {
    font-size: 3rem;
  }
  
  .error-state h2 {
    color: var(--danger-text);
    font-size: 1.25rem;
  }
  
  .retry-btn {
    background: var(--accent-blue);
    color: white;
    border: none;
    padding: 0.75rem 1.5rem;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-weight: 500;
    font-size: 0.9rem;
    transition: background 0.15s ease;
  }
  
  .retry-btn:hover {
    background: var(--accent-blue-dark);
  }
  
  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.9rem;
  }
  
  @media (max-width: 768px) {
    .device-id {
      font-size: 1.5rem;
    }
    
    .metrics-section {
      grid-template-columns: 1fr;
    }
    
    .info-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    
    .detail-header {
      flex-direction: column;
    }
  }
</style>
