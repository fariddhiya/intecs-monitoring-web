<script>
  import { onMount } from 'svelte';
  
  let devices = $state([]);
  let allDevices = $state([]);
  let error = $state(null);
  let loading = $state(true);
  let searchQuery = $state('');
  let filterConnection = $state('');
  let filterEquipment = $state('');
  let sortColumn = $state('device_id');
  let sortDirection = $state('ASC');
  let currentPage = $state(1);
  let pageSize = $state(5);
  let totalPages = $state(1);
  let totalItems = $state(0);

  async function loadDevices() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (searchQuery) params.set('search', searchQuery);
      if (filterConnection) params.set('connection', filterConnection);
      if (filterEquipment) params.set('equipment_status', filterEquipment);
      if (sortColumn) params.set('sort', sortColumn);
      if (sortDirection) params.set('dir', sortDirection);
      params.set('page', currentPage);
      params.set('page_size', pageSize);
      
      const res = await fetch(`/api/devices?${params}`);
      if (res.ok) {
        const data = await res.json();
        if (data.data) {
          allDevices = data.data;
          devices = allDevices;
          totalPages = data.total_pages || 1;
          totalItems = data.total_items || 0;
        } else {
          allDevices = data;
          devices = allDevices;
          totalPages = 1;
          totalItems = data.length;
        }
        error = null;
      } else {
        error = 'Failed to load devices';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    applyFilters();
  });

  function applyFilters() {
    devices = [...allDevices];
    
    // Apply in-memory filters for connection status
    if (filterConnection) {
      devices = devices.filter(d => d.connection === filterConnection);
    }
    
    // Sort
    devices.sort((a, b) => {
      let valA = a[sortColumn] ?? '';
      let valB = b[sortColumn] ?? '';
      
      if (typeof valA === 'number' && typeof valB === 'number') {
        return sortDirection === 'ASC' ? valA - valB : valB - valA;
      }
      
      valA = String(valA).toLowerCase();
      valB = String(valB).toLowerCase();
      
      if (valA < valB) return sortDirection === 'ASC' ? -1 : 1;
      if (valA > valB) return sortDirection === 'ASC' ? 1 : -1;
      return 0;
    });
  }

  function handleSearch(e) {
    searchQuery = e.target.value;
    currentPage = 1;
    loadDevices();
  }

  function handleFilterChange(field, value) {
    switch(field) {
      case 'connection': filterConnection = value; break;
      case 'equipment': filterEquipment = value; break;
    }
    currentPage = 1;
    loadDevices();
  }

  function toggleSort(column) {
    if (sortColumn === column) {
      sortDirection = sortDirection === 'ASC' ? 'DESC' : 'ASC';
    } else {
      sortColumn = column;
      sortDirection = 'ASC';
    }
    currentPage = 1;
    loadDevices();
  }

  function getSortIcon(column) {
    if (sortColumn !== column) return '';
    return sortDirection === 'ASC' ? ' ↑' : ' ↓';
  }

  function getFuelColor(percent) {
    if (percent < 10) return '#ef4444';
    if (percent < 20) return '#f59e0b';
    if (percent < 50) return '#eab308';
    return '#10b981';
  }

  function getConnectionClass(conn) {
    switch(conn) {
      case 'ONLINE': return 'status-online';
      case 'STALE': return 'status-stale';
      default: return 'status-offline';
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

  function exportCSV() {
    const headers = ['Device ID', 'Name', 'Site', 'Fuel %', 'Temp °C', 'Flow L/min', 'Equipment', 'Connection', 'Last Seen'];
    const rows = allDevices.map(d => [
      d.device_id,
      d.name || '',
      d.site || '',
      d.fuel_percent.toFixed(1),
      d.temperature.toFixed(1),
      d.flow_rate.toFixed(1),
      d.equipment_status,
      d.connection,
      d.last_seen,
    ]);
    
    const csvContent = [headers.join(','), ...rows.map(r => r.map(v => `"${v}"`).join(','))].join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.setAttribute('href', url);
    link.setAttribute('download', `devices_${new Date().toISOString().split('T')[0]}.csv`);
    link.style.display = 'none';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }

  function changePage(page) {
    if (page >= 1 && page <= totalPages) {
      currentPage = page;
      loadDevices();
      window.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }

  function getPageNumbers() {
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
    pageSize = Number(e.target.value);
    currentPage = 1;
    loadDevices();
  }

  onMount(() => {
    loadDevices();
  });
</script>

<div class="table-container">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; gap: 1rem; flex-wrap: wrap;">
    <h2>Device Status</h2>
    <button class="export-btn" on:click={exportCSV} title="Export to CSV">
      &#128196; Export CSV
    </button>
  </div>
  
  <div class="filters-bar">
    <div class="search-box">
      <span class="search-icon">&#128269;</span>
      <input 
        type="text" 
        placeholder="Search by device ID..." 
        value={searchQuery}
        on:input={handleSearch}
      />
    </div>
    
    <select 
      class="filter-select" 
      value={filterConnection}
      on:change={(e) => handleFilterChange('connection', e.target.value)}>
      <option value="">All Connections</option>
      <option value="ONLINE">Online</option>
      <option value="STALE">Stale</option>
      <option value="OFFLINE">Offline</option>
    </select>
    
    <select 
      class="filter-select" 
      value={filterEquipment}
      on:change={(e) => handleFilterChange('equipment', e.target.value)}>
      <option value="">All Equipment</option>
      <option value="running">Running</option>
      <option value="idle">Idle</option>
      <option value="maintenance">Maintenance</option>
    </select>
  </div>
  
  {#if loading}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading devices...</span>
    </div>
  {:else if error}
    <div class="error-message" style="margin: 1rem;">&#9888; {error}</div>
  {:else if devices.length === 0}
    <div class="empty-state">
      <div class="empty-icon">&#128247;</div>
      <h3 class="empty-title">{searchQuery ? 'No devices matching your search' : 'No Devices Found'}</h3>
      <p class="empty-message">
        {searchQuery 
          ? `We couldn't find any devices with ID "${searchQuery}". Try a different search term.` 
          : 'There are currently no devices in the system. Data will appear here once devices start reporting.'}
      </p>
      {#if searchQuery}
        <button class="clear-search-btn" on:click={() => { searchQuery = ''; currentPage = 1; loadDevices(); }}>
          &#128269; Clear Search
        </button>
      {/if}
    </div>
  {:else}
    <table class="device-table">
      <thead>
        <tr>
          <th class="sortable" on:click={() => toggleSort('device_id')}>Device{getSortIcon('device_id')}</th>
          <th class="sortable" on:click={() => toggleSort('site')}>Site{getSortIcon('site')}</th>
          <th class="sortable" on:click={() => toggleSort('fuel_percent')}>Fuel{getSortIcon('fuel_percent')}</th>
          <th class="sortable" on:click={() => toggleSort('temperature')}>Temp{getSortIcon('temperature')}</th>
          <th class="sortable" on:click={() => toggleSort('flow_rate')}>Flow{getSortIcon('flow_rate')}</th>
          <th>Equipment</th>
          <th>Connection</th>
          <th>Last Seen</th>
        </tr>
      </thead>
      <tbody>
        {#each devices as device}
          <tr>
            <td><a href="/devices/{device.device_id}" class="device-link">{device.device_id}</a></td>
            <td>{device.site || '-'}</td>
            <td>
              <div class="fuel-container">
                <div class="fuel-bar-track">
                  <div class="fuel-bar-fill" style="width: {Math.max(device.fuel_percent, 2)}%; background: {getFuelColor(device.fuel_percent)}"></div>
                </div>
                <span class="fuel-value" style="color: {getFuelColor(device.fuel_percent)}">{Math.round(device.fuel_percent)}%</span>
              </div>
            </td>
            <td style={device.temperature > 85 ? 'color: #ef4444; font-weight: 600;' : ''}>{device.temperature.toFixed(1)}&deg;C</td>
            <td>{device.flow_rate.toFixed(1)} L/min</td>
            <td><span class="badge {getEquipClass(device.equipment_status)}">{device.equipment_status}</span></td>
            <td>
              <span class="badge {getConnectionClass(device.connection)}">
                <span class="badge-dot"></span>
                {device.connection}
              </span>
            </td>
            <td class="time-cell">{device.last_seen ? new Date(device.last_seen).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'}) : '-'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    
    <div class="pagination-container">
      <div class="pagination-info">
        <span class="info-text">
          Showing {((currentPage - 1) * pageSize) + 1}–{Math.min(currentPage * pageSize, totalItems)} of {totalItems} devices
        </span>
        <select class="page-size-select" bind:value={pageSize} on:change={handlePageSizeChange}>
          <option value="5">5 per page</option>
          <option value="10">10 per page</option>
          <option value="15">15 per page</option>
        </select>
      </div>
      
      <nav class="pagination-nav">
        <button 
          class="page-btn" 
          disabled={currentPage <= 1}
          on:click={() => changePage(currentPage - 1)}>
          &laquo; Prev
        </button>
        
        {#each getPageNumbers() as page}
          {#if page === '...'}
            <span class="ellipsis">...</span>
          {:else}
            <button 
              class="page-btn {page === currentPage ? 'active' : ''}"
              on:click={() => changePage(page)}>
              {page}
            </button>
          {/if}
        {/each}
        
        <button 
          class="page-btn" 
          disabled={currentPage >= totalPages}
          on:click={() => changePage(currentPage + 1)}>
          Next &raquo;
        </button>
      </nav>
    </div>
  {/if}
</div>

<style>
  .loading-spinner {
    display: inline-block;
    width: 24px;
    height: 24px;
    border: 3px solid var(--border-light);
    border-top-color: var(--accent-blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
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
  
  .table-container {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    overflow: hidden;
  }
  
  .filters-bar {
    display: flex;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-bottom: 1px solid var(--border-light);
    flex-wrap: wrap;
    align-items: center;
  }
  
  .search-box {
    position: relative;
    flex: 1;
    min-width: 200px;
  }
  
  .search-icon {
    position: absolute;
    left: 0.75rem;
    top: 50%;
    transform: translateY(-50%);
    color: var(--text-muted);
    pointer-events: none;
  }
  
  .search-box input {
    width: 100%;
    padding: 0.5rem 0.75rem 0.5rem 2.25rem;
    border: 1px solid var(--border-light);
    border-radius: var(--radius-md);
    font-size: 0.9rem;
    color: var(--text-primary);
    background: var(--bg-tertiary);
    transition: border-color 0.2s ease;
  }
  
  .search-box input:focus {
    outline: none;
    border-color: var(--accent-blue);
    background: var(--bg-secondary);
  }
  
  .search-box input::placeholder {
    color: var(--text-muted);
  }
  
  .filter-select {
    padding: 0.5rem 2rem 0.5rem 0.75rem;
    border: 1px solid var(--border-light);
    border-radius: var(--radius-md);
    font-size: 0.85rem;
    color: var(--text-primary);
    background: var(--bg-tertiary) url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%2364748b' d='M6 8L2 4h8z'/%3E%3C/svg%3E") no-repeat right 0.5rem center;
    cursor: pointer;
    appearance: none;
    min-width: 130px;
  }
  
  .filter-select:focus {
    outline: none;
    border-color: var(--accent-blue);
  }
  
  .export-btn {
    background: var(--success-bg);
    color: var(--success-text);
    border: 1px solid var(--success);
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    font-size: 0.85rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    white-space: nowrap;
  }
  
  .export-btn:hover {
    background: var(--success);
    color: white;
  }
  
  .time-cell {
    color: var(--text-secondary);
    font-size: 0.85rem;
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
  
  @media (max-width: 768px) {
    .pagination-container {
      flex-direction: column;
      align-items: stretch;
    }
    
    .pagination-nav {
      justify-content: center;
      flex-wrap: wrap;
    }
  }
  
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    text-align: center;
    color: var(--text-muted);
  }
  
  .empty-icon {
    font-size: 4rem;
    margin-bottom: 1.5rem;
    opacity: 0.5;
  }
  
  .empty-title {
    font-size: 1.3rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin: 0 0 0.75rem 0;
  }
  
  .empty-message {
    font-size: 0.95rem;
    max-width: 400px;
    line-height: 1.6;
    margin: 0 0 1.5rem 0;
  }
  
  .clear-search-btn {
    background: var(--bg-tertiary);
    color: var(--accent-blue);
    border: 1px solid var(--border-light);
    padding: 0.6rem 1.25rem;
    border-radius: var(--radius-md);
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .clear-search-btn:hover {
    background: var(--accent-blue);
    color: white;
    border-color: var(--accent-blue);
  }
</style>
