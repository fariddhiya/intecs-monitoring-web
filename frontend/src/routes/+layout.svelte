<script>
  import { page } from "$app/stores";
  import { onMount, onDestroy } from "svelte";
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

  function isTokenValid() {
    const token = localStorage.getItem("token");
    if (!token) return false;

    try {
      const decoded = JSON.parse(atob(token));
      return decoded.exp > Date.now();
    } catch (e) {
      localStorage.removeItem("token");
      localStorage.removeItem("user");
      return false;
    }
  }

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
    const delay = Math.min(1000 * Math.pow(2, Math.min(wsReconnectAttempts, maxAttempts)), 60000);
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

<nav class="navbar">
  <div class="nav-container">
    <a href="/" class="nav-brand">
      <span class="nav-brand-icon">&#9881;</span>
      INTECS
    </a>
    <div class="nav-links">
      <a href="/" class:active={$page.url.pathname === "/"}>Dashboard</a>
      <a
        href="/devices"
        class:active={$page.url.pathname.startsWith("/devices")}>Devices</a
      >
      <a href="/alerts" class:active={$page.url.pathname === "/alerts"}
        >Alerts</a
      >
    </div>
    {#if isAuthenticated}
      <div class="nav-user">
        <span class="user-email">&#128100; {user?.email || "Admin"}</span>
        <button class="logout-btn" on:click={logout}>Logout</button>
      </div>
    {/if}
    <button
      class="theme-toggle"
      on:click={toggleDarkMode}
      title="Toggle dark mode"
    >
      {#if darkMode}
        &#9788;
      {:else}
        &#9790;
      {/if}
    </button>
  </div>
</nav>

<div class="mqtt-status-bar" class:disconnected={!mqttConnected}>
  <span class="mqtt-status-dot" class:online={mqttConnected}></span>
  <span class="mqtt-status-text">
    {mqttConnected ? "MQTT Connected" : "MQTT Disconnected"}
    {mqttLastMsg
      ? ` · Last msg: ${new Date(mqttLastMsg).toLocaleTimeString()}`
      : ""}
  </span>
</div>

<main>
  <slot />
</main>

<style>
  .theme-toggle {
    background: none;
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: #94a3b8;
    padding: 0.5rem;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: 1.1rem;
    line-height: 1;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
  }

  .theme-toggle:hover {
    color: white;
    background: rgba(255, 255, 255, 0.1);
    border-color: rgba(255, 255, 255, 0.3);
  }

  .nav-user {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-right: 0.5rem;
    background-color: red;
  }

  .user-email {
    font-size: 0.85rem;
    color: #94a3b8;
    white-space: nowrap;
    font-weight: 500;
  }

  .logout-btn {
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: #fca5a5;
    padding: 0.4rem 0.85rem;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: 0.8rem;
    font-weight: 500;
    transition: all 0.2s ease;
    white-space: nowrap;
  }

  .logout-btn:hover {
    background: #ef4444;
    border-color: #ef4444;
    color: white;
  }

  @media (max-width: 768px) {
    .nav-user {
      display: none;
    }
  }
</style>
