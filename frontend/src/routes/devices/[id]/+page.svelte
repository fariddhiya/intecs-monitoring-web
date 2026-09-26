<script>
  import { page } from '$app/stores';
  import { onMount, onDestroy } from 'svelte';
  import { Chart, registerables } from 'chart.js';
  
  Chart.register(...registerables);

  $: deviceId = $page.params.id;
  
  let device = null;
  let telemetry = [];
  let loading = true;
  let error = null;
  let timeRange = '24h';
  let fuelChart = null;
  let tempChart = null;
  let fuelCanvas = null;
  let tempCanvas = null;
  let chartLoaded = false;
  let wsTelemetryHandler = null;
  let downloading = false;
  let liveStatus = 'connecting'; // connecting | live | updating | error
  let lastUpdated = 0;
  let connectionAttempts = 0;
  let pollTimerId = null;
  let deviceTimerId = null;

  const rangeOptions = [
    { label: '1 Jam', value: '1h' },
    { label: '6 Jam', value: '6h' },
    { label: '24 Jam', value: '24h' },
    { label: '7 Hari', value: '7d' },
  ];

  async function refreshDevice() {
    try {
      setLiveStatus('updating');
      const res = await fetch(`/api/devices/${deviceId}`);
      if (res.ok) {
        device = await res.json();
        error = null;
        if (connectionAttempts > 0) {
          connectionAttempts = 0;
          setLiveStatus('live');
        }
      } else {
        if (!device) {
          error = 'Device not found';
        } else {
          connectionAttempts++;
          if (connectionAttempts >= 3) {
            setError('Unable to retrieve device status.');
          }
        }
      }
    } catch (e) {
      connectionAttempts++;
      if (!device) {
        setError(e.message);
      } else if (connectionAttempts >= 3) {
        setError('Unable to retrieve device status.');
      }
    } finally {
      loading = false;
    }
  }

  async function loadTelemetry() {
    try {
      const res = await fetch(`/api/devices/${deviceId}/telemetry?range=${timeRange}&limit=500`);
      if (res.ok) {
        telemetry = await res.json();
        lastUpdated = Date.now();
        connectionAttempts = 0;
        
        if (telemetry.length === 0 && !error) {
          setLiveStatus('live');
        } else {
          setLiveStatus('live');
        }
      } else {
        throw new Error('Failed to load telemetry');
      }
    } catch (e) {
      console.error('Error loading telemetry:', e);
      connectionAttempts++;
      if (connectionAttempts >= 3) {
        setLiveStatus('error');
      }
    }
  }

  async function exportTelemetryCSV() {
    if (!telemetry.length) return;
    
    downloading = true;
    
    try {
      const headers = ['Timestamp', 'Fuel %', 'Fuel Level', 'Temperature °C', 'Flow Rate L/min'];
      const rows = telemetry.map(t => [
        t.timestamp,
        (t.fuel_percent ?? 0).toFixed(2),
        (t.fuel_level ?? 0).toFixed(2),
        (t.temperature ?? 0).toFixed(2),
        (t.flow_rate ?? 0).toFixed(2),
      ]);
      
      const csvContent = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.setAttribute('href', url);
      
      const now = new Date().toISOString().split('T')[0];
      const rangeLabel = rangeOptions.find(o => o.value === timeRange)?.label || timeRange;
      link.setAttribute('download', `${deviceId}_${rangeLabel}_${now}.csv`);
      link.style.display = 'none';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (e) {
      console.error('Export failed:', e);
    } finally {
      downloading = false;
    }
  }

  function handleWsTelemetry(event) {
    if (event.detail && event.detail.device_id === deviceId) {
      const ts = new Date(event.detail.timestamp).getTime();
      telemetry = telemetry.filter(t => new Date(t.timestamp).getTime() < ts);
      
      const newPoint = {
        timestamp: event.detail.timestamp,
        fuel_percent: event.detail.fuel_percent || 0,
        fuel_level: event.detail.fuel_level || 0,
        temperature: event.detail.temperature || 0,
        flow_rate: event.detail.flow_rate || 0,
      };
      
      telemetry.push(newPoint);
      updateCharts();
    }
  }

  function updateTimeRange(newRange) {
    timeRange = newRange;
    chartLoaded = false;
    destroyCharts();
    loadTelemetry();
  }

  function setLiveStatus(status) {
    liveStatus = status;
  }

  function setError(msg) {
    error = msg;
  }

  function formatTime(ts) {
    return ts ? new Date(ts).toLocaleString([], {
      month: 'short', day: 'numeric',
      hour: '2-digit', minute: '2-digit'
    }) : '-';
  }

  function getTimeLabel(ts) {
    const d = new Date(ts);
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function getRelativeTime(secs) {
    if (secs < 60) return `${Math.round(secs)}s ago`;
    if (secs < 3600) return `${Math.round(secs / 60)}m ago`;
    return `${Math.round(secs / 3600)}h ago`;
  }

  function getFuelColor(p) {
    if (p < 10) return '#ef4444';
    if (p < 20) return '#f59e0b';
    if (p < 50) return '#eab308';
    return '#10b981';
  }

  function getConnClass(c) {
    switch(c) {
      case 'ONLINE': return 'conn-online';
      case 'STALE': return 'conn-stale';
      default: return 'conn-offline';
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

  $: fuelColor = device ? getFuelColor(device.fuel_percent) : '#94a3b8';
  $: tempWarning = device && device.temperature > 85;
  $: connClass = device ? getConnClass(device.connection) : '';
  $: equipStatus = device?.equipment_status || '-';
  $: noDataForRange = !loading && telemetry.length === 0 && device !== null;
  $: liveDotClass = liveStatus === 'live' ? 'live-dot-live' 
                  : liveStatus === 'updating' ? 'live-dot-updating'
                  : liveStatus === 'error' ? 'live-dot-error'
                  : 'live-dot-connecting';

  function destroyCharts() {
    if (fuelChart) {
      fuelChart.destroy();
      fuelChart = null;
    }
    if (tempChart) {
      tempChart.destroy();
      tempChart = null;
    }
  }

  function updateCharts() {
    if (!chartLoaded || !telemetry.length) return;
    
    const labels = telemetry.map(t => getTimeLabel(t.timestamp));
    const fuelData = telemetry.map(t => t.fuel_percent);
    const tempData = telemetry.map(t => t.temperature);

    if (fuelChart) {
      fuelChart.data.labels = labels;
      fuelChart.data.datasets[0].data = fuelData;
      fuelChart.update('default');
    }

    if (tempChart) {
      tempChart.data.labels = labels;
      tempChart.data.datasets[0].data = tempData;
      tempChart.update('default');
    }
  }

  function initCharts() {
    if (!chartLoaded) {
      chartLoaded = true;
    }

    destroyCharts();
    if (!fuelCanvas || !tempCanvas || !telemetry.length) return;

    const ctx1 = fuelCanvas.getContext('2d');
    const ctx2 = tempCanvas.getContext('2d');

    const labels = telemetry.map(t => getTimeLabel(t.timestamp));
    const fuelData = telemetry.map(t => t.fuel_percent);
    const tempData = telemetry.map(t => t.temperature);

    const commonOptions = {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 300 },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: '#1e293b',
          titleColor: '#fff',
          bodyColor: '#cbd5e1',
          padding: 10,
          cornerRadius: 8,
          displayColors: true,
          callbacks: {
            title: function(items) {
              if (items.length) {
                const index = items[0].dataIndex;
                if (telemetry[index]) {
                  return new Date(telemetry[index].timestamp).toLocaleString();
                }
              }
              return '';
            }
          }
        },
      },
      scales: {
        x: {
          grid: { color: 'rgba(148, 163, 184, 0.1)', drawBorder: false },
          ticks: { color: 'var(--text-muted)', maxTicksLimit: 10, font: { size: 11 } },
        },
        y: {
          grid: { color: 'rgba(148, 163, 184, 0.1)', drawBorder: false },
          ticks: { color: 'var(--text-muted)', font: { size: 11 } },
        },
      },
      elements: {
        point: { radius: 2, hoverRadius: 5, hitRadius: 5 },
        line: { tension: 0.3, borderWidth: 2 },
      },
    };

    fuelChart = new Chart(ctx1, {
      type: 'line',
      data: {
        labels: labels,
        datasets: [{
          data: fuelData,
          borderColor: '#10b981',
          backgroundColor: 'rgba(16, 185, 129, 0.1)',
          fill: true,
          yAxisID: 'y',
        }],
      },
      options: {
        ...commonOptions,
        scales: {
          ...commonOptions.scales,
          y: {
            ...commonOptions.scales.y,
            min: 0,
            max: 100,
            ticks: {
              ...commonOptions.scales.y.ticks,
              callback: v => v + '%',
            },
          },
        },
      },
    });

    tempChart = new Chart(ctx2, {
      type: 'line',
      data: {
        labels: labels,
        datasets: [{
          data: tempData,
          borderColor: '#3b82f6',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          fill: true,
        }],
      },
      options: commonOptions,
    });
  }

  $: if (telemetry.length && chartLoaded) {
    updateCharts();
  }

  $: if (telemetry.length > 0) {
    initCharts();
  }

  onMount(() => {
    refreshDevice();
    loadTelemetry();
    
    deviceTimerId = setInterval(refreshDevice, 5000);
    pollTimerId = setInterval(loadTelemetry, 15000);

    wsTelemetryHandler = (e) => handleWsTelemetry(e);
    window.addEventListener('intecs:telemetry', wsTelemetryHandler);
  });

  onDestroy(() => {
    if (deviceTimerId) clearInterval(deviceTimerId);
    if (pollTimerId) clearInterval(pollTimerId);
    destroyCharts();
    if (wsTelemetryHandler) {
      window.removeEventListener('intecs:telemetry', wsTelemetryHandler);
    }
  });
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
        <span class="metric-sub">{device.fuel_level ? Math.round(device.fuel_level) : 0} Liters</span>
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

    <div class="chart-card">
      <div class="chart-header">
        <div class="chart-title-area">
          <h2>Fuel Percentage History</h2>
          <div class="live-status">
            <span class="live-dot {liveDotClass}"></span>
            <span class="live-text">
              {#if liveStatus === 'live'}LIVE{:else if liveStatus === 'updating'}Updating...{:else if liveStatus === 'error'}Connection issue{:else}Connecting...{/if}
              {#if lastUpdated > 0 && liveStatus !== 'error'}
                <span class="live-time">Updated {getRelativeTime((Date.now() - lastUpdated) / 1000)}</span>
              {/if}
            </span>
          </div>
        </div>
        <div class="chart-controls">
          <div class="range-selector">
            {#each rangeOptions as opt}
              <button 
                class="range-btn {timeRange === opt.value ? 'active' : ''}" 
                on:click={() => updateTimeRange(opt.value)}>
                {opt.label}
              </button>
            {/each}
          </div>
          <button class="export-btn" on:click={exportTelemetryCSV} disabled={downloading || !telemetry.length}>
            {#if downloading}&#x21bb; Downloading...{:else}&#128190; Download CSV{/if}
          </button>
        </div>
      </div>
      
      {#if noDataForRange}
        <div class="chart-empty-state">No telemetry data available for this time range.</div>
      {:else}
        <div class="chart-container">
          <canvas bind:this={fuelCanvas}></canvas>
        </div>
      {/if}
    </div>

    <div class="chart-card">
      <div class="chart-header">
        <div class="chart-title-area">
          <h2>Temperature History</h2>
          <div class="live-status">
            <span class="live-dot {liveDotClass}"></span>
            <span class="live-text">
              {#if liveStatus === 'live'}LIVE{:else if liveStatus === 'updating'}Updating...{:else if liveStatus === 'error'}Connection issue{:else}Connecting...{/if}
              {#if lastUpdated > 0 && liveStatus !== 'error'}
                <span class="live-time">Updated {getRelativeTime((Date.now() - lastUpdated) / 1000)}</span>
              {/if}
            </span>
          </div>
        </div>
        <div class="chart-controls">
          <div class="range-selector">
            {#each rangeOptions as opt}
              <button 
                class="range-btn {timeRange === opt.value ? 'active' : ''}" 
                on:click={() => updateTimeRange(opt.value)}>
                {opt.label}
              </button>
            {/each}
          </div>
          <button class="export-btn" on:click={exportTelemetryCSV} disabled={downloading || !telemetry.length}>
            {#if downloading}&#x21bb; Downloading...{:else}&#128190; Download CSV{/if}
          </button>
        </div>
      </div>
      
      {#if noDataForRange}
        <div class="chart-empty-state">No telemetry data available for this time range.</div>
      {:else}
        <div class="chart-container">
          <canvas bind:this={tempCanvas}></canvas>
        </div>
      {/if}
    </div>
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
    transition: color 0.2s ease;
  }
  
  .back-link:hover {
    text-decoration: underline;
    color: var(--accent-blue-dark);
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
    box-shadow: var(--shadow-md);
    padding: 1.5rem;
    margin-bottom: 2rem;
  }
  
  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1.25rem;
    gap: 1rem;
    flex-wrap: wrap;
  }
  
  .chart-title-area {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .chart-header h2 {
    font-size: 1rem;
    color: var(--text-secondary);
    font-weight: 600;
  }
  
  /* Live Status Indicator */
  .live-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .live-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    display: inline-block;
    flex-shrink: 0;
  }
  
  .live-dot-connecting {
    background: var(--text-muted);
  }
  
  .live-dot-live {
    background: var(--success);
    animation: pulse-green 2s infinite;
  }
  
  .live-dot-updating {
    background: var(--warning);
  }
  
  .live-dot-error {
    background: var(--danger);
  }
  
  @keyframes pulse-green {
    0%, 100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4); }
    50% { box-shadow: 0 0 0 4px rgba(16, 185, 129, 0); }
  }
  
  .live-text {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 500;
  }
  
  .live-time {
    margin-left: 0.5rem;
    opacity: 0.7;
  }
  
  .chart-controls {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }
  
  .range-selector {
    display: flex;
    gap: 2px;
    background: var(--bg-tertiary);
    padding: 3px;
    border-radius: var(--radius-md);
  }
  
  .range-btn {
    background: transparent;
    border: none;
    padding: 0.375rem 0.75rem;
    font-size: 0.8rem;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: calc(var(--radius-md) - 2px);
    transition: all 0.2s ease;
    font-weight: 500;
  }
  
  .range-btn:hover {
    color: var(--text-primary);
    background: rgba(128, 128, 128, 0.1);
  }
  
  .range-btn.active {
    background: var(--bg-secondary);
    color: var(--accent-blue);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
    font-weight: 600;
  }
  
  .export-btn {
    background: var(--success-bg);
    color: var(--success-text);
    border: 1px solid var(--success);
    padding: 0.4rem 0.85rem;
    border-radius: var(--radius-sm);
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
    font-weight: 500;
  }
  
  .export-btn:hover:not(:disabled) {
    background: var(--success);
    color: white;
  }
  
  .export-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .chart-container {
    position: relative;
    height: 250px;
    width: 100%;
  }
  
  .chart-empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
    color: var(--text-muted);
    font-size: 0.9rem;
    font-style: italic;
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
    
    .chart-header {
      flex-direction: column;
      align-items: flex-start;
    }
    
    .chart-controls {
      width: 100%;
      justify-content: flex-start;
    }
    
    .chart-container {
      height: 200px;
    }
  }
  
  @media (max-width: 640px) {
    .info-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
