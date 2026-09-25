<script>
  import '../app.css';
</script>

<nav class="navbar">
  <div class="nav-container">
    <a href="/" class="nav-brand">🏭 IoT Monitoring</a>
    <div class="nav-links">
      <a href="/">Dashboard</a>
      <a href="/devices">Devices</a>
      <a href="/alerts">Alerts</a>
    </div>
  </div>
</nav>

<main>
  <slot />
</main>
