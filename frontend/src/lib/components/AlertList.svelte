<script>
  let alerts = [];
  let loading = true;
  let error = null;

  async function loadAlerts() {
    try {
      const res = await fetch('/api/alerts?status=active');
      if (res.ok) {
        alerts = await res.json();
      } else {
        error = 'Failed to load alerts';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    } finally {
      loading = false;
    }
  }

  async function acknowledge(id) {
    try {
      await fetch(`/api/alerts/${id}/acknowledge`, { method: 'POST' });
      await loadAlerts();
    } catch (e) {
      console.error('Failed to acknowledge alert:', e);
    }
  }

  loadAlerts();
  setInterval(loadAlerts, 10000);

  function getSeverityClass(severity) {
    switch(severity) {
      case 'critical': return 'severity-critical';
      case 'high': return 'severity-high';
      case 'medium': return 'severity-medium';
      case 'warning': return 'severity-warning';
      default: return '';
    }
  }
</script>

{#if alerts.length > 0}
  <div class="alerts-section">
    <h2 style="margin-bottom: 1rem;">Active Alerts</h2>
    <div class="alerts-list">
      {#each alerts.slice(0, 5) as alert}
        <div class="alert-item {getSeverityClass(alert.severity)}">
          <div class="alert-content">
            <span class="alert-type">{alert.type}</span>
            <span class="alert-device">{alert.device_id}</span>
            <p>{alert.message}</p>
            <span class="alert-time">{new Date(alert.created_at).toLocaleString()}</span>
          </div>
          <button class="ack-btn" on:click={() => acknowledge(alert.id)}>Acknowledge</button>
        </div>
      {/each}
    </div>
  </div>
{/if}

{#if loading && alerts.length === 0}
  <p>Loading alerts...</p>
{/if}

{#if error}
  <div class="error-message">{error}</div>
{/if}

<style>
  .alerts-section {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    margin-top: 2rem;
  }
  
  .alerts-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  
  .alert-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-left: 4px solid #ccc;
    border-radius: 4px;
    background: #fafafa;
  }
  
  .severity-critical {
    border-left-color: #f44336;
    background: #ffebee;
  }
  
  .severity-high {
    border-left-color: #FF9800;
    background: #fff3e0;
  }
  
  .severity-medium {
    border-left-color: #2196F3;
    background: #e3f2fd;
  }
  
  .severity-warning {
    border-left-color: #FFC107;
    background: #fffde7;
  }
  
  .alert-content {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  
  .alert-type {
    font-weight: bold;
    text-transform: uppercase;
    font-size: 0.8rem;
    color: #666;
  }
  
  .alert-device {
    font-size: 0.9rem;
    color: #2196F3;
  }
  
  .alert-time {
    font-size: 0.75rem;
    color: #999;
  }
  
  .ack-btn {
    background: #4CAF50;
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85rem;
  }
  
  .ack-btn:hover {
    background: #388E3C;
  }
  
  .error-message {
    background: #ffebee;
    color: #c62828;
    padding: 1rem;
    border-radius: 8px;
    margin-top: 1rem;
  }
</style>
