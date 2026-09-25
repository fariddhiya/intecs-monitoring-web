<script>
  import { createEventDispatcher } from 'svelte';
  
  export let alerts = [];
  
  const dispatch = createEventDispatcher();
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

  async function handleAcknowledge(id) {
    try {
      await fetch(`/api/alerts/${id}/acknowledge`, { method: 'POST' });
      dispatch('acknowledge', id);
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

  function getSeverityIcon(severity) {
    switch(severity) {
      case 'critical': return '&#9888;&#65039;';
      case 'high': return '&#9888;';
      case 'medium': return '&#9432;';
      case 'warning': return '&#9898;';
      default: return '&#8505;';
    }
  }
</script>

<div class="alerts-section">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center;">
    <h2>Active Alerts</h2>
    <span class="alert-count">{alerts.length} alert{alerts.length !== 1 ? 's' : ''}</span>
  </div>
  
  {#if loading && alerts.length === 0}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading alerts...</span>
    </div>
  {:else if alerts.length === 0}
    <div class="no-data">No active alerts</div>
  {:else}
    <div class="alerts-list">
      {#each alerts.slice(0, 10) as alert}
        <div class="alert-item {getSeverityClass(alert.severity)}">
          <div class="alert-content">
            <div class="alert-top">
              <span class="alert-type">{alert.type}</span>
              <span class="alert-device">{alert.device_id}</span>
            </div>
            <p class="alert-message">{alert.message}</p>
            <span class="alert-time">{new Date(alert.created_at).toLocaleString()}</span>
          </div>
          <button class="ack-btn" on:click={() => handleAcknowledge(alert.id)}>Acknowledge</button>
        </div>
      {/each}
    </div>
  {/if}
  
  {#if error}
    <div class="error-message" style="margin-top: 1rem;">&#9888; {error}</div>
  {/if}
</div>

<style>
  .alerts-section {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }
  
  .alert-count {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-weight: 500;
    background: var(--bg-tertiary);
    padding: 0.25rem 0.75rem;
    border-radius: 100px;
  }
  
  .alerts-list {
    display: flex;
    flex-direction: column;
  }
  
  .alert-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--border-light);
  }
  
  .alert-item:last-child {
    border-bottom: none;
  }
  
  .alert-top {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.25rem;
  }
  
  .alert-message {
    font-size: 0.9rem;
    color: var(--text-secondary);
    line-height: 1.5;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
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
  
  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 3rem 2rem;
    color: var(--text-muted);
    gap: 0.75rem;
  }
  
  @media (max-width: 640px) {
    .alert-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.75rem;
    }
    
    .ack-btn {
      width: 100%;
    }
  }
</style>
