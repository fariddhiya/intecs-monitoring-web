<script>
  export let label = '';
  export let value = '--';
  export let colorClass = '';
  export let icon = '';
  
  let loading = true;
  
  $: {
    loading = false;
  }
</script>

<div class="stat-card {colorClass}">
  <div class="stat-header">
    {#if icon}
      <span class="stat-icon">{icon}</span>
    {/if}
    <span class="stat-label">{label}</span>
  </div>
  {#if loading}
    <span class="loading-spinner" style="margin-top: 0.5rem;"></span>
  {:else}
    <div class="stat-value">{value}</div>
  {/if}
</div>

<style>
  .stat-card {
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: 1.25rem 1.5rem;
    border: 1px solid var(--border-light);
    box-shadow: var(--shadow-sm);
    transition: all 0.2s ease;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .stat-card:hover {
    box-shadow: var(--shadow-md);
    transform: translateY(-1px);
  }
  
  .stat-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .stat-icon {
    font-size: 1.25rem;
  }
  
  .stat-label {
    font-size: 0.8rem;
    color: var(--text-muted);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  .stat-value {
    font-size: 1.875rem;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1.2;
  }
  
  .stat-total .stat-value { color: var(--accent-blue); }
  .stat-online .stat-value { color: var(--success); }
  .stat-offline .stat-value { color: var(--danger); }
  .stat-alerts .stat-value { color: var(--warning); }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .loading-spinner {
    display: inline-block;
    width: 20px;
    height: 20px;
    border: 2px solid var(--border-medium);
    border-top-color: var(--accent-blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
</style>
