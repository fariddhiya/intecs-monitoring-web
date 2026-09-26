<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  
  const dispatch = createEventDispatcher();
  let activeTab = 'active';
  let allAlerts = [];
  let historyAlerts = [];
  let activePage = 1;
  let historyPage = 1;
  let pageSize = 5;
  let activeTotalPages = 1;
  let activeTotalItems = 0;
  let historyTotalPages = 1;
  let historyTotalItems = 0;
  let loading = true;
  let error = null;

  $: visibleAlerts = activeTab === 'history' ? historyAlerts : allAlerts;
  $: activeCount = activeTotalItems;
  $: historyCount = historyTotalItems;
  $: currentCount = visibleAlerts?.length || 0;
  $: isVisibleFirstPage = (activeTab === 'active' ? activePage : historyPage) <= 1;

  async function loadAlerts() {
    try {
      const res = await fetch(`/api/alerts?page=${activePage}&page_size=${pageSize}&status=active`);
      if (res.ok) {
        const json = await res.json();
        allAlerts = Array.isArray(json.data) ? json.data : json;
        activeTotalItems = json.total_items || allAlerts.length;
        activeTotalPages = json.total_pages || Math.ceil(allAlerts.length / pageSize);
      } else {
        error = 'Failed to load alerts';
      }
    } catch (e) {
      error = e.message;
    }
  }

  async function loadHistory() {
    try {
      const res = await fetch(`/api/alerts/history?page=${historyPage}&page_size=${pageSize}`);
      if (res.ok) {
        const json = await res.json();
        historyAlerts = Array.isArray(json.data) ? json.data : json;
        historyTotalItems = json.total_items || historyAlerts.length;
        historyTotalPages = json.total_pages || Math.ceil(historyAlerts.length / pageSize);
      } else {
        error = 'Failed to load history';
      }
    } catch (e) {
      console.error('Error loading alert history:', e);
    }
  }

  async function handleAcknowledge(id) {
    try {
      await fetch(`/api/alerts/${id}/acknowledge`, { method: 'POST' });
      dispatch('acknowledge', id);
      await Promise.all([loadAlerts(), loadHistory()]);
    } catch (e) {
      console.error('Failed to acknowledge alert:', e);
    }
  }

  function switchTab(tab) {
    activeTab = tab;
    if (tab === 'active') {
      activePage = 1;
      loadAlerts();
    } else {
      historyPage = 1;
      loadHistory();
    }
  }

  function changePage(page) {
    if (activeTab === 'active') {
      if (page >= 1 && page <= activeTotalPages) {
        activePage = page;
        loadAlerts();
      }
    } else {
      if (page >= 1 && page <= historyTotalPages) {
        historyPage = page;
        loadHistory();
      }
    }
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function getPageNumbers(totalPages, currentPage) {
    const pages = [];
    const maxVisible = 7;
    
    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      pages.push(1);
      if (currentPage > 3) pages.push('...');
      
      const start = Math.max(2, currentPage - 1);
      const end = Math.min(totalPages - 1, currentPage + 1);
      
      for (let i = start; i <= end; i++) {
        pages.push(i);
      }
      
      if (currentPage < totalPages - 2) pages.push('...');
      pages.push(totalPages);
    }
    
    return pages;
  }

  function handlePageSizeChange(e) {
    pageSize = parseInt(e.target.value);
    activePage = 1;
    historyPage = 1;
    loadAlerts();
    loadHistory();
  }

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

  function exportCSV(alertsToExport, filename) {
    if (!alertsToExport.length) return;
    
    const headers = ['Type', 'Severity', 'Device ID', 'Message', 'Status', 'Created At', 'Resolved At'];
    const rows = alertsToExport.map(a => [
      a.type,
      a.severity,
      a.device_id,
      a.message,
      a.status,
      a.created_at,
      a.resolved_at || '',
    ]);
    
    const csvContent = [headers.join(','), ...rows.map(r => r.map(v => `"${v}"`).join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `${filename}_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  window.addEventListener('intecs:alert', () => {
    loadAlerts();
    loadHistory();
  });

  onMount(async () => {
    loading = true;
    error = null;
    await Promise.all([loadAlerts(), loadHistory()]);
    loading = false;
    requestNotificationPermission();
  });

  setInterval(() => loadAlerts(), 10000);
  setInterval(() => loadHistory(), 60000);
</script>

<div class="alerts-section">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button 
        class="tab-btn {activeTab === 'active' ? 'active' : ''}" 
        on:click={() => switchTab('active')}>
        Active ({activeCount})
      </button>
      <button 
        class="tab-btn {activeTab === 'history' ? 'active' : ''}" 
        on:click={() => switchTab('history')}>
        History ({historyCount})
      </button>
    </div>
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button 
        class="export-btn-small" 
        on:click={() => exportCSV(visibleAlerts, activeTab === 'active' ? 'active_alerts' : 'alert_history')}>
        &#128196; Export
      </button>
      <span class="alert-count">{currentCount} alert{currentCount !== 1 ? 's' : ''}</span>
    </div>
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
    
    <div class="pagination-container">
      <div class="pagination-info">
        <span class="info-text">
          Showing {((activeTab === 'active' ? activePage : historyPage) - 1) * pageSize + 1}–{Math.min((activeTab === 'active' ? activePage : historyPage) * pageSize, activeCount + historyCount)} of {activeTab === 'active' ? activeCount : historyCount} alerts
        </span>
        <select class="page-size-select" value={pageSize} on:change={handlePageSizeChange}>
          <option value="5">5 per page</option>
          <option value="10">10 per page</option>
          <option value="15">15 per page</option>
        </select>
      </div>
      
      <nav class="pagination-nav">
        <button 
          class="page-btn" 
          disabled={isVisibleFirstPage}
          on:click={() => changePage(activePage - 1)}>
          &laquo; Prev
        </button>
        
        {#each getPageNumbers(activeTab === 'active' ? activeTotalPages : historyTotalPages, activeTab === 'active' ? activePage : historyPage) as page}
          {#if page === '...'}
            <span class="ellipsis">...</span>
          {:else}
            <button 
              class="page-btn {page === (activeTab === 'active' ? activePage : historyPage) ? 'active' : ''}"
              on:click={() => changePage(page)}>
              {page}
            </button>
          {/if}
        {/each}
        
        <button 
          class="page-btn" 
          disabled={(activeTab === 'active' ? activePage : historyPage) >= (activeTab === 'active' ? activeTotalPages : historyTotalPages)}
          on:click={() => changePage(activePage + 1)}>
          Next &raquo;
        </button>
      </nav>
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
  
  .export-btn-small {
    background: transparent;
    border: 1px solid var(--border-light);
    color: var(--text-secondary);
    padding: 0.35rem 0.6rem;
    border-radius: var(--radius-sm);
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .export-btn-small:hover {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-color: var(--accent-blue);
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
    max-height: 600px;
    overflow-y: auto;
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
  
  .pagination-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.25rem;
    border-top: 1px solid var(--border-light);
    flex-wrap: wrap;
    gap: 1rem;
  }
  
  .pagination-info {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  
  .info-text {
    font-size: 0.85rem;
    color: var(--text-muted);
  }
  
  .page-size-select {
    padding: 0.35rem 2rem 0.35rem 0.6rem;
    border: 1px solid var(--border-light);
    border-radius: var(--radius-sm);
    font-size: 0.8rem;
    color: var(--text-primary);
    background: var(--bg-tertiary) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 10 10'%3E%3Cpath fill='%2364748b' d='M5 7L1 3h8z'/%3E%3C/svg%3E") no-repeat right 0.4rem center;
    cursor: pointer;
    appearance: none;
  }
  
  .page-size-select:focus {
    outline: none;
    border-color: var(--accent-blue);
  }
  
  .pagination-nav {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }
  
  .page-btn {
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--border-light);
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-primary);
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.2s ease;
    min-width: 36px;
  }
  
  .page-btn:hover:not(:disabled):not(.active) {
    background: var(--bg-tertiary);
    border-color: var(--accent-blue);
  }
  
  .page-btn.active {
    background: var(--accent-blue);
    border-color: var(--accent-blue);
    color: white;
    font-weight: 600;
  }
  
  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
  
  .ellipsis {
    padding: 0.4rem 0.25rem;
    color: var(--text-muted);
    user-select: none;
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
    
    .pagination-container {
      flex-direction: column;
      align-items: stretch;
    }
    
    .pagination-nav {
      justify-content: center;
      flex-wrap: wrap;
    }
  }
</style>
