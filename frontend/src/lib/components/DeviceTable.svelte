<script>
  import { onMount } from 'svelte';
  
  let devices = [];
  let allDevices = [];
  let error = null;
  let loading = true;
  let searchQuery = '';
  let filterConnection = '';
  let filterEquipment = '';
  let sortColumn = 'device_id';
  let sortDirection = 'ASC';

  async function loadDevices() {
    try {
      const params = new URLSearchParams();
      if (searchQuery) params.set('search', searchQuery);
      if (filterConnection) params.set('connection', filterConnection);
      if (filterEquipment) params.set('equipment_status', filterEquipment);
      if (sortColumn) params.set('sort', sortColumn);
      if (sortDirection) params.set('dir', sortDirection);
      
      const res = await fetch(`/api/devices?${params}`);
      if (res.ok) {
        allDevices = await res.json();
        applyFilters();
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

  $: {
    applyFilters();
  }

  function handleSearch(e) {
    searchQuery = e.target.value;
    loadDevices();
  }

  function handleFilterChange(field, value) {
    switch(field) {
      case 'connection': filterConnection = value; break;
      case 'equipment': filterEquipment = value; break;
    }
    loadDevices();
  }

  function toggleSort(column) {
    if (sortColumn === column) {
      sortDirection = sortDirection === 'ASC' ? 'DESC' : 'ASC';
    } else {
      sortColumn = column;
      sortDirection = 'ASC';
    }
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

  onMount(() => {
    loadDevices();
  });
</script>

<div class="table-container">
  <div class="table-header" style="display: flex; justify-content: space-between; align-items: center; gap: 1rem; flex-wrap: wrap;">
    <h2>Device Status ({devices.length} of {allDevices.length})</h2>
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
    
    {#if devices.length < allDevices.length}
      <span class="filter-active-badge">{allDevices.length - devices.length} filtered</span>
    {/if}
  </div>
  
  {#if loading}
    <div class="loading-state">
      <span class="loading-spinner"></span>
      <span>Loading devices...</span>
    </div>
  {:else if error}
    <div class="error-message" style="margin: 1rem;">&#9888; {error}</div>
  {:else if devices.length === 0}
    <div class="no-data">No devices found{searchQuery ? ' matching your search' : ''}</div>
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
  
  .filter-active-badge {
    font-size: 0.75rem;
    color: var(--info);
    background: rgba(14, 165, 233, 0.1);
    padding: 0.25rem 0.5rem;
    border-radius: 100px;
    white-space: nowrap;
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
</style>
