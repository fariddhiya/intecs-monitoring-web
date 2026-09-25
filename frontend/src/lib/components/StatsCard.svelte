<script>
  let stats = { total_devices: 0, online_devices: 0, offline_devices: 0, active_alerts: 0 };
  let error = null;

  async function loadStats() {
    try {
      const res = await fetch('/api/dashboard');
      if (res.ok) {
        const data = await res.json();
        stats = data;
        error = null;
      } else {
        error = 'Failed to load dashboard data';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    }
  }

  loadStats();
  setInterval(loadStats, 5000);
</script>

<div class="stats-grid">
  <div class="stat-card stat-total">
    <div class="stat-value">{stats.total_devices}</div>
    <div class="stat-label">Total Devices</div>
  </div>
  
  <div class="stat-card stat-online">
    <div class="stat-value">{stats.online_devices}</div>
    <div class="stat-label">Online</div>
  </div>
  
  <div class="stat-card stat-offline">
    <div class="stat-value">{stats.offline_devices}</div>
    <div class="stat-label">Offline</div>
  </div>
  
  <div class="stat-card stat-alerts">
    <div class="stat-value">{stats.active_alerts}</div>
    <div class="stat-label">Active Alerts</div>
  </div>
</div>

{#if error}
  <div class="error-message">⚠️ {error}</div>
{/if}

<style>
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }
  
  .stat-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    text-align: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }
  
  .stat-value {
    font-size: 2.5rem;
    font-weight: bold;
    color: #333;
  }
  
  .stat-label {
    font-size: 0.9rem;
    color: #666;
    margin-top: 0.5rem;
  }
  
  .stat-total .stat-value { color: #2196F3; }
  .stat-online .stat-value { color: #4CAF50; }
  .stat-offline .stat-value { color: #f44336; }
  .stat-alerts .stat-value { color: #FF9800; }
  
  .error-message {
    background: #ffebee;
    color: #c62828;
    padding: 1rem;
    border-radius: 8px;
    margin-bottom: 1rem;
  }
</style>
