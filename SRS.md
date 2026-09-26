# Software Requirements Specification (SRS)
## INTECS IoT Monitoring Web Application

| Field | Details |
|-------|---------|
| **Project Name** | INTECS IoT Monitoring Web |
| **Version** | 1.0 |
| **Status** | Draft |
| **Date** | 2026-09-26 |

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Overall Description](#2-overall-description)
3. [System Architecture](#3-system-architecture)
4. [External Interface Requirements](#4-external-interface-requirements)
5. [Functional Requirements](#5-functional-requirements)
6. [Non-Functional Requirements](#6-non-functional-requirements)
7. [Database Schema](#7-database-schema)
8. [API Reference](#8-api-reference)
9. [Configuration](#9-configuration)
10. [Deployment](#10-deployment)

---

## 1. Introduction

### 1.1 Purpose
This document specifies the software requirements for the INTECS IoT Monitoring Web application. It defines system behavior, interfaces, data models, constraints, and quality attributes for developers, testers, and reviewers.

### 1.2 Document Conventions
- **MUST**, **SHALL**: Absolute requirement
- **SHOULD**: Recommended, with valid reasons to deviate
- **MAY**: Optional capability

### 1.3 Definitions & Acronyms

| Term | Definition |
|------|------------|
| MQTT | Message Queuing Telemetry Transport — lightweight publish/subscribe protocol |
| WebSocket | Full-duplex communication channel over TCP (RFC 6455) |
| Telemetry | Time-series sensor data: fuel %, temperature, flow rate, equipment status |
| Hub | WebSocket connection multiplexer managing client connections and message routing |
| ReadPump / WritePump | Goroutine pairs handling inbound/outbound WebSocket traffic per client |
| STALE | Device connectivity state indicating delayed telemetry (1-5 minutes) |

### 1.4 References
- `rules.md` — Project development rules
- `.env.example` — Environment variable definitions
- `docker-compose.yml` — Deployment orchestration

---

## 2. Overall Description

### 2.1 Product Perspective
The INTECS platform is a self-contained monitoring system comprising five integrated services:
1. **Backend API** (Go) — MQTT ingestion, business logic, REST/WebSocket endpoints
2. **Frontend Dashboard** (SvelteKit) — Operator-facing web interface
3. **PostgreSQL Database** — Persistent storage for telemetry, devices, alerts, thresholds
4. **HiveMQ Broker** — Message routing between simulated/physical devices and backend
5. **IoT Simulator** — Go program generating realistic telemetry for development/demo

### 2.2 Product Functions (Summary)
| Function | Description |
|----------|-------------|
| Telemetry Ingestion | Parse and persist MQTT-published sensor readings |
| Real-Time Dashboard | Aggregate stats and device list with live status indicators |
| Alert Engine | Evaluate threshold conditions and generate structured alerts |
| Historical Visualization | Time-series charts (fuel, temperature) with time-range selection |
| Data Export | CSV download of telemetry, device inventory, and alert records |
| Threshold Configuration | Per-device customization of alert trigger values |
| WebSocket Notifications | Push-based updates for telemetry, alerts, and broker status |

### 2.3 User Characteristics

| Role | Technical Level | Expected Activities |
|------|----------------|--------------------|
| Plant Operator | Low-Medium | Monitor dashboard, acknowledge alerts, view device details |
| Maintenance Engineer | Medium | Analyze historical trends, export CSV reports |
| System Administrator | High | Configure thresholds, manage infrastructure |

### 2.4 Constraints
- PostgreSQL 16.x self-hosted (Docker container)
- HiveMQ 4.x as MQTT broker
- Go 1.23+ for backend and simulator
- SvelteKit 2 + Svelte 4 + Vite 5 for frontend
- Single admin authentication (no RBAC in Phase 1)
- On-premise deployment only (no cloud dependency)

### 2.5 Assumptions & Dependencies
- Devices publish at approximately 5-second intervals via MQTT QoS 1
- The `intecs/site/{site_id}/device/{device_id}/telemetry` topic pattern is consistent
- Server retains at least 4 GB RAM and 2 CPU cores for production use
- No network partitions exceeding 30 seconds between devices and broker

---

## 3. System Architecture

### 3.1 Component Diagram

```
┌─────────────┐         ┌──────────────────────────────────────────┐
│   Devices    │────────>│           HiveMQ Broker                  │
│  (Simulated) │  MQTT   │       (port 1883)                       │
└─────────────┘         └──────────────┬───────────────────────────┘
                                       │ subscribe: intecs/site/+/device/+/telemetry
                                       v
                     ┌─────────────────────────────────────┐
                     │        Backend API (Go)              │
                     │                                     │
                     │  ┌──────────┐ ┌────────────────┐   │
                     │  │ MQTT     │ │  Alert Engine  │   │
                     │  │ Client   │ │                │   │
                     │  └────┬─────┘ └────────┬───────┘   │
                     │       │                 │           │
                     │  ┌────▼─────────────────▼───────┐  │
                     │  │       Business Logic          │  │
                     │  │ • Device upsert               │  │
                     │  │ • Telemetry persistence       │  │
                     │  │ • Threshold evaluation        │  │
                     │  └────┬──────────────────┬──────┘  │
                     │       │                  │          │
                     │  ┌────▼─────┐   ┌────────▼───────┐ │
                     │  │ Postgres │   │  WebSocket Hub │ │
                     │  │          │   │                │ │
                     │  └──────────┘   │  • readPump    │ │
                     │                 │  • writePump   │ │
                     │                 │  • broadcast   │ │
                     │                 └───────┬────────┘ │
                     └─────────────────────────┼──────────┘
                                               │ WebSocket
                                               v
                     ┌─────────────────────────────────────┐
                     │      Frontend (SvelteKit)            │
                     │                                      │
                     │  ┌──────────┐ ┌────────────────┐   │
                     │  │Dashboard │ │Device Detail   │   │
                     │  │Component │ │Charts + CSV    │   │
                     │  └──────────┘ └────────────────┘   │
                     │  ┌──────────┐ ┌────────────────┐   │
                     │  │AlertList │ │WebSocket Client │   │
                     │  │Component│ │Reconnect        │   │
                     │  └──────────┘ └────────────────┘   │
                     └─────────────────────────────────────┘
```

### 3.2 Process Flow — Telemetry Ingestion

```
Device ──MQTT Publish──> HiveMQ ──MQTT Subscribe──> Backend
                                                        │
                                                       ▼
                                              Parse & Validate
                                                  JSON
                                                        │
                    ┌───────────────────────────────────┼───────────────────────┐
                    v                                   v                       v
              Upsert Site                         Upsert Device          Save Telemetry
              (sites table)                      (devices table)        (telemetry table)
                    │                                   │                       │
                    └───────────────────────────────────┴───────────────────────┘
                                                        │
                                                       ▼
                                              Evaluate Alerts
                                           (threshold checks +
                                            cooldown dedup)
                                                │         │
                                    if triggered  │         │
                                        │       v         │
                                 Insert Alert     │    Broadcast
                                 (alerts table)   │    via WebSocket
                                                    │
                                                    ▼
                                            All connected clients
                                            receive update in real-time
```

### 3.3 WebSocket Lifecycle

```
Client                          Hub                       Backend Handler
  │                               │                                │
  │  GET /api/ws upgrade          │                                │
  ├──────────────────────────────>│                                │
  │                               │  AddClient(conn)               │
  │                               ├─► register(client)             │
  │                               ├─► go writePump(hub)            │
  │                               ├─► go readPump(hub)             │
  │  200 Switching Protocols      │                                │
  │<──────────────────────────────┤                                │
  │                               │  SendMessage(mqtt_status)      │
  │  onopen → startHeartbeat()   │                                │
  │                               │                                │
  │  { type: "ping" }  every 30s  │                                │
  ├──────────────────────────────>│  readPump: SetReadDeadline(90s)│
  │                               │  reset on every ReadMessage    │
  │  PONG (auto-frame)            │                                │
  │<──────────────────────────────┤                                │
  │                               │                                │
  │  disconnect                   │                                │
  ├──────────────────────────────x│  removeClient(client)          │
  │ scheduleReconnect()           │  close(send channel)           │
  │ exponential backoff (20 max)  │                                │
```

---

## 4. External Interface Requirements

### 4.1 User Interfaces

#### 4.1.1 Login Page (`/login`)

| Element | Specification |
|---------|--------------|
| Credentials | Hardcoded: `admin@email.com` / `pass` |
| Token Storage | `localStorage`: `token` (base64 JWT-like), `user` |
| Token Expiry | 24 hours from creation |
| Auto-redirect | To `/` if valid token exists; to `/login` if expired |

#### 4.1.2 Dashboard (`/`)

| Element | Specification |
|---------|--------------|
| Stats Cards | Total Devices, Online, Offline, Active Alerts |
| Polling Interval | Every 5 seconds (`setInterval`) |
| Device Table | Paginated, searchable, filterable, sortable |
| Alert List | Two tabs: Active / History, auto-refresh every 10s / 60s |
| MQTT Status Bar | Green/red dot + "Last msg" timestamp |

#### 4.1.3 Device Detail (`/devices/[id]`)

| Element | Specification |
|---------|--------------|
| Metrics Display | Fuel %, Fuel Level, Temperature °C, Flow Rate L/min, Equipment Status, Connection Status |
| Charts | Chart.js line/area charts: Fuel History, Temperature History |
| Time Range Selector | 1h, 6h, 24h, 7d |
| Polling | Device info every 5s, telemetry every 15s |
| Live Indicator | Pulsing green dot when WebSocket telemetry arrives |

#### 4.1.4 Global Layout

| Element | Specification |
|---------|--------------|
| Navigation | Brand link, Dashboard, Devices, Alerts |
| Theme Toggle | Dark/Light mode persisted in localStorage |
| User Badge | Displays email role, logout button |

### 4.2 Hardware Interfaces

| Component | Minimum Specification |
|-----------|----------------------|
| Server | 2 vCPU, 4 GB RAM, 20 GB disk |
| Database | Shared host or dedicated container |
| Network | Internal VLAN for MQTT (< 50ms RTT) |

### 4.3 Software Interfaces

| Interface | Protocol/Format | Port |
|-----------|-----------------|------|
| Backend ↔ PostgreSQL | lib/pq (TCP) | 5432 |
| Backend ↔ HiveMQ | MQTT over TCP (QoS 1) | 1883 |
| Backend ↔ Frontend | REST (JSON) + WebSocket | 8080 → proxied to :5173 |
| Frontend ↔ Browser | HTTPS/WSS | 443/8443 (production) |

### 4.4 Communications Interface

#### MQTT Topic Pattern
```
intecs/site/{site_id}/device/{device_id}/telemetry
```

#### MQTT Message Format (JSON)
```json
{
  "device_id": "DT-001",
  "timestamp": "2026-09-25T15:30:00Z",
  "fuel_percentage": 72.50,
  "fuel_level": 7250.00,
  "temperature": 81.20,
  "flow_rate": 35.20,
  "equipment_status": "running"
}
```

#### MQTT Client Configuration
| Parameter | Value |
|-----------|-------|
| Client ID | `intecs-backend` |
| Keep Alive | 30 seconds |
| Auto Reconnect | Enabled |
| Clean Session | true |
| QoS | 1 (At least once) |

#### WebSocket Message Types (Server → Client)
| Type | Payload Structure |
|------|------------------|
| `telemetry` | `{ device_id, timestamp, fuel_percent, fuel_level, temperature, flow_rate, equipment_status }` |
| `alert` | `{ device_id, type, severity, message }` |
| `mqtt_status` | `{ connected: bool, last_msg_at: string }` |
| `error` | `{ message: string }` |

---

## 5. Functional Requirements

### 5.1 Authentication

| ID | Requirement | Details |
|----|-------------|---------|
| FR-AUTH-01 | System MUST enforce login before accessing any page | Layout component checks `localStorage.token` expiry on mount |
| FR-AUTH-02 | System MUST support single admin credential | Hardcoded: `admin@email.com` / `pass` |
| FR-AUTH-03 | System MUST store auth token with 24-hour expiry | Base64-encoded JSON in `localStorage`, validated client-side |
| FR-AUTH-04 | System MUST redirect unauthenticated users to `/login` | Any route access triggers `checkAuth()` guard |
| FR-AUTH-05 | System MUST clear session on logout | Clears `token` and `user` from `localStorage`, redirects to `/login` |

### 5.2 Dashboard

| ID | Requirement | Details |
|----|-------------|---------|
| FR-DASH-01 | System MUST display aggregate statistics | Total devices, online count, offline count, active alerts — fetched from `GET /api/dashboard` |
| FR-DASH-02 | System MUST refresh dashboard data every 5 seconds | `setInterval` calling `loadDashboardData()` |
| FR-DASH-03 | System MUST render paginated device table | Calls `GET /api/devices?page=&page_size=5&search=&connection=&equipment_status=&sort=&dir=` |
| FR-DASH-04 | System MUST provide client-side search by device ID | Text input filters results via API `?search=` parameter |
| FR-DASH-05 | System MUST allow filtering by connection status | Dropdown: ONLINE / STALE / OFFLINE mapped to API `?connection=` |
| FR-DASH-06 | System MUST allow filtering by equipment status | Dropdown: Running / Idle / Maintenance mapped to API `?equipment_status=` |
| FR-DASH-07 | System MUST sort any column ascending or descending | Click column header toggles ASC/DESC, sorts via API `?sort=&dir=` |
| FR-DASH-08 | System MUST support configurable page sizes | Dropdown selects 5, 10, or 15 items per page |
| FR-DASH-09 | System MUST render smart pagination UI | Shows first/last pages plus ellipsis (...) for large ranges |
| FR-DASH-10 | System MUST visualize fuel levels with color bars | Red (<10%), Orange (10-20%), Yellow (20-50%), Green (>50%) |
| FR-DASH-11 | System MUST indicate device connection status visually | ONLINE = green badge with pulse, STALE = yellow, OFFLINE = red |
| FR-DASH-12 | System MUST export visible devices as CSV | Generates CSV with columns: Device ID, Name, Site, Fuel %, Temp, Flow, Equipment, Connection, Last Seen |

### 5.3 Device Detail

| ID | Requirement | Details |
|----|-------------|---------|
| FR-DEV-01 | System MUST display real-time device metrics | Fetches `GET /api/devices/{id}` every 5 seconds |
| FR-DEV-02 | System MUST render two interactive charts | Fuel Percentage History (green area) and Temperature History (blue area) using Chart.js |
| FR-DEV-03 | System MUST allow time range selection | Buttons: 1 Jam, 6 Jam, 24 Jam, 7 Hari mapped to API `?range=1h\|6h\|24h\|7d` |
| FR-DEV-04 | System MUST fetch historical telemetry with limit | Calls `GET /api/devices/{id}/telemetry?range=&limit=500`, returns max 500 points |
| FR-DEV-05 | System MUST update charts on WebSocket arrival | Listens to `intecs:telemetry` custom event, filters by matching `device_id` |
| FR-DEV-06 | System MUST display live status indicator | Pulsing green dot while streaming, gray when polling only |
| FR-DEV-07 | System MUST export telemetry as CSV | Downloads file named `{deviceId}_{rangeLabel}_{date}.csv` with columns: Timestamp, Fuel %, Fuel Level, Temperature °C, Flow Rate L/min |
| FR-DEV-08 | System MUST show device metadata | Site name, device ID, last seen timestamp, equipment status, connection status |
| FR-DEV-09 | System MUST handle API errors gracefully | Shows error message with retry button after 3 consecutive failures |

### 5.4 Alert Management

| ID | Requirement | Details |
|----|-------------|---------|
| FR-ALERT-01 | System MUST classify alerts into types | `LOW_FUEL`, `CRITICAL_FUEL`, `HIGH_TEMPERATURE`, `CRITICAL_TEMPERATURE`, `DEVICE_OFFLINE` |
| FR-ALERT-02 | System MUST assign severity levels | `warning` (warn threshold breach), `critical` (critical threshold breach), `medium` (offline) |
| FR-ALERT-03 | System MUST persist alerts to database | Stored in `alerts` table with UUID, device_id, type, severity, message, status, timestamps |
| FR-ALERT-04 | System MUST support acknowledging alerts | POST `POST /api/alerts/{id}/acknowledge` sets status to `acknowledged`, populates `resolved_at` |
| FR-ALERT-05 | System MUST provide separate tabs for active and history alerts | Active tab polls every 10s, History tab polls every 60s |
| FR-ALERT-06 | System MUST display relative timestamps | "Just now", "2m ago", "1h ago", "3d ago", etc. |
| FR-ALERT-07 | System MUST color-code alerts by severity | Critical = red, Warning = orange, Medium = blue |
| FR-ALERT-08 | System MUST support independent pagination per tab | Active and history tabs have separate page/state and page size settings |
| FR-ALERT-09 | System MUST export alerts as CSV | Separate buttons for active alerts and history, includes filename prefix |
| FR-ALERT-10 | System MUST request browser notification permission | Uses `Notification.requestPermission()` on mount |
| FR-ALERT-11 | System MUST re-fetch on WebSocket alert events | Listens to `intecs:alert` custom event to trigger `loadAlerts()` and `loadHistory()` |

### 5.5 Alert Engine

| ID | Requirement | Details |
|----|-------------|---------|
| FR-ENG-01 | System MUST evaluate LOW_FUEL condition | Triggered when `fuel_percentage < warn_fuel_threshold` (default 20%) |
| FR-ENG-02 | System MUST evaluate CRITICAL_FUEL condition | Triggered when `fuel_percentage < crit_fuel_threshold` (default 10%) |
| FR-ENG-03 | System MUST evaluate HIGH_TEMPERATURE condition | Triggered when `temperature > warn_temp_threshold` (default 85°C) |
| FR-ENG-04 | System MUST evaluate CRITICAL_TEMPERATURE condition | Triggered when `temperature > crit_temp_threshold` (default 90°C) |
| FR-ENG-05 | System MUST enforce cooldown deduplication | In-memory map keyed by `"deviceID:alertType"` prevents duplicate alerts within `cooldown_seconds` (default 30s) |
| FR-ENG-06 | System MUST support per-device threshold configuration | Stored in `device_thresholds` table, upserted via API |
| FR-ENG-07 | System MUST fall back to system defaults | If no entry in `device_thresholds`, uses: warn_temp=85, crit_temp=90, warn_fuel=20, crit_fuel=10, cooldown=30 |
| FR-ENG-08 | System MUST broadcast new alerts via WebSocket | Sends `{ device_id, type, severity, message }` through hub `Broadcast(MessageTypeAlert, payload)` |

### 5.6 WebSocket

| ID | Requirement | Details |
|----|-------------|---------|
| FR-WS-01 | System MUST accept WebSocket upgrades at `/api/ws` | Gorilla websocket upgrader with `CheckOrigin` returning true |
| FR-WS-02 | System MUST send initial MQTT status on connect | `SendMessage(MessageTypeMQTTStatus, {connected, last_msg_at})` immediately after registration |
| FR-WS-03 | System MUST send PingMessage frames every 30 seconds | Via `writePump` ticker in server goroutine |
| FR-WS-04 | Server MUST reset ReadDeadline on each successful read | Prevents idle timeout disconnection; deadline set to 90 seconds from last activity |
| FR-WS-05 | System MUST handle client disconnects gracefully | `readPump` calls `hub.RemoveClient(client)` and `client.close()` on any read error |
| FR-WS-06 | System MUST broadcast incoming MQTT telemetry to all connected clients | Hub drains `broadcast` channel and fan-outs to every client's `send` channel |
| FR-WS-07 | Slow clients MUST be handled without blocking other clients | Timeout fallback of 100ms during broadcast instead of unconditional drop |
| FR-WS-08 | Client MUST implement exponential backoff reconnection | Delay formula: `min(1000 * 2^attempt, 60000)` ms, maximum 20 attempts |
| FR-WS-09 | Client MUST send periodic heartbeat pings | `{ type: "ping" }` sent every 30 seconds when `readyState === WebSocket.OPEN` |
| FR-WS-10 | Client MUST stop heartbeat and nullify ref on close | Clears interval timer, sets `ws = null`, schedules reconnect |
| FR-WS-11 | Client MUST parse and dispatch server messages | Routes by `msg.type`: `mqtt_status` → theme bar, `telemetry` → CustomEvent, `alert` → CustomEvent, `error` → console |

### 5.7 MQTT Broker Integration

| ID | Requirement | Details |
|----|-------------|---------|
| FR-MQTT-01 | System MUST subscribe to `intecs/site/+/device/+/telemetry` | Wildcard subscription on successful broker connect |
| FR-MQTT-02 | System MUST re-subscribe on broker reconnect | `SetOnConnectHandler` callback executed on every reconnect |
| FR-MQTT-03 | System MUST extract site_id and device_id from topic | Splits topic by `/`, parts[2] = site_id, parts[4] = device_id |
| FR-MQTT-04 | System MUST persist sites on first device message | Upserts into `sites` table automatically |
| FR-MQTT-05 | System MUST create device record on first telemetry | Inserts into `devices` table with type `"Industrial Equipment"` |
| FR-MQTT-06 | System MUST skip malformed JSON payloads | Logs parsing error and continues processing next message |

### 5.8 Theme Management

| ID | Requirement | Details |
|----|-------------|---------|
| FR-THEME-01 | System MUST support dark and light themes | CSS custom properties swap values via `[data-theme="dark"]` selector |
| FR-THEME-02 | System MUST persist theme preference | Stored in `localStorage.intecs-theme` |
| FR-THEME-03 | System MUST apply theme on mount | Reads from localStorage and calls `applyTheme()` which sets `document.documentElement.setAttribute("data-theme")` |

---

## 6. Non-Functional Requirements

### 6.1 Performance

| ID | Requirement | Metric |
|----|-------------|--------|
| NFR-PERF-01 | Dashboard initial load | < 3 seconds from route navigation |
| NFR-PERF-02 | API response time (p95) | < 200 ms for standard queries |
| NFR-PERF-03 | WebSocket message delivery latency | < 2 seconds from MQTT publish to client receipt |
| NFR-PERF-04 | Chart rendering time | < 2 seconds for full dataset (500 points) |
| NFR-PERF-05 | Concurrent WebSocket clients | Support minimum 50 simultaneous connections |
| NFR-PERF-06 | Telemetry ingestion throughput | Handle 100 devices × 12 messages/min = 1200 msg/min sustained |

### 6.2 Reliability

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-REL-01 | WebSocket reconnection success rate | > 95% of attempted reconnects succeed within 60 seconds |
| NFR-REL-02 | MQTT message delivery guarantee | QoS 1 ensures at-least-once delivery; broker handles duplicates |
| NFR-REL-03 | Database query reliability | Zero panics on unexpected schema or NULL values; uses `sql.NullFloat64` |
| NFR-REL-04 | Graceful degradation | If WebSocket unavailable, HTTP polling serves as fallback |
| NFR-REL-05 | Memory leak prevention | Each closed WebSocket cleanup releases both `readPump` and `writePump` goroutines and send channel |

### 6.3 Security

| ID | Requirement | Details |
|----|-------------|---------|
| NFR-SEC-01 | Origin validation | `CheckOrigin` returns true (development); SHOULD restrict to specific origins in production |
| NFR-SEC-02 | No secret exposure | Secrets NEVER committed or logged; loaded from environment variables |
| NFR-SEC-03 | Token validation | Client-side base64 decode with expiry check before granting access |
| NFR-SEC-04 | HTTPS/WSS ready | Nginx proxy supports TLS termination; upgrade headers configured |
| NFR-SEC-05 | SQL injection prevention | All queries use parameterized placeholders (`$1`, `$2`) |

### 6.4 Maintainability

| ID | Requirement | Details |
|----|-------------|---------|
| NFR-MAINT-01 | Code structure | Single Go package per functional domain; no circular dependencies |
| NFR-MAINT-02 | Logging | Structured log output with context: WebSocket codes, MQTT errors, DB query failures |
| NFR-MAINT-03 | Environment configuration | All external addresses via `.env` variables loaded by `godotenv` |
| NFR-MAINT-04 | Database migrations | Automatic schema creation on startup via `database.Migrate()` |

### 6.5 Usability

| ID | Requirement | Details |
|----|-------------|---------|
| NFR-USABLE-01 | Responsive design | Adapts layout at breakpoints: 480px (mobile), 768px (tablet), 1024px (desktop) |
| NFR-USABLE-02 | Empty states | Descriptive messages with actionable CTA (e.g., "No results found. Clear Search") |
| NFR-USABLE-03 | Loading feedback | Spinner overlays during data fetching; disabled buttons during operations |
| NFR-USABLE-04 | Error recovery | Retry buttons on failed API calls; manual reload links on persistent failures |

### 6.6 Scalability

| ID | Requirement | Consideration |
|----|-------------|---------------|
| NFR-SCALE-01 | Horizontal device scaling | Database indexes support efficient per-device queries (`idx_telemetry_device_timestamp`) |
| NFR-SCALE-02 | Broadcast fan-out optimization | Non-blocking channel send with default-case drop prevents single slow client blocking entire hub |
| NFR-SCALE-03 | Future sharding path | Site-level partitioning possible by adding `site_id` filters to queries |

### 6.7 Portability

| ID | Requirement | Details |
|----|-------------|---------|
| NFR-PORT-01 | Containerization | All services defined in `docker-compose.yml`; deployable on any Docker-compatible host |
| NFR-PORT-02 | Cross-platform build | Go modules compile for linux/amd64; frontend bundles static assets independently |

---

## 7. Database Schema

### 7.1 Sites Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `name` | TEXT | NOT NULL | Human-readable name (e.g., "Sangatta Site") |
| `site_id` | TEXT | UNIQUE, NOT NULL | Business key used in MQTT topics |
| `created_at` | TIMESTAMP | DEFAULT NOW() | Record creation time |

### 7.2 Devices Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `device_id` | TEXT | UNIQUE, NOT NULL | MQTT-derived identifier (e.g., "DT-001") |
| `site_id` | UUID | FK → sites(id) | Associated site |
| `name` | TEXT | | Display name |
| `type` | TEXT | NOT NULL, DEFAULT 'Industrial Equipment' | Device classification |
| `status` | TEXT | DEFAULT 'OFFLINE' | Runtime status (overridden by last_seen computation) |
| `last_seen` | TIMESTAMP | | Most recent telemetry timestamp |
| `created_at` | TIMESTAMP | DEFAULT NOW() | Record creation time |

### 7.3 Telemetry Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `device_id` | TEXT | NOT NULL | Reference to originating device |
| `timestamp` | TIMESTAMP | DEFAULT NOW() | Time the reading was taken |
| `fuel_percentage` | NUMERIC(5,2) | | Fuel level as percentage (0–100) |
| `fuel_level` | NUMERIC(10,2) | | Actual fuel volume in liters |
| `temperature` | NUMERIC(5,2) | | Current temperature in Celsius |
| `flow_rate` | NUMERIC(8,2) | | Fluid consumption rate in L/min |
| `equipment_status` | TEXT | | running / idle / maintenance |

**Indexes:**
- `idx_telemetry_device_timestamp ON telemetry(device_id, timestamp)`

### 7.4 Alerts Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `device_id` | TEXT | NOT NULL | Affected device |
| `type` | TEXT | NOT NULL | LOW_FUEL, CRITICAL_FUEL, HIGH_TEMPERATURE, CRITICAL_TEMPERATURE, DEVICE_OFFLINE |
| `severity` | TEXT | NOT NULL | warning, critical, medium |
| `message` | TEXT | NOT NULL | Human-readable alert description |
| `status` | TEXT | DEFAULT 'active' | active / acknowledged |
| `created_at` | TIMESTAMP | DEFAULT NOW() | Alert creation time |
| `resolved_at` | TIMESTAMP | | When alert was acknowledged |

**Indexes:**
- `idx_alerts_device ON alerts(device_id)`
- `idx_alerts_status ON alerts(status)`

### 7.5 Device Thresholds Table

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() | Internal identifier |
| `device_id` | TEXT | UNIQUE, NOT NULL | One row per device |
| `warn_temp_threshold` | NUMERIC(5,2) | DEFAULT 85 | Warning temperature threshold |
| `crit_temp_threshold` | NUMERIC(5,2) | DEFAULT 90 | Critical temperature threshold |
| `warn_fuel_threshold` | NUMERIC(5,2) | DEFAULT 20 | Warning low fuel threshold |
| `crit_fuel_threshold` | NUMERIC(5,2) | DEFAULT 10 | Critical low fuel threshold |
| `cooldown_seconds` | INT | DEFAULT 30 | Minimum interval between repeated alerts |
| `created_at` | TIMESTAMP | DEFAULT NOW() | |
| `updated_at` | TIMESTAMP | DEFAULT NOW() | |

### 7.6 Seed Data

On migration, if no site with `site_id = 'sangatta'` exists, one is created:

| name | site_id |
|------|---------|
| Sangatta Site | sangatta |

---

## 8. API Reference

### 8.1 Dashboard

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard` | Returns summary stats + current device list with latest telemetry |

**Response:**
```json
{
  "total_devices": 10,
  "online_devices": 7,
  "offline_devices": 3,
  "active_alerts": 2,
  "devices": [
    {
      "device_id": "DT-001",
      "name": "Engine Alpha",
      "site": "Sangatta Site",
      "fuel_percent": 45.20,
      "fuel_level": 4520.00,
      "temperature": 78.50,
      "flow_rate": 32.10,
      "equipment_status": "running",
      "connection": "ONLINE",
      "last_seen": "2026-09-25T15:30:00Z"
    }
  ]
}
```

### 8.2 Devices

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/devices` | List devices with pagination, search, filter, sort |
| GET | `/api/devices/{id}` | Get single device detail |
| GET | `/api/devices/{id}/telemetry` | Get historical telemetry points |

**Query Parameters (list):**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number (1-indexed) |
| `page_size` | integer | No | 5 | Items per page (valid: 5, 10, 15+) |
| `search` | string | No | "" | Filter by device_id substring |
| `connection` | string | No | "" | Filter: ONLINE, STALE, OFFLINE |
| `equipment_status` | string | No | "" | Filter: running, idle, maintenance |
| `sort` | string | No | "" | Sort column name |
| `dir` | string | No | desc | Sort direction: asc, desc |

**Query Parameters (telemetry):**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `range` | string | No | 24h | Time window: 1h, 6h, 24h, 7d |
| `limit` | integer | No | 500 | Maximum data points returned |

**Telemetry Response:**
```json
[
  {
    "timestamp": "2026-09-25T15:30:00Z",
    "fuel_percent": 72.50,
    "fuel_level": 7250.00,
    "temperature": 81.20,
    "flow_rate": 35.20
  }
]
```

### 8.3 Alerts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/alerts` | List active alerts with pagination |
| GET | `/api/alerts/history` | Full alert history with pagination |
| POST | `/api/alerts/{id}/acknowledge` | Mark alert as acknowledged |

**Query Parameters (alerts):**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `page_size` | integer | No | 10 | Items per page |
| `status` | string | No | active | Filter by status |

**Query Parameters (history):**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `page_size` | integer | No | 10 | Items per page |
| `device_id` | string | No | "" | Filter by device |
| `status` | string | No | "" | Filter by status |

### 8.4 Thresholds

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/thresholds` | List per-device or global thresholds |
| POST | `/api/thresholds` | Create or update per-device thresholds |

**Query Parameters (list):**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `device_id` | string | No | "" | Filter for specific device config |

**Request Body (update):**
```json
{
  "device_id": "DT-001",
  "warn_temp_threshold": 85,
  "crit_temp_threshold": 90,
  "warn_fuel_threshold": 20,
  "crit_fuel_threshold": 10,
  "cooldown_seconds": 30
}
```

**Response:**
```json
{
  "device_id": "DT-001",
  "warn_temp_threshold": 85,
  "crit_temp_threshold": 90,
  "warn_fuel_threshold": 20,
  "crit_fuel_threshold": 10,
  "cooldown_seconds": 30,
  "created_at": "2026-09-25T10:00:00Z",
  "updated_at": "2026-09-25T14:30:00Z"
}
```

### 8.5 WebSocket & MQTT Status

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/ws` | Upgrade to WebSocket connection |
| GET | `/api/mqtt/status` | Check MQTT broker connection status |

**MQTT Status Response:**
```json
{
  "connected": true,
  "last_msg_at": "2026-09-25T15:30:00Z"
}
```

### 8.6 Device Connectivity Status Rules

| Status | Condition | Time Window |
|--------|-----------|-------------|
| ONLINE | Recent telemetry received | `< 1 minute` ago |
| STALE | Telemetry delayed | `1–5 minutes` ago |
| OFFLINE | No telemetry reported | `> 5 minutes` ago |

---

## 9. Configuration

### 9.1 Environment Variables

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DATABASE_URL` | string | Yes | — | PostgreSQL connection string |
| `MQTT_BROKER_URL` | string | Yes | — | MQTT broker address (tcp://host:port) |
| `MQTT_USERNAME` | string | No | — | Broker authentication username |
| `MQTT_PASSWORD` | string | No | — | Broker authentication password |
| `DEVICE_COUNT` | integer | No | 10 | Number of simulated devices |
| `PUBLISH_INTERVAL` | duration | No | 5s | Simulator publish cadence |
| `SITE_ID` | string | No | sangatta | Default site identifier |
| `HIGH_TEMP_THRESHOLD` | float | No | 90 | System-wide critical temperature alert threshold |
| `ADMIN_PASSWORD` | string | No | admin123 | Listed but not actively used in code |

### 9.2 Default Alert Thresholds

| Parameter | Value | Unit | Description |
|-----------|-------|------|-------------|
| `warn_temp_threshold` | 85 | °C | Warning: temperature exceeds this value |
| `crit_temp_threshold` | 90 | °C | Critical: temperature exceeds this value |
| `warn_fuel_threshold` | 20 | % | Warning: fuel drops below this percentage |
| `crit_fuel_threshold` | 10 | % | Critical: fuel drops below this percentage |
| `cooldown_seconds` | 30 | seconds | Minimum gap between repeated alerts per device |

### 9.3 Nginx WebSocket Proxy Settings

| Directive | Value | Purpose |
|-----------|-------|---------|
| `proxy_http_version` | 1.1 | Required for WebSocket upgrade handshake |
| `proxy_set_header Upgrade` | `$http_upgrade` | Passes Upgrade header to backend |
| `proxy_set_header Connection` | `"upgrade"` | Signals WebSocket upgrade intent |

---

## 10. Deployment

### 10.1 Docker Compose Topology

| Service | Image | Ports | Depends On | Health Check |
|---------|-------|-------|------------|--------------|
| `postgres` | postgres:16-alpine | 5432:5432 | — | `pg_isready` |
| `hivemq` | hivemq/hivemq4:latest | 1883(MQTT), 8888, 8000(HA), 9090 | — | curl /health |
| `backend` | Build from ./backend | 8080:8080 | postgres (healthy), hivemq (started) | — |
| `simulator` | Build from ./simulator | (internal) | hivemq (started) | — |
| `frontend` | Build from ./frontend | 5173:80 | backend | — |

### 10.2 Startup Sequence

```
1. postgres starts, accepts connections
2. hivemq starts, broker available on port 1883
3. backend starts, runs database.Migrate(), connects to postgres and mqtt
4. backend.go mqtt.Run() subscribes to topics
5. simulator starts, begins publishing telemetry at PUBLISH_INTERVAL
6. frontend builds, nginx serves static assets, proxies /api/* to backend:8080
```

### 10.3 Volume Mounts

| Volume | Mount Point | Purpose |
|--------|-------------|---------|
| `postgres_data` | `/var/lib/postgresql/data` | Persistent database storage |

### 10.4 Production Hardening Checklist

- [ ] Replace hardcoded credentials with secrets management
- [ ] Restrict WebSocket `CheckOrigin` to known domains
- [ ] Enable HTTPS/WSS with valid TLS certificates
- [ ] Add `backend` service health check endpoint
- [ ] Configure log rotation for all containers
- [ ] Set resource limits (CPU/memory) per container
- [ ] Add database backup strategy (pg_dump cron)
- [ ] Implement proper authentication backend (JWT signing, password hashing)
- [ ] Add rate limiting on REST endpoints
- [ ] Enable CORS explicitly for known frontend origins
