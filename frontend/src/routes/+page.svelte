<style>
  main {
    max-width: 1400px;
    margin: 0 auto;
    padding: 2rem 1rem;
  }
</style>

<svelte:head>
  <title>Dashboard - IoT Monitoring</title>
</svelte:head>

<main>
  <h1 style="margin-bottom: 1.5rem; font-size: 1.8rem;">Monitoring Dashboard</h1>
  
  <StatsCard />
  
  <DeviceTable />
  
  <AlertList />
</main>
