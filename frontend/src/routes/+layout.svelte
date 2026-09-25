<script>
  import { page } from '$app/stores';
  import { onMount, onDestroy } from 'svelte';
  import '../app.css';
  
  let mqttConnected = false;
  let mqttLastMsg = '';
  let wsReconnectAttempts = 0;
  let ws = null;
  let heartbeatTimer = null;
  let reconnectTimer = null;
  
  function connectWebSocket() {
    if (ws) return;
    
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      console.log('WebSocket connected');
      wsReconnectAttempts = 0;
      startHeartbeat();
    };
    
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        handleMessage(msg);
      } catch (e) {
        console.error('Error parsing WebSocket message:', e);
      }
    };
    
    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      stopHeartbeat();
      ws = null;
      
      if (!event.wasClean && wsReconnectAttempts < 5) {
        scheduleReconnect();
      }
    };
    
    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
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
        ws.send(JSON.stringify({ type: 'ping' }));
      }
    }, 30000);
  }
  
  function scheduleReconnect() {
    if (reconnectTimer) return;
    wsReconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, wsReconnectAttempts), 30000);
    console.log(`Reconnecting in ${delay}ms (attempt ${wsReconnectAttempts})`);
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null;
      connectWebSocket();
    }, delay);
  }
  
  function handleMessage(msg) {
    switch (msg.type) {
      case 'mqtt_status':
        mqttConnected = msg.payload?.connected ?? false;
        mqttLastMsg = msg.payload?.last_msg_at || '';
        break;
      case 'telemetry':
        window.dispatchEvent(new CustomEvent('intecs:telemetry', { detail: msg.payload }));
        break;
      case 'alert':
        window.dispatchEvent(new CustomEvent('intecs:alert', { detail: msg.payload }));
        break;
      case 'error':
        console.error('Server error:', msg.payload);
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
  
  onMount(() => {
    connectWebSocket();
  });
  
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
      <a href="/" class:active={$page.url.pathname === '/'}>Dashboard</a>
      <a href="/devices" class:active={$page.url.pathname.startsWith('/devices')} >Devices</a>
      <a href="/alerts" class:active={$page.url.pathname === '/alerts'}>Alerts</a>
    </div>
  </div>
</nav>

<div class="mqtt-status-bar" class:disconnected={!mqttConnected}>
  <span class="mqtt-status-dot" class:online={mqttConnected}></span>
  <span class="mqtt-status-text">
    {mqttConnected ? 'MQTT Connected' : 'MQTT Disconnected'}
    {mqttLastMsg ? ` · Last msg: ${new Date(mqttLastMsg).toLocaleTimeString()}` : ''}
  </span>
</div>

<main>
  <slot />
</main>
