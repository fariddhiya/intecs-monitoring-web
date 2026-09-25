<script>
  import { page } from '$app/stores';
  import '../app.css';
</script>

<nav class="navbar">
  <div class="nav-container">
    <a href="/" class="nav-brand">
      <span class="nav-brand-icon">&#9881;</span>
      INTECS
    </a>
    <div class="nav-links">
      <a href="/" class:active={$page.url.pathname === '/'}>Dashboard</a>
      <a href="/devices" class:active={$page.url.pathname.startsWith('/devices')} >Devices</a>
      <a href="/alerts" class:active={$page.url.pathname === '/alerts'}>Alerts</a>
    </div>
  </div>
</nav>

<main>
  <slot />
</main>
