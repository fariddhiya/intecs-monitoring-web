<script>
  import { createEventDispatcher } from 'svelte';
  
  export let alerts = [];
  
  const dispatch = createEventDispatcher();
  let loading = true;
  let error = null;
  let activeTab = 'active';
  let allAlerts = [];
  let historyAlerts = [];

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

  async function loadHistory() {
    try {
      const res = await fetch('/api/alerts/history?limit=100');
      if (res.ok) {
        historyAlerts = await res.json();
      }
    } catch (e) {
      console.error('Error loading alert history:', e);
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

  $: visibleAlerts = activeTab === 'history' ? historyAlerts : alerts;

  loadAlerts();
  loadHistory();
  setInterval(loadAlerts, 10000);
  setInterval(loadHistory, 60000);

  function getSeverityClass(severity) {
    switch(severity?.toLowerCase()) {
      case 'critical': return 'severity-critical';
      case 'warning': return 'severity-warning';
      case 'high': return 'severity-high';
      case 'medium': return 'severity-medium';
      default: return '';
    }
  }

  function getSeverityIcon(severity) {
    switch(severity?.toLowerCase()) {
      case 'critical': return '&#9888;&#65039;';
      case 'warning': return '&#9888;';
      case 'high': return '&#128276;';
      case 'medium': return '&#9432;';
      default: return '&#8505;';
    }
  }

  function getTimeAgo(ts) {
    if (!ts) return '-';
    const now = new Date();
    const created = new Date(ts);
    const diffMs = now - created;
    const diffMin = Math.floor(diffMs / 60000);
    const diffHour = Math.floor(diffMin / 60);
    const diffDay = Math.floor(diffHour / 24);
    
    if (diffMin < 1) return 'Just now';
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHour < 24) return `${diffHour}h ago`;
    return `${diffDay}d ago`;
  }

  function requestNotificationPermission() {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }
  }

  window.addEventListener('intecs:alert', (e) => {
    const alert = e.detail;
    if (!alert || !alert.severity) return;
    
    // Browser notification
    if ('Notification' in window && Notification.permission === 'granted') {
      const icon = alert.severity.toLowerCase() === 'critical' ? '🔴' : '🟡';
      new Notification(`INTECS ${icon} ${alert.type}`, {
        body: alert.message,
        tag: alert.device_id,
        icon: '/favicon.ico',
      });
    }
    
    // Reload alerts list
    loadAlerts();
    loadHistory();
  });

  // Request permission on mount
  requestNotificationPermission();
</script>

<div class="alerts-section">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
    <div style="display: flex; gap: 0.5rem;">
      <button 
        class="tab-btn {activeTab === 'active' ? 'active' : ''}" 
        on:click={() => activeTab = 'active'}>
        Active ({alerts.length})
      </button>
      <button 
        class="tab-btn {activeTab === 'history' ? 'active' : ''}" 
        on:click={() => { activeTab = 'history'; loadHistory(); }}>
        History ({historyAlerts.length})
      </button>
    </div>
    <span class="alert-count">{visibleAlerts.length} alert{visibleAlerts.length !== 1 ? 's' : ''}</span>
  </div>
  
  {#if loading && visibleAlerts.length === 0}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading alerts...</span>
    </div>
  {:else if visibleAlerts.length === 0}
    <div class="no-data">
      {activeTab === 'active' ? 'No active alerts' : 'No alert history'}
    </div>
  {:else}
    <div class="alerts-list">
      {#each visibleAlerts as alert}
        <div class="alert-item {getSeverityClass(alert.severity)}">
          <div class="alert-content">
            <div class="alert-top">
              <span class="alert-badge {getSeverityClass(alert.severity)}">
                <span class="badge-icon" innerHTML={getSeverityIcon(alert.severity)}></span>
                {alert.severity?.toUpperCase() || 'INFO'}
              </span>
              <span class="alert-type">{alert.type}</span>
              <span class="alert-device">{alert.device_id}</span>
            </div>
            <p class="alert-message">{alert.message}</p>
            <div class="alert-footer">
              <span class="alert-time">{getTimeAgo(alert.created_at)}</span>
              {#if alert.status === 'active'}
                <button class="ack-btn" on:click={() => handleAcknowledge(alert.id)}>Acknowledge</button>
              {:else}
                <span class="resolved-badge">Resolved {getTimeAgo(alert.resolved_at)}</span>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
  
  {#if error}
    <div class="error-message">&#9888; {error}</div>
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
  
  .tab-btn {
    background: transparent;
    border: none;
    padding: 0.5rem 1rem;
    font-size: 0.85rem;
    color: var(--text-secondary);
    cursor: pointer;
    border-radius: var(--radius-md);
    transition: all 0.2s ease;
    font-weight: 500;
  }
  
  .tab-btn:hover {
    color: var(--text-primary);
    background: var(--bg-tertiary);
  }
  
  .tab-btn.active {
    background: var(--accent-blue);
    color: white;
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
    border-left: 4px solid transparent;
    transition: background-color 0.2s ease;
  }
  
  .alert-item:hover {
    background: var(--bg-tertiary);
  }
  
  .alert-item:last-child {
    border-bottom: none;
  }
  
  .alert-item.severity-critical {
    border-left-color: #ef4444;
    background: linear-gradient(90deg, rgba(239, 68, 68, 0.05), transparent);
  }
  
  .alert-item.severity-warning {
    border-left-color: #f59e0b;
    background: linear-gradient(90deg, rgba(245, 158, 11, 0.05), transparent);
  }
  
  .alert-item.severity-high {
    border-left-color: #f97316;
  }
  
  .alert-item.severity-medium {
    border-left-color: #0ea5e9;
  }
  
  .alert-top {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.25rem;
    flex-wrap: wrap;
  }
  
  .alert-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.15rem 0.6rem;
    border-radius: 100px;
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  .alert-badge.severity-critical {
    background: #fee2e2;
    color: #dc2626;
  }
  
  .alert-badge.severity-warning {
    background: #fef3c7;
    color: #d97706;
  }
  
  .alert-badge.severity-high {
    background: #ffedd5;
    color: #ea580c;
  }
  
  .alert-badge.severity-medium {
    background: #e0f2fe;
    color: #0284c7;
  }
  
  .alert-type {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
  }
  
  .alert-device {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-family: ui-monospace, monospace;
  }
  
  .alert-message {
    font-size: 0.9rem;
    color: var(--text-secondary);
    line-height: 1.5;
    margin: 0.25rem 0;
  }
  
  .alert-footer {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 0.5rem;
  }
  
  .alert-time {
    font-size: 0.8rem;
    color: var(--text-muted);
  }
  
  .ack-btn {
    background: var(--accent-blue);
    color: white;
    border: none;
    padding: 0.35rem 0.75rem;
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
    transition: background 0.15s ease;
  }
  
  .ack-btn:hover {
    background: var(--accent-blue-dark);
  }
  
  .resolved-badge {
    font-size: 0.8rem;
    color: var(--success-text);
    background: var(--success-bg);
    padding: 0.25rem 0.6rem;
    border-radius: var(--radius-sm);
  }
  
  .badge-icon {
    font-style: normal;
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
  
  .no-data {
    text-align: center;
    padding: 3rem 2rem;
    color: var(--text-muted);
  }
  
  .error-message {
    padding: 1rem 1.25rem;
    color: var(--danger-text);
    background: var(--danger-bg);
    font-size: 0.9rem;
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
