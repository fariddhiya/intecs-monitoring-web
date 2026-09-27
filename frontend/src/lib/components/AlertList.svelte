<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();
  let activeTab = $state('active');
  let allAlerts = $state([]);
  let historyAlerts = $state([]);
  let activePage = $state(1);
  let historyPage = $state(1);
  let pageSize = $state(5);
  let activeTotalPages = $state(1);
  let activeTotalItems = $state(0);
  let historyTotalPages = $state(1);
  let historyTotalItems = $state(0);
  let loading = $state(true);
  let error = $state(null);
  let exporting = $state(false);
  let severityCounts = $state({ active: {}, history: {} });

  let showModal = $state(false);
  let selectedSolveAlert = $state(null);
  let solveNotes = $state('');

  let currentUser = null;
  if (typeof window !== 'undefined') {
    try {
      const token = localStorage.getItem('token');
      if (token) {
        currentUser = JSON.parse(atob(token));
      } else {
        const savedUser = localStorage.getItem('user');
        if (savedUser) currentUser = JSON.parse(savedUser);
      }
    } catch (e) {
      currentUser = { email: 'admin@email.com' };
    }
  }
  if (!currentUser) currentUser = { email: 'admin@email.com' };

  function loadSeverityCounts() {
    fetch('/api/alerts/severity-counts')
      .then(r => r.json())
      .then(data => {
        severityCounts = data;
      })
      .catch(e => console.error('Failed to load severity counts:', e));
  }

  let filterSeverities = $state({ active: [], history: [] });

  function getFilteredAlerts(alertList, filterKey) {
    const filters = filterSeverities[filterKey] || [];
    if (!filters.length) return alertList;
    return alertList.filter(a => {
      for (const f of filters) {
        if (f === 'acknowledged') {
          if (a.status === 'acknowledged') return true;
        } else {
          if (a.severity?.toLowerCase() === f) return true;
        }
      }
      return false;
    });
  }

  function toggleFilter(filter) {
    const key = activeTab === 'active' ? 'active' : 'history';
    const filters = filterSeverities[key];
    const idx = filters.indexOf(filter);
    
    if (idx >= 0) {
      filters.splice(idx, 1);
      filterSeverities[key] = [...filters];
    } else {
      filterSeverities[key] = [...filters, filter];
    }
  }

  function isActive(filter) {
    return (filterSeverities[activeTab === 'active' ? 'active' : 'history'] || []).includes(filter);
  }

  function clearFilters() {
    const key = activeTab === 'active' ? 'active' : 'history';
    filterSeverities[key] = [];
    filterSeverities.active = [];
    filterSeverities.history = [];
    filterSeverities = { ...filterSeverities };
  }

  function hasActiveFilters() {
    const activeFilters = filterSeverities['active'] || [];
    const historyFilters = filterSeverities['history'] || [];
    return activeFilters.length > 0 || historyFilters.length > 0;
  }

  function getVisibleFilters() {
    const key = activeTab === 'active' ? 'active' : 'history';
    return filterSeverities[key] || [];
  }

  let visibleAlerts = $derived(getFilteredAlerts(
    activeTab === 'history' ? historyAlerts : allAlerts,
    activeTab === 'history' ? 'history' : 'active'
  ));
  let activeCount = $derived(activeTotalItems);
  let historyCount = $derived(historyTotalItems);
  let currentCount = $derived(visibleAlerts?.length || 0);
  let isVisibleFirstPage = $derived((activeTab === 'active' ? activePage : historyPage) <= 1);

  let activeSeverityTotals = $derived({
    critical: severityCounts.active.critical || 0,
    warning: severityCounts.active.warning || 0,
  });

  let historySeverityTotals = $derived({
    critical: severityCounts.history.critical || 0,
    warning: severityCounts.history.warning || 0,
  });

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
      await fetch(`/api/alerts/${id}/acknowledge`, { method: 'POST', headers: { 'X-User-Email': currentUser.email } });
      dispatch('acknowledge', id);
      await Promise.all([loadAlerts(), loadHistory(), loadSeverityCounts()]);
    } catch (e) {
      console.error('Failed to acknowledge alert:', e);
    }
  }

  function openSolveModal(alert) {
    selectedSolveAlert = alert;
    solveNotes = '';
    showModal = true;
  }

  function closeSolveModal() {
    showModal = false;
    selectedSolveAlert = null;
    solveNotes = '';
  }

  async function handleSolve() {
    if (!solveNotes.trim() || !selectedSolveAlert) return;
    try {
      await fetch(`/api/alerts/${selectedSolveAlert.id}/solve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes: solveNotes, user_email: currentUser.email })
      });
      closeModal();
      await Promise.all([loadAlerts(), loadHistory(), loadSeverityCounts()]);
    } catch (e) {
      console.error('Failed to solve alert:', e);
    }
  }

  function closeModal() {
    showModal = false;
    selectedSolveAlert = null;
    solveNotes = '';
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

  function resetCurrentFilters() {
    const key = activeTab === 'active' ? 'active' : 'history';
    filterSeverities[key] = [];
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
    const newValue = Number(e.target.value);
    if (newValue !== pageSize) {
      pageSize = newValue;
      activePage = 1;
      historyPage = 1;
      loadAlerts();
      loadHistory();
    }
  }

  function getSeverityClass(severity) {
    switch(severity?.toLowerCase()) {
      case 'critical': return 'severity-critical';
      case 'warning': return 'severity-warning';
      case 'high': return 'severity-high';
      default: return '';
    }
  }

  function getSeverityIcon(severity) {
    switch(severity?.toLowerCase()) {
      case 'critical': return '&#9888;&#65039;';
      case 'warning': return '&#9888;';
      case 'high': return '&#128276;';
      default: return '&#8505;';
    }
  }

  function formatAlertTime(ts) {
    if (!ts) return '-';
    const created = new Date(ts);
    const dd = String(created.getDate()).padStart(2, '0');
    const mm = String(created.getMonth() + 1).padStart(2, '0');
    const yy = String(created.getFullYear() % 100).padStart(2, '0');
    const hh = String(created.getHours()).padStart(2, '0');
    const mn = String(created.getMinutes()).padStart(2, '0');
    const ss = String(created.getSeconds()).padStart(2, '0');
    return `${dd}:${mm}:${yy} ${hh}:${mn}:${ss}`;
  }

  function requestNotificationPermission() {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }
  }

  async function exportAllActiveAlerts() {
    exporting = true;
    try {
      let allData = [];
      let page = 1;
      while (true) {
        const res = await fetch(`/api/alerts?page=${page}&page_size=50&status=active`);
        if (!res.ok) break;
        const data = await res.json();
        const items = Array.isArray(data.data) ? data.data : [];
        if (items.length === 0) break;
        allData = [...allData, ...items];
        if (page >= (data.total_pages || 1)) break;
        page++;
      }
      if (allData.length === 0) allData = allAlerts;
      await generateAndDownloadCSV(allData, 'active_alerts');
    } catch (e) {
      console.error('Export failed:', e);
    } finally {
      exporting = false;
    }
  }

  async function exportAllHistoryAlerts() {
    exporting = true;
    try {
      let allData = [];
      let page = 1;
      while (true) {
        const res = await fetch(`/api/alerts/history?page=${page}&page_size=50`);
        if (!res.ok) break;
        const data = await res.json();
        const items = Array.isArray(data.data) ? data.data : [];
        if (items.length === 0) break;
        allData = [...allData, ...items];
        if (page >= (data.total_pages || 1)) break;
        page++;
      }
      if (allData.length === 0) allData = historyAlerts;
      await generateAndDownloadCSV(allData, 'alert_history');
    } catch (e) {
      console.error('Export failed:', e);
    } finally {
      exporting = false;
    }
  }

  async function generateAndDownloadCSV(alerts, filename) {
    if (!alerts.length) return;
    const headers = ['Type', 'Severity', 'Device ID', 'Message', 'Status', 'Created At', 'Resolved At'];
    const rows = alerts.map(a => [
      a.type, a.severity, a.device_id, a.message, a.status, a.created_at, a.resolved_at || ''
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
    URL.revokeObjectURL(url);
  }

  window.addEventListener('intecs:alert', () => {
    loadAlerts();
    loadHistory();
    loadSeverityCounts();
  });

  window.addEventListener('alert_updated', () => {
    loadAlerts();
    loadHistory();
    loadSeverityCounts();
  });

  onMount(async () => {
    loading = true;
    error = null;
    await Promise.all([loadAlerts(), loadHistory(), loadSeverityCounts()]);
    loading = false;
    requestNotificationPermission();
  });

  setInterval(() => loadAlerts(), 10000);
  setInterval(() => loadHistory(), 60000);
  setInterval(() => loadSeverityCounts(), 30000);
</script>

<div class="alerts-section">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button class="tab-btn {activeTab === 'active' ? 'active' : ''}" onclick={() => switchTab('active')}>
        Active ({activeCount})
      </button>
      <button class="tab-btn {activeTab === 'history' ? 'active' : ''}" onclick={() => switchTab('history')}>
        History ({historyCount})
      </button>
    </div>
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button class="export-btn-small" onclick={activeTab === 'active' ? exportAllActiveAlerts : exportAllHistoryAlerts} disabled={exporting}>
        {#if exporting}&#x21bb; Downloading...{:else}&#128196; Export All{/if}
      </button>
      <span class="alert-count">{currentCount} alert{currentCount !== 1 ? 's' : ''}</span>
    </div>
  </div>

  <div class="filter-chips-row">
    <div class="filter-label">Severity:</div>
    {#each ['critical', 'warning'] as sev}
      <button class="filter-chip severity-{sev} {isActive(sev) ? 'active' : ''}" onclick={() => toggleFilter(sev)}>
        {sev.charAt(0).toUpperCase() + sev.slice(1)}
        <span class="chip-count">{activeTab === 'active' ? (sev === 'critical' ? activeSeverityTotals.critical : activeSeverityTotals.warning) : (sev === 'critical' ? historySeverityTotals.critical : historySeverityTotals.warning)}</span>
      </button>
    {/each}
    <div class="filter-label" style="margin-left: 0.5rem; margin-right: -0.2rem;">Status:</div>
    <button class="filter-chip filter-status {isActive('acknowledged') ? 'active' : ''}" onclick={() => toggleFilter('acknowledged')}>
      Acknowledged
      <span class="chip-count">
        {activeTab === 'active' ? (() => allAlerts.filter(a => a.status === 'acknowledged').length)() : (historySeverityTotals.acknowledged || 0)}
      </span>
    </button>
    {#if getVisibleFilters().length > 0}
      <button class="clear-filters-btn" onclick={resetCurrentFilters}>
        Clear Filters
      </button>
    {/if}
  </div>

  {#if loading && visibleAlerts.length === 0}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading alerts...</span>
    </div>
  {:else if visibleAlerts.length === 0 && !hasActiveFilters()}
    <div class="no-data">
      {activeTab === 'active' ? 'No active alerts' : 'No alert history'}
    </div>
  {:else if visibleAlerts.length === 0}
    <div class="no-data">
      No alerts match the selected filters
    </div>
  {:else}
    <div class="alerts-list">
      {#each visibleAlerts as alert}
        <div class="alert-item {getSeverityClass(alert.severity)} status-{alert.status}">
          <div class="alert-content">
            <div class="alert-top">
              {#if alert.status === 'solved'}
                <span class="alert-badge solved">
                  &#10003; SOLVED
                </span>
              {:else}
                <span class="alert-badge {getSeverityClass(alert.severity)}">
                  <span class="badge-icon" innerHTML={getSeverityIcon(alert.severity)}></span>
                  {alert.severity?.toUpperCase() || 'INFO'}
                </span>
              {/if}
              <span class="alert-type">{alert.type}</span>
              <span class="alert-device">{alert.device_id}</span>
            </div>
            <p class="alert-message">{alert.message}</p>
            {#if alert.status === 'acknowledged' && alert.acknowledged_by}
              <div class="audit-trail">
                Acknowledged by <strong>{alert.acknowledged_by}</strong> — {formatAlertTime(alert.created_at)}
              </div>
            {/if}
            {#if alert.status === 'solved'}
              <div class="audit-trail audit-solved">
                <span>&#10003;</span> Solved by <strong>{alert.solved_by}</strong> — {formatAlertTime(alert.resolved_at)}
              </div>
              {#if alert.solve_notes}
                <div class="solve-notes">&#8220; {alert.solve_notes} &#8221;</div>
              {/if}
            {/if}
            <div class="alert-footer">
              <span class="alert-time" title={alert.created_at}>{formatAlertTime(alert.created_at)}</span>
              {#if alert.status === 'active'}
                <button class="ack-btn" onclick={() => handleAcknowledge(alert.id)}>Acknowledge</button>
                <button class="solve-btn" onclick={() => openSolveModal(alert)}>Solve</button>
              {:else if alert.status === 'acknowledged'}
                <button class="solve-btn" onclick={() => openSolveModal(alert)}>Solve</button>
              {:else}
                <span class="resolved-badge" title={alert.resolved_at}>Resolved {formatAlertTime(alert.resolved_at)}</span>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>

    <div class="pagination-container">
      <div class="pagination-info">
        <span class="info-text">
          Showing {((activeTab === 'active' ? activePage : historyPage) - 1) * pageSize + 1}&ndash;{Math.min((activeTab === 'active' ? activePage : historyPage) * pageSize, activeCount + historyCount)} of {activeTab === 'active' ? activeCount : historyCount} alerts
        </span>
        <select class="page-size-select" value={String(pageSize)} onchange={handlePageSizeChange}>
          <option value="5">5 per page</option>
          <option value="10">10 per page</option>
          <option value="15">15 per page</option>
        </select>
      </div>

      <nav class="pagination-nav">
        <button class="page-btn" disabled={isVisibleFirstPage} onclick={() => changePage(activePage - 1)}>
          &laquo; Prev
        </button>

        {#each getPageNumbers(activeTab === 'active' ? activeTotalPages : historyTotalPages, activeTab === 'active' ? activePage : historyPage) as page}
          {#if page === '...'}
            <span class="ellipsis">...</span>
          {:else}
            <button class="page-btn {page === (activeTab === 'active' ? activePage : historyPage) ? 'active' : ''}" onclick={() => changePage(page)}>
              {page}
            </button>
          {/if}
        {/each}

        <button class="page-btn" disabled={(activeTab === 'active' ? activePage : historyPage) >= (activeTab === 'active' ? activeTotalPages : historyTotalPages)} onclick={() => changePage(activePage + 1)}>
          Next &raquo;
        </button>
      </nav>
    </div>
  {/if}

  {#if error}
    <div class="error-message">&#9888; {error}</div>
  {/if}
</div>

{#if showModal}
  <div class="modal-overlay" onclick={closeModal}></div>
  <div class="modal">
    <div class="modal-header">
      <h3>Resolve Alert</h3>
      <button class="modal-close" onclick={closeModal}>&times;</button>
    </div>
    <div class="modal-body">
      <div class="modal-alert-info">
        <span class="modal-alert-type">{selectedSolveAlert?.type}</span>
        <span class="modal-alert-device">{selectedSolveAlert?.device_id}</span>
        <span class="modal-alert-msg">{selectedSolveAlert?.message}</span>
      </div>
      <label class="modal-label">
        Resolution Notes <span class="required">*</span>
        <textarea
          bind:value={solveNotes}
          placeholder="Contoh: Sensor diganti, suhu sudah normal kembali"
          rows="4"
        ></textarea>
      </label>
    </div>
    <div class="modal-footer">
      <button class="modal-cancel" onclick={closeModal}>Cancel</button>
      <button class="modal-confirm" onclick={handleSolve} disabled={!solveNotes.trim()}>
        Confirm Resolve
      </button>
    </div>
  </div>
{/if}

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

  .export-btn-small:hover:not(:disabled) {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-color: var(--accent-blue);
  }

  .export-btn-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .filter-chips-row {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.85rem 1.25rem;
    border-bottom: 1px solid var(--border-light);
    flex-wrap: wrap;
    background: var(--bg-tertiary);
  }

  .filter-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .filter-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.25rem 0.75rem;
    border-radius: 100px;
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    cursor: pointer;
    transition: all 0.2s ease;
    border: 2px solid transparent;
    position: relative;
    background: #e2e8f0;
    color: #475569;
    border-color: rgba(100, 116, 139, 0.2);
  }

  .filter-chip.filter-status.active {
    background: #3b82f6;
    color: white;
    box-shadow: 0 0 0 1px #3b82f6;
  }

  .filter-chip.severity-critical {
    background: #fee2e2;
    color: #dc2626;
    border-color: rgba(220, 38, 38, 0.2);
  }

  .filter-chip.severity-critical.active {
    background: #dc2626;
    color: white;
    box-shadow: 0 0 0 1px #dc2626;
  }

  .filter-chip.severity-warning {
    background: #fef3c7;
    color: #d97706;
    border-color: rgba(217, 119, 6, 0.2);
  }

  .filter-chip.severity-warning.active {
    background: #d97706;
    color: white;
    box-shadow: 0 0 0 1px #d97706;
  }

  .filter-chip:hover:not(.active) {
    opacity: 0.8;
    transform: translateY(-1px);
  }

  .chip-count {
    background: rgba(0, 0, 0, 0.1);
    padding: 0.1rem 0.4rem;
    border-radius: 100px;
    font-size: 0.65rem;
    font-weight: 600;
  }

  .filter-chip.active .chip-count {
    background: rgba(255, 255, 255, 0.25);
  }

  .clear-filters-btn {
    padding: 0.2rem 0.6rem;
    border-radius: 100px;
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--text-muted);
    background: transparent;
    border: 1px dashed var(--border-light);
    cursor: pointer;
    transition: all 0.2s ease;
    margin-left: auto;
  }

  .clear-filters-btn:hover {
    color: var(--danger-text);
    border-color: var(--danger-text);
    background: var(--danger-bg);
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
    align-items: flex-start;
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

  .alert-item.status-acknowledged {
    border-left-color: #3b82f6;
    background: linear-gradient(90deg, rgba(59, 130, 246, 0.05), transparent);
  }

  .alert-item.status-solved {
    border-left-color: #22c55e;
    background: linear-gradient(90deg, rgba(34, 197, 94, 0.05), transparent);
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

  .alert-badge.solved {
    background: #dcfce7;
    color: #16a34a;
    border: 1px solid #bbf7d0;
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

  .audit-trail {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-top: 0.25rem;
    padding: 0.25rem 0.5rem;
    background: rgba(0, 0, 0, 0.02);
    border-radius: 4px;
  }

  .audit-solved {
    color: #16a34a;
    background: rgba(34, 197, 94, 0.08);
  }

  .solve-notes {
    font-size: 0.8rem;
    color: #475569;
    margin-top: 0.35rem;
    padding: 0.4rem 0.6rem;
    background: rgba(34, 197, 94, 0.06);
    border-left: 3px solid #22c55e;
    border-radius: 0 4px 4px 0;
    font-style: italic;
  }

  .alert-footer {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.5rem;
    flex-wrap: wrap;
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

  .solve-btn {
    background: #16a34a;
    color: white;
    border: none;
    padding: 0.35rem 0.75rem;
    border-radius: var(--radius-sm);
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
    transition: background 0.15s ease;
  }

  .solve-btn:hover {
    background: #15803d;
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

  /* Modal Styles */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 1000;
    backdrop-filter: blur(2px);
  }

  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: var(--bg-primary);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-light);
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
    z-index: 1001;
    width: 90%;
    max-width: 500px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border-light);
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.1rem;
    color: var(--text-primary);
    font-weight: 600;
  }

  .modal-close {
    background: none;
    border: none;
    font-size: 1.5rem;
    color: var(--text-muted);
    cursor: pointer;
    padding: 0.25rem;
    line-height: 1;
  }

  .modal-close:hover {
    color: var(--text-primary);
  }

  .modal-body {
    padding: 1.5rem;
  }

  .modal-alert-info {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-light);
  }

  .modal-alert-type {
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    background: var(--bg-tertiary);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
  }

  .modal-alert-device {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-family: ui-monospace, monospace;
    background: var(--bg-tertiary);
    padding: 0.25rem 0.5rem;
    border-radius: 4px;
  }

  .modal-alert-msg {
    font-size: 0.85rem;
    color: var(--text-secondary);
    width: 100%;
    margin-top: 0.25rem;
  }

  .modal-label {
    display: block;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 0.5rem;
  }

  .modal-label .required {
    color: #ef4444;
  }

  .modal-label textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--border-light);
    border-radius: var(--radius-md);
    font-size: 0.9rem;
    font-family: inherit;
    line-height: 1.5;
    resize: vertical;
    background: var(--bg-secondary);
    color: var(--text-primary);
    margin-top: 0.5rem;
  }

  .modal-label textarea:focus {
    outline: none;
    border-color: var(--accent-blue);
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
  }

  .modal-label textarea:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-top: 1px solid var(--border-light);
  }

  .modal-cancel {
    padding: 0.5rem 1rem;
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border: 1px solid var(--border-light);
    border-radius: var(--radius-md);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .modal-cancel:hover {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  .modal-confirm {
    padding: 0.5rem 1.25rem;
    background: #16a34a;
    color: white;
    border: none;
    border-radius: var(--radius-md);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .modal-confirm:hover:not(:disabled) {
    background: #15803d;
  }

  .modal-confirm:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  @media (max-width: 640px) {
    .alert-item {
      flex-direction: column;
      align-items: flex-start;
      gap: 0.75rem;
    }

    .ack-btn, .solve-btn {
      width: auto;
    }

    .pagination-container {
      flex-direction: column;
      align-items: stretch;
    }

    .pagination-nav {
      justify-content: center;
      flex-wrap: wrap;
    }

    .modal {
      width: 95%;
    }
  }
</style>
