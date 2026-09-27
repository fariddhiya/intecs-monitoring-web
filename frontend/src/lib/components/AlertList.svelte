<script>
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();
  let activeTab = $state('active');
  let pageSize = $state(5);
  let loading = $state(true);
  let error = $state(null);
  let exporting = $state(false);

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

  // Store data per tab as objects keyed by tab name
  let tabsData = $state({
    active: { data: [], page: 1, totalPages: 1, totalItems: 0 },
    acknowledged: { data: [], page: 1, totalPages: 1, totalItems: 0 },
    solved: { data: [], page: 1, totalPages: 1, totalItems: 0 }
  });

  let severityCounts = $state({ active: {}, acknowledged: {}, solved: {} });

  let filterSeverities = $state({ active: [], acknowledged: [], solved: [] });

  function loadSeverityCounts() {
    fetch('/api/alerts/severity-counts')
      .then(r => r.json())
      .then(data => {
        severityCounts = data;
      })
      .catch(e => console.error('Failed to load severity counts:', e));
  }

  function getFilteredAlerts(alertList) {
    const filters = filterSeverities[activeTab] || [];
    if (!filters.length) return alertList;
    return alertList.filter(a => {
      for (const f of filters) {
        if (a.severity?.toLowerCase() === f) return true;
      }
      return false;
    });
  }

  function toggleFilter(filter) {
    const filters = filterSeverities[activeTab];
    const idx = filters.indexOf(filter);
    
    if (idx >= 0) {
      filters.splice(idx, 1);
      filterSeverities[activeTab] = [...filters];
    } else {
      filterSeverities[activeTab] = [...filters, filter];
    }
  }

  function isActive(filter) {
    return (filterSeverities[activeTab] || []).includes(filter);
  }

  function clearFilters() {
    filterSeverities[activeTab] = [];
  }

  function hasActiveFilters() {
    return !!(filterSeverities[activeTab] || []).length;
  }

  let visibleAlerts = $derived(getFilteredAlerts(tabsData[activeTab]?.data || []));
  let currentCount = $derived(visibleAlerts?.length || 0);
  let currentPage = $derived(tabsData[activeTab]?.page || 1);
  let currentTotalPages = $derived(tabsData[activeTab]?.totalPages || 1);
  let currentTotalItems = $derived(tabsData[activeTab]?.totalItems || 0);
  let isVisibleFirstPage = $derived(currentPage <= 1);

  let activeSeverityTotals = $derived({
    critical: severityCounts.active?.critical || 0,
    warning: severityCounts.active?.warning || 0,
  });

  let acknowledgedSeverityTotals = $derived({
    critical: severityCounts.acknowledged?.critical || 0,
    warning: severityCounts.acknowledged?.warning || 0,
  });

  let solvedSeverityTotals = $derived({
    critical: severityCounts.solved?.critical || 0,
    warning: severityCounts.solved?.warning || 0,
  });

  let activeCount = $derived(tabsData.active?.totalItems || 0);
  let acknowledgedCount = $derived(tabsData.acknowledged?.totalItems || 0);
  let solvedCount = $derived(tabsData.solved?.totalItems || 0);

  let severitiesDisplay = $derived({
    active: activeSeverityTotals,
    acknowledged: acknowledgedSeverityTotals,
    solved: solvedSeverityTotals
  });

  function buildFetchUrl(status) {
    if (status === 'active') {
      return `/api/alerts?page=${tabsData.active.page}&page_size=${pageSize}&status=active`;
    }
    if (status === 'acknowledged') {
      return `/api/alerts?page=${tabsData.acknowledged.page}&page_size=${pageSize}&status=acknowledged`;
    }
    if (status === 'solved') {
      return `/api/alerts?page=${tabsData.solved.page}&page_size=${pageSize}&status=solved`;
    }
    return '';
  }

  async function loadTab(tabName) {
    try {
      const statusMap = { active: 'active', acknowledged: 'acknowledged', solved: 'solved' };
      const url = buildFetchUrl(statusMap[tabName]);
      const res = await fetch(url);
      if (res.ok) {
        const json = await res.json();
        tabsData[tabName] = {
          data: Array.isArray(json.data) ? json.data : json,
          page: tabsData[tabName].page,
          totalItems: json.total_items || 0,
          totalPages: json.total_pages || 1
        };
      } else {
        if (!error) error = `Failed to load ${tabName} alerts`;
      }
    } catch (e) {
      if (!error) error = e.message;
    }
  }

  async function loadAllTabs() {
    loading = true;
    await Promise.all([
      loadTab('active'),
      loadTab('acknowledged'),
      loadTab('solved'),
      loadSeverityCounts()
    ]);
    loading = false;
  }

  async function refreshAll() {
    await Promise.all([
      loadTab('active'),
      loadTab('acknowledged'),
      loadTab('solved'),
      loadSeverityCounts()
    ]);
  }

  async function handleAcknowledge(id) {
    try {
      await fetch(`/api/alerts/${id}/acknowledge`, { method: 'POST', headers: { 'X-User-Email': currentUser.email } });
      dispatch('acknowledge', id);
      await refreshAll();
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
      await refreshAll();
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
    loadTab(tab);
  }

  function resetCurrentFilters() {
    filterSeverities[activeTab] = [];
  }

  async function changePage(page) {
    if (page < 1 || page > currentTotalPages) return;
    tabsData[activeTab] = { ...tabsData[activeTab], page };
    loadTab(activeTab);
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
      for (const tab in tabsData) {
        tabsData[tab] = { ...tabsData[tab], page: 1 };
      }
      loadAllTabs();
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

  async function exportCurrentTabAlerts() {
    exporting = true;
    try {
      const statusMap = { active: 'active', acknowledged: 'acknowledged', solved: 'solved' };
      const suffixMap = { active: 'active_alerts', acknowledged: 'acknowledged_alerts', solved: 'solved_alerts' };
      let allData = [];
      let page = 1;
      while (true) {
        const status = statusMap[activeTab];
        const res = await fetch(`/api/alerts?page=${page}&page_size=50&status=${status}`);
        if (!res.ok) break;
        const data = await res.json();
        const items = Array.isArray(data.data) ? data.data : [];
        if (items.length === 0) break;
        allData = [...allData, ...items];
        if (page >= (data.total_pages || 1)) break;
        page++;
      }
      if (allData.length === 0) allData = tabsData[activeTab].data;
      await generateAndDownloadCSV(allData, suffixMap[activeTab]);
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
    refreshAll();
  });

  window.addEventListener('alert_updated', () => {
    refreshAll();
  });

  onMount(async () => {
    loading = true;
    error = null;
    await loadAllTabs();
    requestNotificationPermission();
  });

  setInterval(() => refreshAll(), 10000);
</script>

<div class="alerts-section">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button class="tab-btn {activeTab === 'active' ? 'active' : ''}" onclick={() => switchTab('active')}>
        Active ({activeCount})
      </button>
      <button class="tab-btn {activeTab === 'acknowledged' ? 'active' : ''}" onclick={() => switchTab('acknowledged')}>
        Acknowledged ({acknowledgedCount})
      </button>
      <button class="tab-btn {activeTab === 'solved' ? 'active' : ''}" onclick={() => switchTab('solved')}>
        Solved ({solvedCount})
      </button>
    </div>
    <div style="display: flex; gap: 0.5rem; align-items: center;">
      <button class="export-btn-small" onclick={exportCurrentTabAlerts} disabled={exporting}>
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
        <span class="chip-count">{severitiesDisplay[activeTab][sev] || 0}</span>
      </button>
    {/each}
    {#if hasActiveFilters()}
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
      {activeTab === 'active' ? 'No active alerts' : activeTab === 'acknowledged' ? 'No acknowledged alerts' : 'No solved alerts'}
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
          Showing {((currentPage - 1) * pageSize + 1)}–{Math.min(currentPage * pageSize, currentTotalItems)} of {currentTotalItems} alerts
        </span>
        <select class="page-size-select" value={String(pageSize)} onchange={handlePageSizeChange}>
          <option value="5">5 per page</option>
          <option value="10">10 per page</option>
          <option value="15">15 per page</option>
        </select>
      </div>

      <nav class="pagination-nav">
        <button class="page-btn" disabled={isVisibleFirstPage} onclick={() => changePage(currentPage - 1)}>
          &laquo; Prev
        </button>

        {#each getPageNumbers(currentTotalPages, currentPage) as page}
          {#if page === '...'}
            <span class="ellipsis">...</span>
          {:else}
            <button class="page-btn {page === currentPage ? 'active' : ''}" onclick={() => changePage(page)}>
              {page}
            </button>
          {/if}
        {/each}

        <button class="page-btn" disabled={currentPage >= currentTotalPages} onclick={() => changePage(currentPage + 1)}>
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
  /* ── Main wrapper ─────────────────────────────────────── */
  .alerts-section {
    background: var(--bg-secondary);
    border-radius: 16px;
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-card);
    overflow: hidden;
  }

  /* ── Tabs ──────────────────────────────────────────────── */
  .tab-btn {
    background: transparent;
    border: none;
    padding: 0.6rem 1.25rem;
    font-size: 0.85rem;
    color: var(--text-muted);
    cursor: pointer;
    border-radius: 10px;
    transition: all 0.2s ease;
    font-weight: 500;
    white-space: nowrap;
  }

  .tab-btn:hover:not(.active) {
    color: var(--text-secondary);
    background: var(--bg-tertiary);
  }

  .tab-btn.active {
    background: var(--accent-blue);
    color: #ffffff;
    font-weight: 600;
    box-shadow: 0 0 12px rgba(96, 165, 250, 0.2);
  }

  /* ── Export button ─────────────────────────────────────── */
  .export-btn-small {
    background: transparent;
    border: 1px solid var(--border-medium);
    color: var(--text-secondary);
    padding: 0.45rem 0.85rem;
    border-radius: 8px;
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 0.2s ease;
    font-weight: 500;
  }

  .export-btn-small:hover:not(:disabled) {
    background: var(--bg-elevated);
    color: var(--text-primary);
    border-color: var(--accent-blue);
    box-shadow: 0 0 0 1px var(--accent-blue);
  }

  .export-btn-small:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* ── Filter chips row ──────────────────────────────────── */
  .filter-chips-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.5rem;
    border-bottom: 1px solid var(--border-light);
    flex-wrap: wrap;
    background: var(--bg-tertiary);
  }

  .filter-label {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .filter-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.35rem 0.85rem;
    border-radius: 100px;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    cursor: pointer;
    transition: all 0.2s ease;
    border: none;
    position: relative;
  }

  /* Light-mode defaults (kept for light theme) */
  .filter-chip.severity-critical {
    background: #fee2e2;
    color: #dc2626;
  }
  .filter-chip.severity-critical.active {
    background: #dc2626;
    color: white;
  }

  .filter-chip.severity-warning {
    background: #fef3c7;
    color: #d97706;
  }
  .filter-chip.severity-warning.active {
    background: #d97706;
    color: white;
  }

  .filter-chip:hover:not(.active) {
    transform: translateY(-1px);
    filter: brightness(0.95);
  }

  .chip-count {
    background: rgba(0, 0, 0, 0.08);
    padding: 0.05rem 0.45rem;
    border-radius: 100px;
    font-size: 0.65rem;
    font-weight: 600;
  }

  .filter-chip.active .chip-count {
    background: rgba(255, 255, 255, 0.2);
  }

  .clear-filters-btn {
    padding: 0.25rem 0.75rem;
    border-radius: 100px;
    font-size: 0.7rem;
    font-weight: 600;
    color: var(--text-muted);
    background: transparent;
    border: 1px dashed var(--border-medium);
    cursor: pointer;
    transition: all 0.2s ease;
    margin-left: auto;
  }

  .clear-filters-btn:hover {
    color: var(--danger-text);
    border-color: var(--danger-text);
    background: var(--danger-bg);
  }

  /* ── Alert count badge ─────────────────────────────────── */
  .alert-count {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-weight: 500;
    background: var(--bg-tertiary);
    padding: 0.25rem 0.85rem;
    border-radius: 100px;
    border: 1px solid var(--border-light);
  }

  /* ── Alerts list ───────────────────────────────────────── */
  .alerts-list {
    display: flex;
    flex-direction: column;
    max-height: 600px;
    overflow-y: auto;
  }

  /* ── Alert card ────────────────────────────────────────── */
  .alert-item {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border-light);
    transition: background-color 0.15s ease;
    position: relative;
  }

  .alert-item:hover {
    background: var(--bg-tertiary);
  }

  .alert-item:last-child {
    border-bottom: none;
  }

  /* Left accent bar — thickened with inner radius */
  .alert-item::before {
    content: '';
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 4px;
    border-radius: 0 4px 4px 0;
  }

  /* Severity accent bars */
  .alert-item.severity-critical::before {
    background: linear-gradient(180deg, #ef4444 0%, #dc2626 100%);
    box-shadow: 0 0 12px rgba(239, 68, 68, 0.2);
  }
  .alert-item.severity-critical {
    background: linear-gradient(90deg, rgba(239, 68, 68, 0.04), transparent);
  }

  .alert-item.severity-warning::before {
    background: linear-gradient(180deg, #f59e0b 0%, #d97706 100%);
    box-shadow: 0 0 12px rgba(245, 158, 11, 0.15);
  }
  .alert-item.severity-warning {
    background: linear-gradient(90deg, rgba(245, 158, 11, 0.04), transparent);
  }

  .alert-item.severity-high::before {
    background: linear-gradient(180deg, #f97316 0%, #ea580c 100%);
  }

  .alert-item.status-acknowledged::before {
    background: linear-gradient(180deg, #3b82f6 0%, #2563eb 100%);
    box-shadow: 0 0 12px rgba(59, 130, 246, 0.15);
  }
  .alert-item.status-acknowledged {
    background: linear-gradient(90deg, rgba(59, 130, 246, 0.04), transparent);
  }

  .alert-item.status-solved::before {
    background: linear-gradient(180deg, #22c55e 0%, #16a34a 100%);
    box-shadow: 0 0 12px rgba(34, 197, 94, 0.15);
  }
  .alert-item.status-solved {
    background: linear-gradient(90deg, rgba(34, 197, 94, 0.04), transparent);
  }

  /* ── Alert header row ──────────────────────────────────── */
  .alert-top {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 0.35rem;
    flex-wrap: wrap;
  }

  /* ── Badge pills — modern chip style ───────────────────── */
  .alert-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.2rem 0.7rem;
    border-radius: 100px;
    font-size: 0.68rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border: none;
  }

  /* Critical — red pill with glow on severity critical */
  .alert-badge.severity-critical {
    background: rgba(248, 113, 113, 0.18);
    color: #fca5a5;
    box-shadow: 0 0 8px rgba(248, 113, 113, 0.15);
  }

  /* Warning — amber pill */
  .alert-badge.severity-warning {
    background: rgba(251, 191, 36, 0.18);
    color: #fcd34d;
  }

  /* High — orange pill */
  .alert-badge.severity-high {
    background: rgba(249, 115, 22, 0.18);
    color: #fdba74;
  }

  /* Solved — green pill */
  .alert-badge.solved {
    background: rgba(52, 211, 153, 0.18);
    color: #6ee7b7;
  }

  .alert-type {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .alert-device {
    font-size: 0.78rem;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
    background: var(--bg-tertiary);
    padding: 0.15rem 0.55rem;
    border-radius: 6px;
    border: 1px solid var(--border-light);
  }

  /* ── Alert message ─────────────────────────────────────── */
  .alert-message {
    font-size: 0.88rem;
    color: var(--text-secondary);
    line-height: 1.55;
    margin: 0.3rem 0;
  }

  /* ── Audit trail / solved info ─────────────────────────── */
  .audit-trail {
    font-size: 0.78rem;
    color: var(--text-muted);
    margin-top: 0.3rem;
    padding: 0.35rem 0.65rem;
    background: rgba(255, 255, 255, 0.02);
    border-radius: 6px;
    border: 1px solid var(--border-light);
  }

  .audit-solved {
    color: var(--success-text);
    background: rgba(52, 211, 153, 0.06);
    border-color: rgba(52, 211, 153, 0.12);
  }

  .solve-notes {
    font-size: 0.78rem;
    color: var(--text-secondary);
    margin-top: 0.4rem;
    padding: 0.5rem 0.75rem;
    background: rgba(52, 211, 153, 0.06);
    border-left: 3px solid var(--success);
    border-radius: 0 8px 8px 0;
    font-style: italic;
  }

  /* ── Alert footer ──────────────────────────────────────── */
  .alert-footer {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.65rem;
    flex-wrap: wrap;
  }

  .alert-time {
    font-size: 0.78rem;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
  }

  /* ── Action buttons ────────────────────────────────────── */
  .ack-btn {
    background: var(--accent-blue);
    color: #ffffff;
    border: none;
    padding: 0.4rem 0.85rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.78rem;
    font-weight: 500;
    transition: all 0.15s ease;
  }

  .ack-btn:hover {
    background: var(--accent-blue-dark);
    box-shadow: 0 0 0 1px var(--accent-blue);
  }

  .solve-btn {
    background: rgba(52, 211, 153, 0.18);
    color: var(--success-text);
    border: 1px solid rgba(52, 211, 153, 0.25);
    padding: 0.4rem 0.85rem;
    border-radius: 8px;
    cursor: pointer;
    font-size: 0.78rem;
    font-weight: 500;
    transition: all 0.15s ease;
  }

  .solve-btn:hover {
    background: rgba(52, 211, 153, 0.28);
    border-color: var(--success);
  }

  .resolved-badge {
    font-size: 0.78rem;
    color: var(--success-text);
    background: var(--success-bg);
    padding: 0.3rem 0.75rem;
    border-radius: 8px;
    border: 1px solid rgba(52, 211, 153, 0.15);
  }

  .badge-icon {
    font-style: normal;
  }

  /* ── Loading / empty states ────────────────────────────── */
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
    padding: 1rem 1.5rem;
    color: var(--danger-text);
    background: var(--danger-bg);
    font-size: 0.88rem;
    border-top: 1px solid var(--border-light);
  }

  /* ── Pagination ────────────────────────────────────────── */
  .pagination-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
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
    font-size: 0.82rem;
    color: var(--text-muted);
  }

  .page-size-select {
    padding: 0.4rem 2rem 0.4rem 0.75rem;
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    font-size: 0.8rem;
    color: var(--text-primary);
    background: var(--bg-tertiary) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 10 10'%3E%3Cpath fill='%238b95a8' d='M5 7L1 3h8z'/%3E%3C/svg%3E") no-repeat right 0.5rem center;
    cursor: pointer;
    appearance: none;
    transition: border-color 0.2s ease;
  }

  .page-size-select:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .pagination-nav {
    display: flex;
    align-items: center;
    gap: 0.3rem;
  }

  .page-btn {
    padding: 0.45rem 0.85rem;
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.82rem;
    cursor: pointer;
    transition: all 0.2s ease;
    min-width: 38px;
    font-weight: 500;
  }

  .page-btn:hover:not(:disabled):not(.active) {
    background: var(--bg-tertiary);
    border-color: var(--accent-blue);
    color: var(--text-primary);
  }

  .page-btn.active {
    background: var(--accent-blue);
    border-color: var(--accent-blue);
    color: #ffffff;
    font-weight: 600;
    box-shadow: 0 0 8px rgba(96, 165, 250, 0.2);
  }

  .page-btn:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .ellipsis {
    padding: 0.4rem 0.3rem;
    color: var(--text-muted);
    user-select: none;
  }

  /* ── Modal ─────────────────────────────────────────────── */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 1000;
    backdrop-filter: blur(4px);
  }

  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: var(--bg-elevated);
    border-radius: 16px;
    border: 1px solid var(--border-medium);
    box-shadow: 0 25px 60px -12px rgba(0, 0, 0, 0.5);
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
    font-size: 1.05rem;
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
    transition: color 0.15s ease;
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
    font-size: 0.82rem;
    font-weight: 600;
    color: var(--text-primary);
    background: var(--bg-tertiary);
    padding: 0.3rem 0.6rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
  }

  .modal-alert-device {
    font-size: 0.78rem;
    color: var(--text-muted);
    font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
    background: var(--bg-tertiary);
    padding: 0.3rem 0.6rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
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
    color: var(--danger);
  }

  .modal-label textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--border-medium);
    border-radius: 10px;
    font-size: 0.9rem;
    font-family: inherit;
    line-height: 1.5;
    resize: vertical;
    background: var(--bg-tertiary);
    color: var(--text-primary);
    margin-top: 0.5rem;
    transition: border-color 0.2s ease;
  }

  .modal-label textarea:focus {
    outline: none;
    border-color: var(--accent-blue);
    box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.15);
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
    padding: 0.5rem 1.15rem;
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border: 1px solid var(--border-medium);
    border-radius: 10px;
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .modal-cancel:hover {
    background: var(--bg-elevated);
    color: var(--text-primary);
  }

  .modal-confirm {
    padding: 0.5rem 1.25rem;
    background: var(--success);
    color: #ffffff;
    border: none;
    border-radius: 10px;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .modal-confirm:hover:not(:disabled) {
    filter: brightness(1.1);
    box-shadow: 0 0 0 1px var(--success);
  }

  .modal-confirm:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* ── Responsive ────────────────────────────────────────── */
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
