<script>
  import { page } from "$app/stores";
  import { onMount, onDestroy } from "svelte";
  import { isTokenValid } from "$lib/utils/auth";
  import "../app.css";

  let mqttConnected = false;
  let mqttLastMsg = "";
  let wsReconnectAttempts = 0;
  let ws = null;
  let heartbeatTimer = null;
  let reconnectTimer = null;
  let darkMode = false;
  let themeChanged = 0;
  let user = null;
  let isAuthenticated = false;
  let navOpen = false;

  function checkAuth() {
    if (typeof window === "undefined") return;

    if ($page.url.pathname.startsWith("/login")) {
      if (isTokenValid()) {
        window.location.href = "/";
      }
      return;
    }

    if (!isTokenValid()) {
      window.location.href = "/login";
      return;
    }

    try {
      const token = localStorage.getItem("token");
      user = JSON.parse(atob(token));
      isAuthenticated = true;
    } catch (e) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      window.location.href = "/login";
    }
  }

  function logout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    user = null;
    isAuthenticated = false;
    window.location.href = "/login";
  }

  onMount(() => {
    checkAuth();

    if (typeof window !== "undefined") {
      darkMode = localStorage.getItem("intecs-theme") === "dark";
      applyTheme();
    }

    connectWebSocket();

    window.addEventListener("storage", () => {
      checkAuth();
    });
  });

  $: {
    if (themeChanged) {
      applyTheme();
    }
  }

  function applyTheme() {
    if (typeof document !== "undefined") {
      document.documentElement.setAttribute(
        "data-theme",
        darkMode ? "dark" : "light",
      );
    }
    localStorage.setItem("intecs-theme", darkMode ? "dark" : "light");
  }

  function toggleDarkMode() {
    darkMode = !darkMode;
    themeChanged += 1;
  }

  function connectWebSocket() {
    if (ws) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log("WebSocket connected");
      wsReconnectAttempts = 0;
      startHeartbeat();
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (e) {
        console.error("Error parsing WebSocket message:", e);
      }
    };

    ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason);
      stopHeartbeat();
      ws = null;
      scheduleReconnect();
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
  }

  function stopHeartbeat() {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer);
      heartbeatTimer = null;
    }
  }

  function startHeartbeat() {
    stopHeartbeat();
    heartbeatTimer = setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "ping" }));
      }
    }, 30000);
  }

  function scheduleReconnect() {
    if (reconnectTimer) return;
    const maxAttempts = 20;
    wsReconnectAttempts++;
    const delay = Math.min(
      1000 * Math.pow(2, Math.min(wsReconnectAttempts, maxAttempts)),
      60000,
    );
    console.log(`Reconnecting in ${delay}ms (attempt ${wsReconnectAttempts})`);
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connectWebSocket();
    }, delay);
  }

  function handleMessage(msg) {
    switch (msg.type) {
      case "mqtt_status":
        mqttConnected = msg.payload?.connected ?? false;
        mqttLastMsg = msg.payload?.last_msg_at || "";
        break;
      case "telemetry":
        window.dispatchEvent(
          new CustomEvent("intecs:telemetry", { detail: msg.payload }),
        );
        break;
      case "alert":
        window.dispatchEvent(
          new CustomEvent("intecs:alert", { detail: msg.payload }),
        );
        break;
      case "alert_updated":
        window.dispatchEvent(
          new CustomEvent("alert_updated", { detail: msg.payload }),
        );
        break;
      case "error":
        console.error("Server error:", msg.payload);
        break;
    }
  }

  function disconnectWebSocket() {
    stopHeartbeat();
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
    if (ws) {
      ws.close();
      ws = null;
    }
  }

  onDestroy(() => {
    disconnectWebSocket();
  });
</script>

{#if !$page.url.pathname.startsWith('/login')}
  <nav class="navbar" aria-label="Main navigation">
    <div class="nav-container">
      <a href="/" class="nav-brand" aria-label="INTECS dashboard home">
        <span class="nav-brand-icon">&#9881;</span>
        INTECS
      </a>

      <button
        class="nav-hamburger"
        type="button"
        aria-label="Toggle menu"
        aria-expanded={navOpen}
        on:click={() => navOpen = !navOpen}>
        <span class="hamburger-line"></span>
        <span class="hamburger-line"></span>
        <span class="hamburger-line"></span>
      </button>

      <div class="nav-links-wrapper" class:nav-links-open={navOpen}>
        <div class="nav-groups">
          <div class="nav-group-center">
            <a
              href="/"
              class="nav-link"
              class:active={$page.url.pathname === "/"}
              on:click={() => navOpen = false}>
              Dashboard
            </a>
            <a
              href="/devices"
              class="nav-link"
              class:active={$page.url.pathname.startsWith("/devices")}
              on:click={() => navOpen = false}>
              Devices
            </a>
            <a
              href="/alerts"
              class="nav-link"
              class:active={$page.url.pathname === "/alerts"}
              on:click={() => navOpen = false}>
              Alerts
            </a>
          </div>

          <div class="nav-group-right">
            {#if isAuthenticated}
              <span class="nav-user-avatar">&#128100;</span>
              <span class="nav-user-email">{user?.email || "Admin"}</span>
            {/if}
            <button class="nav-logout" type="button" on:click={logout} aria-label="Logout">
              Logout
            </button>
            <button
              class="nav-theme-toggle"
              type="button"
              on:click={toggleDarkMode}
              title="Toggle dark mode"
              aria-label="Toggle dark mode">
              {#if darkMode}
                &#9788;
              {:else}
                &#9790;
              {/if}
            </button>
          </div>
        </div>
      </div>
    </div>
    <div class="mqtt-status-bar" class:disconnected={!mqttConnected}>
      <span class="mqtt-status-dot" class:online={mqttConnected}></span>
      <span class="mqtt-status-text">
        {mqttConnected ? "MQTT Connected" : "MQTT Disconnected"}
        {mqttLastMsg
          ? ` · Last msg: ${(() => { const d = new Date(mqttLastMsg); const dd = String(d.getDate()).padStart(2,'0'); const mm = String(d.getMonth()+1).padStart(2,'0'); const yy = String(d.getFullYear()%100).padStart(2,'0'); const hh = String(d.getHours()).padStart(2,'0'); const mn = String(d.getMinutes()).padStart(2,'0'); const ss = String(d.getSeconds()).padStart(2,'0'); return `${dd}:${mm}:${yy} ${hh}:${mn}:${ss}`; })()}`
          : ""}
      </span>
    </div>
  </nav>

  {#if navOpen}
    <div class="nav-overlay" role="presentation" tabindex="-1" on:keydown={(e) => e.key === 'Escape' && (navOpen = false)} on:click={() => navOpen = false}></div>
  {/if}
{/if}

<main class:login-main={$page.url.pathname.startsWith('/login')}>
  <slot />
</main>

<style>
  nav.navbar {
    position: sticky;
    top: 0;
    z-index: 50;
    background-color: #151e2d;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3), 0 1px 2px rgba(0, 0, 0, 0.2);
  }

  [data-theme="light"] nav.navbar {
    background-color: #ffffff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08), 0 1px 2px rgba(0, 0, 0, 0.06);
  }

  .nav-container {
    max-width: 1280px;
    margin: 0 auto;
    padding: 0 1.5rem;
    display: flex;
    align-items: center;
    height: 64px;
    gap: 1rem;
  }

  /* Brand */
  .nav-brand {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    text-decoration: none;
    color: white;
    font-weight: 700;
    font-size: 1.15rem;
    letter-spacing: 0.03em;
    flex-shrink: 0;
    transition: opacity 0.2s ease;
  }

  [data-theme="light"] .nav-brand {
    color: #0f172a;
  }

  .nav-brand:hover {
    opacity: 0.85;
  }

  .nav-brand-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    background: linear-gradient(135deg, #3b82f6, #1d4ed8);
    border-radius: 6px;
    font-size: 1rem;
    flex-shrink: 0;
  }

  /* Hamburger */
  .nav-hamburger {
    display: none;
    flex-direction: column;
    justify-content: center;
    gap: 5px;
    background: none;
    border: none;
    cursor: pointer;
    padding: 8px;
    border-radius: 6px;
    transition: background 0.2s ease;
  }

  .nav-hamburger:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  [data-theme="light"] .nav-hamburger:hover {
    background: var(--bg-tertiary);
  }

  .hamburger-line {
    display: block;
    width: 22px;
    height: 2px;
    background: #e2e8f0;
    border-radius: 2px;
    transition: transform 0.3s ease, opacity 0.3s ease;
  }

  [data-theme="dark"] .hamburger-line {
    background: #94a3b8;
  }

  /* Links wrapper for stacking on mobile */
  .nav-links-wrapper {
    display: flex;
    align-items: center;
    flex: 1;
    gap: 2rem;
  }

  /* 3-group distribution with space-evenly */
  .nav-groups {
    display: flex;
    align-items: center;
    justify-content: space-evenly;
    width: 100%;
    gap: 2rem;
  }

  .nav-group-center {
    display: flex;
    align-items: center;
    gap: 0.375rem;
  }

  .nav-group-right {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .nav-link {
    text-decoration: none;
    color: #cbd5e1;
    padding: 0.45rem 0.85rem;
    border-radius: 8px;
    transition: all 0.2s ease;
    font-weight: 500;
    font-size: 0.875rem;
    line-height: 1.375;
    white-space: nowrap;
    position: relative;
  }

  [data-theme="light"] .nav-link {
    color: #475569;
  }

  .nav-link:hover {
    color: white;
    background: rgba(255, 255, 255, 0.08);
  }

  [data-theme="light"] .nav-link:hover {
    color: #0f172a;
    background: var(--bg-tertiary);
  }

  .nav-link.active {
    color: white;
    background: rgba(59, 130, 246, 0.2);
    font-weight: 600;
  }

  [data-theme="light"] .nav-link.active {
    color: #1e40af;
    background: #dbeafe;
  }

  .nav-link:focus-visible {
    outline: 2px solid #60a5fa;
    outline-offset: 2px;
  }

  [data-theme="light"] .nav-link:focus-visible {
    outline-color: #3b82f6;
  }

  /* Actions (user + logout + theme toggle) */
  .nav-actions {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex-shrink: 0;
  }

  .nav-user-avatar {
    font-size: 1rem;
    line-height: 1;
    flex-shrink: 0;
  }

  .nav-user-email {
    font-size: 0.8125rem;
    color: #94a3b8;
    white-space: nowrap;
    font-weight: 400;
    max-width: 160px;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-left: 0.375rem;
    flex-shrink: 1;
  }

  [data-theme="light"] .nav-user-email {
    color: #64748b;
  }

  .nav-logout {
    background: transparent;
    border: 1px solid rgba(239, 68, 68, 0.25);
    color: #f87171;
    padding: 0.4rem 0.75rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.8125rem;
    font-weight: 500;
    transition: all 0.2s ease;
    white-space: nowrap;
    margin-left: 0.25rem;
  }

  [data-theme="light"] .nav-logout {
    border-color: rgba(239, 68, 68, 0.2);
    color: #dc2626;
  }

  .nav-logout:hover {
    background: rgba(239, 68, 68, 0.15);
    border-color: rgba(239, 68, 68, 0.4);
    color: #fca5a5;
  }

  [data-theme="light"] .nav-logout:hover {
    background: rgba(239, 68, 68, 0.08);
    border-color: rgba(239, 68, 68, 0.3);
    color: #b91c1c;
  }

  .nav-logout:focus-visible {
    outline: 2px solid #f87171;
    outline-offset: 2px;
  }

  [data-theme="light"] .nav-logout:focus-visible {
    outline-color: #dc2626;
  }

  /* Theme toggle */
  .nav-theme-toggle {
    background: none;
    border: 1px solid rgba(255, 255, 255, 0.1);
    color: #94a3b8;
    padding: 0.4rem;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1rem;
    line-height: 1;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    margin-left: 0.25rem;
  }

  [data-theme="light"] .nav-theme-toggle {
    border-color: var(--border-light);
    color: #64748b;
  }

  .nav-theme-toggle:hover {
    color: white;
    background: rgba(255, 255, 255, 0.08);
    border-color: rgba(255, 255, 255, 0.2);
  }

  [data-theme="light"] .nav-theme-toggle:hover {
    color: #0f172a;
    background: var(--bg-tertiary);
    border-color: var(--border-medium);
  }

  .nav-theme-toggle:focus-visible {
    outline: 2px solid #60a5fa;
    outline-offset: 2px;
  }

  /* Mobile overlay */
  .nav-overlay {
    position: fixed;
    inset: 0;
    top: 64px;
    z-index: 40;
    background: rgba(0, 0, 0, 0.5);
  }

  /* MQTT Status Bar */
  .mqtt-status-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.3rem 1.5rem;
    font-size: 0.75rem;
    background-color: rgba(0, 0, 0, 0.2);
    color: #94a3b8;
    transition: background-color 0.3s ease, color 0.3s ease;
    border-top: 1px solid rgba(255, 255, 255, 0.04);
  }

  [data-theme="light"] .mqtt-status-bar {
    background-color: var(--bg-tertiary);
    color: var(--text-muted);
    border-top: 1px solid var(--border-light);
  }

  .mqtt-status-bar.disconnected {
    background-color: rgba(239, 68, 68, 0.1);
    color: #fca5a5;
  }

  [data-theme="light"] .mqtt-status-bar.disconnected {
    background-color: var(--danger-bg);
    color: var(--danger-text);
  }

  .mqtt-status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background-color: var(--danger);
    transition: background-color 0.3s ease;
    flex-shrink: 0;
  }

  .mqtt-status-dot.online {
    background-color: var(--success);
    box-shadow: 0 0 4px rgba(16, 185, 129, 0.4);
  }

  .mqtt-status-text {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  main.login-main {
    padding: 0;
  }

  /* Tablet */
  @media (max-width: 1024px) {
    .nav-container {
      padding: 0 1.25rem;
      gap: 1.25rem;
    }

    .nav-user-email {
      max-width: 120px;
    }
  }

  /* Mobile */
  @media (max-width: 768px) {
    .nav-hamburger {
      display: flex;
    }

    .nav-container {
      height: 56px;
      gap: 0.75rem;
      padding: 0 1rem;
    }

    .nav-brand {
      font-size: 1.05rem;
    }

    .nav-brand-icon {
      width: 28px;
      height: 28px;
      font-size: 0.95rem;
    }

    .nav-links-wrapper {
      flex-direction: column;
      align-items: stretch;
      gap: 0;
      position: absolute;
      top: 56px;
      left: 0;
      right: 0;
      z-index: 45;
      background-color: #151e2d;
      padding: 0;
      max-height: 0;
      overflow: hidden;
      transition: max-height 0.3s ease, padding 0.3s ease;
    }

    [data-theme="light"] .nav-links-wrapper {
      background-color: #ffffff;
    }

    .nav-links-wrapper.nav-links-open {
      max-height: calc(100vh - 56px);
      overflow-y: auto;
    }

    .nav-links-wrapper > :first-child {
      padding: 0.5rem 1rem;
      border-bottom: 1px solid rgba(255, 255, 255, 0.06);
    }

    [data-theme="light"] .nav-links-wrapper > :first-child {
      border-bottom-color: var(--border-light);
    }

    .nav-groups {
      flex-direction: column;
      align-items: stretch;
      gap: 0;
    }

    .nav-group-center {
      flex-direction: column;
      gap: 0;
    }

    .nav-group-right {
      justify-content: center;
      flex-wrap: wrap;
      padding: 0.75rem 0;
      border-top: 1px solid rgba(255, 255, 255, 0.06);
      gap: 0.5rem;
    }

    [data-theme="light"] .nav-group-right {
      border-top-color: var(--border-light);
    }

    .nav-actions {
      padding: 0.75rem 1rem;
      gap: 0.5rem;
      flex-wrap: wrap;
      border-top: 1px solid rgba(255, 255, 255, 0.06);
    }

    [data-theme="light"] .nav-actions {
      border-top-color: var(--border-light);
    }

    .nav-user-email {
      max-width: none;
    }

    .nav-link {
      display: block;
      width: 100%;
      padding: 0.625rem 0.75rem;
      font-size: 0.9rem;
      min-height: 44px;
      display: flex;
      align-items: center;
    }

    .nav-link.active {
      padding-left: 0.75rem;
    }

    .nav-overlay {
      top: 56px;
    }

    .mqtt-status-bar {
      padding: 0.25rem 1rem;
      font-size: 0.7rem;
    }
  }

  @media (max-width: 480px) {
    .nav-container {
      padding: 0 0.75rem;
    }

    .nav-user-avatar {
      font-size: 0.9rem;
    }

    .nav-user-email {
      font-size: 0.75rem;
    }

    .nav-logout {
      font-size: 0.75rem;
      padding: 0.35rem 0.6rem;
    }

    .nav-theme-toggle {
      width: 32px;
      height: 32px;
    }
  }
</style>
