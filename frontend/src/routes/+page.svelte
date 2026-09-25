<script>
  import { onMount } from 'svelte';
  import StatsCard from '$lib/components/StatsCard.svelte';
  import DeviceTable from '$lib/components/DeviceTable.svelte';
  import AlertList from '$lib/components/AlertList.svelte';

  let stats = { total_devices: 0, online_devices: 0, offline_devices: 0, active_alerts: 0 };
  let loading = true;
  let error = null;

  async function loadDashboard() {
    try {
      const res = await fetch('/api/dashboard');
      if (res.ok) {
        const data = await res.json();
        stats = {
          total_devices: data.total_devices,
          online_devices: data.online_devices,
          offline_devices: data.offline_devices,
          active_alerts: data.active_alerts,
        };
        error = null;
      } else {
        error = 'Failed to load dashboard data';
      }
    } catch (e) {
      error = 'Connection failed: ' + e.message;
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadDashboard();
    setInterval(loadDashboard, 5000);
  });
</script>

<svelte:head>
  <title>Dashboard - INTECS Monitoring</title>
</svelte:head>

<h1 class="page-title">Monitoring Dashboard</h1>

{#if error}
  <div class="error-message">&#9888; {error}</div>
{/if}

<div class="stats-grid">
  <StatsCard label="Total Devices" value={stats.total_devices} colorClass="stat-total" icon="&#128228;" />
  <StatsCard label="Online" value={stats.online_devices} colorClass="stat-online" icon="&#9989;" />
  <StatsCard label="Offline" value={stats.offline_devices} colorClass="stat-offline" icon="&#10060;" />
  <StatsCard label="Active Alerts" value={stats.active_alerts} colorClass="stat-alerts" icon="&#9888;" />
</div>

<DeviceTable />

<AlertList />

<style>
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1rem;
    margin-bottom: 2rem;
  }
</style>
