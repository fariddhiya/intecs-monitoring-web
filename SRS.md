# Software Requirements Specification (SRS)

## INTECS IoT Monitoring Web Application

| Field | Details |
|-------|---------|
| **Project Name** | INTECS IoT Monitoring Web |
| **Version** | 2.0 |
| **Status** | Approved |
| **Date** | 2026-09-27 |

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
11. [Appendices](#11-appendices)

---

## 1. Introduction

### 1.1 Purpose

This document specifies the software requirements for the INTECS IoT Monitoring Web application. It defines system behavior, interfaces, data models, constraints, and quality attributes for developers, testers, and reviewers. This specification serves as the authoritative technical reference for implementing, testing, and maintaining the system.

### 1.2 Scope

The INTECS platform ingests real-time telemetry from industrial equipment via MQTT, evaluates threshold conditions to generate severity-classified alerts, and provides operators with web-based dashboards for fleet monitoring, historical trend analysis, alert lifecycle management, and CSV reporting. The system supports multi-site operations through an abstracted site hierarchy and configurable per-device thresholds.

### 1.3 Document Conventions

| Keyword | Meaning |
|---------|---------|
| **MUST**, **SHALL** | Absolute requirement; no deviation permitted |
| **SHOULD** | Recommended requirement; valid reasons to deviate must be documented |
| **MAY** | Optional capability; implementation left to developer discretion |
| **[Conditional]** | Applies only when specified condition is met |

### 1.4 Definitions & Acronyms

| Term | Definition |
|------|------------|
| **MQTT** | Message Queuing Telemetry Transport — lightweight publish/subscribe protocol (OASIS Standard) |
| **WebSocket** | Full-duplex communication channel over TCP (RFC 6455) |
| **Telemetry** | Time-series sensor data: fuel %, temperature, flow rate, equipment status |
| **Hub** | WebSocket connection multiplexer managing client connections and message routing |
| **ReadPump / WritePump** | Goroutine pairs handling inbound/outbound WebSocket traffic per client |
| **STALE** | Device connectivity state indicating delayed telemetry (1–5 minutes since last seen) |
| **ONLINE** | Device connectivity state indicating recent telemetry (< 1 minute since last seen) |
| **OFFLINE** | Device connectivity state indicating no telemetry (> 5 minutes since last seen) |
| **Cooldown** | Minimum time interval between repeated alerts for same device+alertType combination |
| **Upsert** | Update or insert database operation; inserts if record doesn't exist, updates otherwise |
| **QoS 1** | MQTT Quality of Service Level 1 — At least once delivery guarantee |
| **MTTD** | Mean Time To Detection — average time from fault occurrence to detection |
| **MTTR** | Mean Time To Resolution — average time from fault occurrence to resolution |

### 1.5 References

| Reference | Description |
|-----------|-------------|
| `BRD.md` | Business Requirements Document — business objectives and use cases |
| `rules.md` | Project development rules and coding conventions |
| `.env.example` | Environment variable definitions and defaults |
| `docker-compose.yml` | Docker Compose deployment orchestration configuration |
| `backend/internal/database/database.go` | Database migration logic and schema definitions |
| RFC 6455 | The WebSocket Protocol |
| OASIS MQTT Spec | MQTT Version 3.1.1 Standard (v3.0) |

### 1.6 System Context

The INTECS IoT Monitoring System operates within an industrial/mining environment where equipment sensors are simulated or connected via physical gateways that publish MQTT messages to a central broker. The backend processes these messages, stores them in PostgreSQL, evaluates alert conditions, and pushes updates to operator dashboards via WebSocket. The frontend renders interactive visualizations and manages alert workflows.

---

## 2. Overall Description

### 2.1 Product Perspective

The INTECS platform is a self-contained monitoring system comprising five integrated services deployed via Docker Compose:

| # | Service | Technology | Port | Responsibility |
|---|---------|-----------|------|----------------|
| 1 | **Backend API** | Go 1.23+ | 8080 | MQTT ingestion, REST/WebSocket endpoints, alert engine, business logic |
| 2 | **Frontend Dashboard** | SvelteKit 2 / Vite 5 | 5173 → Nginx :80 | Operator-facing web interface with charts, tables, and alert management |
| 3 | **PostgreSQL Database** | PostgreSQL 16 | 5432 | Persistent storage for telemetry, devices, alerts, thresholds, sites |
| 4 | **HiveMQ Broker** | HiveMQ 4.x | 1883 (MQTT), 8000 (Admin) | Message routing between devices and backend |
| 5 | **IoT Simulator** | Go 1.23+ | (internal) | Generates realistic telemetry for 10 simulated devices during development/demo |

**Integration Points:**
- Backend subscribes to HiveMQ topic `intecs/site/+/device/+/telemetry`
- Frontend communicates with Backend via REST (JSON) and WebSocket (`/api/ws`)
- Simulator publishes telemetry to HiveMQ at configurable intervals (default: 5 seconds)
- Database auto-migrates schema and seeds default site on first startup

### 2.2 Product Functions (Summary)

| Function | ID | Description | Source BRD |
|----------|----|-------------|------------|
| Telemetry Ingestion | PF-01 | Parse and persist MQTT-published sensor readings with automatic upsert | HR-01, HR-09 |
| Real-Time Dashboard | PF-02 | Aggregate stats and paginated device list with live status indicators | HR-02, H-NFR-01 |
| Alert Engine | PF-03 | Evaluate threshold conditions and generate structured alerts with deduplication | HR-03, HR-11 |
| Historical Visualization | PF-04 | Time-series charts (fuel, temperature) with time-range selection | HR-04 |
| Data Export | PF-05 | CSV download of telemetry, device inventory, and alert records | HR-05 |
| Alert Lifecycle Management | PF-06 | Acknowledge/solve workflow with user attribution and audit trail | HR-08 |
| Threshold Configuration | PF-07 | Per-device customization of alert trigger values | HR-10, UC-05 |
| WebSocket Notifications | PF-08 | Push-based updates for telemetry, alerts, and broker status | HR-06, HR-07 |
| Site Abstraction | PF-09 | Multi-site support through hierarchical grouping | HR-09, UC-06 |
| Theme Management | PF-10 | Dark/light toggle persisted in localStorage | HR-12 |

### 2.3 User Characteristics

| Role | Technical Level | Expected Activities | Authentication |
|------|----------------|--------------------|---------------|
| Plant Operator | Low–Medium | Monitor dashboard, acknowledge alerts, solve alerts, export reports | Single admin account |
| Maintenance Engineer | Medium | Analyze historical trends, export CSV reports, review device details | Single admin account |
| System Administrator | High | Configure thresholds, manage infrastructure, troubleshoot pipelines | Single admin account |
| IoT Engineer | High | Design firmware, manage MQTT broker, validate telemetry, debug ingestion | Single admin account + broker admin access |

### 2.4 Constraints

| Constraint | Specification | Rationale |
|------------|--------------|-----------|
| Database | PostgreSQL 16.x self-hosted (Docker container) | Open-source, ACID compliance, robust JSON support |
| MQTT Broker | HiveMQ 4.x | Proven enterprise MQTT broker with wildcard subscription support |
| Backend Language | Go 1.23+ | Concurrent processing via goroutines, fast compilation, small binary |
| Frontend Framework | SvelteKit 2 + Svelte 4 + Vite 5 | Reactive UI without virtual DOM, excellent DX |
| Containerization | Docker Compose | Simple orchestration, reproducible builds, easy scaling |
| Authentication | Single admin account (no RBAC in Phase 1) | Simplified initial deployment; full auth planned for Phase 2 |
| Deployment Model | On-premise only (no cloud dependency) | Data sovereignty and network security requirements |
| Network Topology | Internal VLAN preferred (< 50ms RTT) | Ensures low-latency telemetry delivery for real-time dashboards |

### 2.5 Assumptions & Dependencies

| Assumption | Validation | Dependency |
|------------|-----------|------------|
| Devices publish at approximately 5-second intervals via MQTT QoS 1 | Configurable via `PUBLISH_INTERVAL` env var | Simulated devices conform; production devices must comply |
| Topic pattern `intecs/site/{site_id}/device/{device_id}/telemetry` is consistent across all devices | Backend parses topic by splitting on `/` | Device firmware/gateways follow naming convention |
| Server retains at least 2 vCPU and 4 GB RAM for production | Hardware provisioned by IT operations | Infrastructure budget approved |
| No network partitions exceeding 30 seconds between devices and broker | Network reliability assessment required | Stable internal network infrastructure |
| Default thresholds are appropriate for initial deployment (warn_temp=85°C, crit_temp=90°C, etc.) | Validated with engineering team; adjustable per device | Domain knowledge from maintenance team |

### 2.6 Limitations

| Limitation | Impact | Mitigation Path |
|------------|--------|----------------|
| Memory-only alert cooldown deduplication | State lost on restart | Persistent cooldown map in database |
| Base64-encoded token (not cryptographically signed) | Vulnerable if token stolen | JWT signing in Phase 2 |
| No mobile application | Desktop-bound interface | Progressive Web App or native app in Phase 3 |
| Single-node PostgreSQL | No automatic failover | Streaming replication planned for Phase 2 |

---

## 3. System Architecture

### 3.1 Component Diagram

```mermaid
graph TB
    subgraph IoT_Layer["IoT Layer"]
        D[Simulated Devices<br/>DT-001 to DT-010]
        style D fill:#e1f5fe
    end

    subgraph Broker_Layer["Broker Layer"]
        H[HiveMQ Broker<br/>port 1883]
        style H fill:#fff3e0
    end

    subgraph Backend_Layer["Backend Layer (Go)"]
        MC[MQTT Client<br/>Eclipse Paho]
        MP[Message Parser]
        DM[Device Manager]
        AM[Alert Manager]
        TP[Telemetry Persist]
        BL[Business Logic]
        RA[REST API<br/>Gorilla Mux]
        WH[WebSocket Hub]
        style MC fill:#e8f5e9
        style RA fill:#e8f5e9
        style WH fill:#e8f5e9
    end

    subgraph Storage_Layer["Storage Layer"]
        PG[(PostgreSQL 16)]
        style PG fill:#fce4ec
    end

    subgraph Frontend_Layer["Frontend Layer (SvelteKit)"]
        DL[Dashboard Layout]
        DPage[Dashboard Page]
        DevDetail[Device Detail]
        AlertList[Alert List]
        WSC[WebSocket Client]
        style DL fill:#f3e5f5
    end

    D -->|MQTT QoS 1| H
    H -->|Subscribe| MC
    MC --> MP
    MP --> DM
    MP --> AM
    DM --> TP
    AM --> TP
    TP --> PG
    BL <--> PG
    RA <--> PG
    WH <--> PG
    DPage -.->|REST| RA
    DevDetail -.->|REST| RA
    AlertList -.->|REST| RA
    DPage --> WSC
    DevDetail --> WSC
    AlertList --> WSC
    WSC <-->|WebSocket| WH
```

### 3.2 Process Flow — Telemetry Ingestion Pipeline

```mermaid
flowchart LR
    A[Device Publishes<br/>MQTT Topic + Payload] --> B[HiveMQ Router]
    B --> C[Backend MQTT Subscribe]
    C --> D{Valid JSON?}
    D -- No --> E[Log Error<br/>Skip Message]
    D -- Yes --> F[Extract site_id<br/>& device_id from Topic]
    F --> G[Parse Payload]
    G --> H[Upsert Site<br/>sites table]
    G --> I[Upsert Device<br/>devices table]
    G --> J[Save Telemetry<br/>telemetry table]
    H --> K[Evaluate Alerts<br/>per threshold rules]
    I --> K
    J --> K
    K --> L{Threshold<br/>Breached?}
    L -- No --> M[Continue Next]
    L -- Yes --> N{Cooldown<br/>Active?}
    N -- Yes --> M
    N -- No --> O[Insert Alert<br/>alerts table]
    O --> P[Broadcast via<br/>WebSocket Hub]
    P --> Q[All Connected Clients]
    Q --> R[Update UI<br/>Charts/Tables]
```

### 3.3 WebSocket Lifecycle

```mermaid
sequenceDiagram
    participant C as Client
    participant H as WebSocket Hub
    participant S as Backend Handler

    C->>S: GET /api/ws upgrade request
    S->>H: AddClient(conn)
    H->>H: register(client)
    H->>H: go writePump(hub)
    H->>H: go readPump(hub)
    S-->>C: 200 Switching Protocols
    H->>C: SendMessage(mqtt_status)
    
    Note over C,H: Heartbeat Loop
    
    C->>H: { type: "ping" } every 30s
    H->>H: readPump: SetReadDeadline(90s)
    H->>H: Reset deadline on every ReadMessage
    H-->>C: PONG (auto-frame by library)
    
    Note over C,H: Disconnect Handling
    
    C-xS: disconnect / error
    S->>H: removeClient(client)
    H->>H: close(send channel)
    C->>C: scheduleReconnect()
    C->>C: exponential backoff (max 20 attempts)
    Note over C: delay = min(1000 × 2^attempt, 60000) ms
```

### 3.4 Alert Evaluation Flow

```mermaid
flowchart TD
    A[Telemetry Received] --> B[Load Device Thresholds]
    B --> C{Per-device config<br/>exists?}
    C -- Yes --> D[Use custom thresholds]
    C -- No --> E[Use system defaults]
    E --> F[warn_temp=85<br/>crit_temp=90<br/>warn_fuel=20<br/>crit_fuel=10<br/>cooldown=30]
    D --> G[Evaluate Conditions]
    F --> G
    G --> H{fuel_pct < crit_fuel<br/>threshold?}
    H -- Yes --> I[Generate CRITICAL_FUEL<br/>severity=critical]
    H -- No --> J{fuel_pct < warn_fuel<br/>threshold?}
    J -- Yes --> K[Generate LOW_FUEL<br/>severity=warning]
    J -- No --> L{temp > crit_temp<br/>threshold?}
    L -- Yes --> M[Generate CRITICAL_TEMP<br/>severity=critical]
    L -- No --> N{temp > warn_temp<br/>threshold?}
    N -- Yes --> O[Generate HIGH_TEMP<br/>severity=warning]
    N -- No --> P{last_seen > 5min?}
    P -- Yes --> Q[Generate DEVICE_OFFLINE<br/>severity=medium]
    P -- No --> R[No Alert Generated]
    I --> S[Check Cooldown Map]
    K --> S
    M --> S
    O --> S
    Q --> S
    S --> T{Key exists in map<br/>& timestamp within<br/>cooldown period?}
    T -- Yes --> U[Duplicate - Skip]
    T -- No --> V[Insert Alert to DB]
    V --> W[Add key to cooldown map<br/>with expiry timestamp]
    W --> X[Broadcast via WebSocket]
```

---

## 4. External Interface Requirements

### 4.1 User Interfaces

#### 4.1.1 Login Page (`/login`)

| Element | Specification |
|---------|--------------|
| **Purpose** | Authenticate operator and establish session |
| **Credentials** | Hardcoded: `admin@email.com` / `pass` |
| **Token Format** | Base64-encoded JSON: `{ email, exp }` where exp = now + 24 hours |
| **Storage** | `localStorage`: keys `token` and `user` |
| **Validation** | Client-side decode + expiry check on mount via `checkAuth()` guard |
| **Auto-redirect** | To `/` if valid token exists; to `/login` if expired or absent |
| **Logout** | Clears `token` and `user` from `localStorage`; redirects to `/login` |

#### 4.1.2 Dashboard (`/`)

| Element | Specification |
|---------|--------------|
| **Stats Cards** | Total Devices, Online, Offline, Active Alerts — fetched from `GET /api/dashboard` |
| **Polling Interval** | Every 5 seconds via `setInterval` calling `loadDashboardData()` |
| **Device Table** | Paginated (server-side), searchable (`?search=`), filterable (`?connection=`, `?equipment_status=`), sortable (`?sort=` + `?dir=`) |
| **Smart Pagination** | Shows first/last pages plus ellipsis (`...`) for large ranges |
| **Page Size Options** | Dropdown: 5, 10, or 15 items per page |
| **Fuel Gauge Bars** | Color-coded embedded in table cells: Red (<10%), Orange (10–20%), Yellow (20–50%), Green (>50%) |
| **Connection Status Badges** | ONLINE = 🟢 green badge with pulse animation; STALE = 🟡 yellow; OFFLINE = 🔴 red |
| **Equipment Status Badges** | Running, Idle, Maintenance — text badges without color coding currently |
| **MQTT Status Bar** | Green/red dot + "Last msg" timestamp from `GET /api/mqtt/status` or WebSocket `mqtt_status` message |
| **CSV Export** | Generates CSV with columns: Device ID, Name, Site, Fuel %, Temp, Flow, Equipment Status, Connection Status, Last Seen |

#### 4.1.3 Device Detail Page (`/devices/[id]`)

| Element | Specification |
|---------|--------------|
| **Metrics Display** | Current readings: Fuel %, Fuel Level, Temperature °C, Flow Rate L/min, Equipment Status, Connection Status |
| **Connection Status** | Computed from `last_seen`: ONLINE (< 1 min), STALE (1–5 min), OFFLINE (> 5 min) |
| **Chart 1: Fuel History** | Chart.js area chart, green fill, data from `GET /api/devices/{id}/telemetry?range=&limit=500` |
| **Chart 2: Temperature History** | Chart.js line chart, blue stroke, same source |
| **Time Range Selector** | Buttons: 1 Jam (1h), 6 Jam (6h), 24 Jam (24h), 7 Hari (7d) — maps to API `?range=` parameter |
| **Polling Intervals** | Device info: every 5s; Telemetry history: every 15s |
| **Live Indicator** | Pulsing green dot while WebSocket telemetry stream active; gray when polling only |
| **CSV Export** | Downloads file named `{deviceId}_{rangeLabel}_{date}.csv` with columns: Timestamp, Fuel %, Fuel Level, Temperature °C, Flow Rate L/min |
| **Error Recovery** | Shows error message with retry button after 3 consecutive failures |

#### 4.1.4 Global Layout

| Element | Specification |
|---------|--------------|
| **Navigation Bar** | Brand link, Dashboard (/), Devices (/), Alerts (/links) |
| **Theme Toggle** | Dark/Light mode switch; icon toggles between sun/moon |
| **Theme Persistence** | Stored in `localStorage.intecs-theme` |
| **Theme Application** | Reads from localStorage on mount; calls `applyTheme()` which sets `document.documentElement.setAttribute("data-theme")` |
| **User Badge** | Displays authenticated email; logout button clears session |

#### 4.1.5 Alert Management (`/alerts`)

| Element | Specification |
|---------|--------------|
| **Tab Navigation** | Three tabs: Active, Acknowledged, Solved |
| **Tab Data Store** | Separate pagination/state per tab: `tabsData.active`, `tabsData.acknowledged`, `tabsData.solved` |
| **Auto-Refresh Rates** | Active: 10s; Acknowledged: 10s; Solved: 60s |
| **Fetch URLs** | Active: `?status=active`; Acknowledged: `?status=acknowledged`; Solved: `?status=solved` |
| **Severity Counters** | Aggregated per tab from `GET /api/alerts/severity-counts` response |
| **Filter Chips** | Per-tab severity filters: Critical, Warning, Medium (client-side filtering of loaded data) |
| **Alert List Items** | Severity icon (🔴 critical, 🟠 warning, 🔵 medium), device info, relative timestamps ("Just now", "2m ago"), action buttons |
| **Acknowledge Button** | Available on Active tab only; POST to `/api/alerts/{id}/acknowledge` with header `X-User-Email` |
| **Solve Modal** | Dialog requires resolution notes input; POST to `/api/alerts/{id}/solve` with body `{ notes, user_email }` |
| **Pagination** | Independent per tab with configurable page size (default: 5) |
| **Export Buttons** | Separate CSV export for each tab state |
| **Browser Notifications** | Requests permission on mount via `Notification.requestPermission()`; dispatches CustomEvent on `intecs:alert` WebSocket message |

### 4.2 Hardware Interfaces

| Component | Minimum Specification | Notes |
|-----------|----------------------|-------|
| **Server** | 2 vCPU, 4 GB RAM, 20 GB disk | Hosts all Docker containers |
| **Database** | Shared host or dedicated container | PostgreSQL consumes ~200 MB base + data growth |
| **Network** | Internal VLAN recommended (< 50ms RTT) | MQTT latency affects real-time responsiveness |

### 4.3 Software Interfaces

| Interface | Protocol/Format | Port | Direction |
|-----------|-----------------|------|-----------|
| Backend ↔ PostgreSQL | lib/pq (TCP, parameterized queries) | 5432 | Bidirectional |
| Backend ↔ HiveMQ | MQTT over TCP (QoS 1) | 1883 | Inbound (subscribe) |
| Backend ↔ Frontend (REST) | HTTP/1.1 + JSON | 8080 → proxied to :5173 | Outbound |
| Backend ↔ Frontend (WS) | WebSocket (Gorilla websocket) | 8080 | Bidirectional |
| Frontend ↔ Browser | HTTPS/WSS | 443/8443 (production) | Outbound |
| Simulator ↔ HiveMQ | MQTT over TCP (QoS 1) | 1883 | Outbound (publish) |

### 4.4 Communications Interface

#### MQTT Topic Structure

```
Pattern: intecs/site/{site_id}/device/{device_id}/telemetry

Examples:
  intecs/site/sangatta/device/DT-001/telemetry
  intecs/site/sangatta/device/DT-010/telemetry

Backend Subscribes To: intecs/site/+/device/+/telemetry
```

#### MQTT Message Format (JSON Payload)

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

**Field Specifications:**

| Field | Type | Required | Range | Description |
|-------|------|----------|-------|-------------|
| `device_id` | string | Yes | Non-empty | Unique device identifier (e.g., "DT-001") |
| `timestamp` | ISO 8601 datetime | No | — | Sensor reading timestamp; defaults to NOW() if omitted |
| `fuel_percentage` | float | Yes | 0–100 | Fuel level as percentage capacity |
| `fuel_level` | float | Yes | ≥ 0 | Actual fuel volume in liters |
| `temperature` | float | Yes | Any | Current temperature in Celsius |
| `flow_rate` | float | Yes | ≥ 0 | Fluid consumption rate in liters per minute |
| `equipment_status` | enum | No | running, idle, maintenance | Operating state of equipment |

#### MQTT Client Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| Client ID | `intecs-backend` | Unique identifier for backend subscriber |
| Keep Alive | 30 seconds | Interval for ping requests to broker |
| Auto Reconnect | Enabled | Automatic reconnection on network failure |
| Clean Session | true | Start fresh topic subscriptions on reconnect |
| QoS | 1 (At least once) | Delivery guarantee; duplicates handled by application |

#### WebSocket Messages (Server → Client)

| Type | Payload Structure | Dispatch |
|------|------------------|----------|
| `telemetry` | `{ device_id, timestamp, fuel_percent, fuel_level, temperature, flow_rate, equipment_status }` | CustomEvent `intecs:telemetry` |
| `alert` | `{ device_id, type, severity, message }` | CustomEvent `intecs:alert` |
| `mqtt_status` | `{ connected: boolean, last_msg_at: string }` | Updates theme bar indicator |
| `error` | `{ message: string }` | Logged to console |

#### WebSocket Client Behavior

| Behavior | Specification |
|----------|--------------|
| **Heartbeat** | Sends `{ type: "ping" }` every 30 seconds when `readyState === WebSocket.OPEN` |
| **Reconnection** | Exponential backoff: `delay = min(1000 × 2^attempt, 60000)` ms; maximum 20 attempts |
| **Message Parsing** | Routes by `msg.type`: `mqtt_status` → theme bar, `telemetry` → CustomEvent, `alert` → CustomEvent, `error` → console.log |
| **Lifecycle Cleanup** | On close: stops heartbeat interval, nullifies WebSocket ref, schedules reconnect |

---

## 5. Functional Requirements

### 5.1 Authentication & Session Management

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-AUTH-01 | System MUST enforce login before accessing any protected route | Layout component checks `localStorage.token` expiry on mount via `checkAuth()` function | P0 |
| FR-AUTH-02 | System MUST support single admin credential authentication | Hardcoded credentials: email `admin@email.com`, password `pass` | P0 |
| FR-AUTH-03 | System MUST store authentication token with 24-hour expiry | Token format: base64-encoded JSON `{ "email": "...", "exp": <unix_timestamp> }`; stored in `localStorage.token` | P0 |
| FR-AUTH-04 | System MUST redirect unauthenticated users to `/login` | Any route access triggers `checkAuth()` guard; redirects to `/login` if token missing or expired | P0 |
| FR-AUTH-05 | System MUST clear session completely on logout | Clears both `token` and `user` from `localStorage`; redirects to `/login` | P0 |

### 5.2 Dashboard Page

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-DASH-01 | System MUST display aggregate statistics cards | Total Devices, Online Devices, Offline Devices, Active Alerts — fetched from `GET /api/dashboard` | P0 |
| FR-DASH-02 | System MUST refresh dashboard data automatically every 5 seconds | `setInterval(loadDashboardData, 5000)` in component lifecycle | P0 |
| FR-DASH-03 | System MUST render paginated device table with server-side processing | Calls `GET /api/devices?page=&page_size=5&search=&connection=&equipment_status=&sort=&dir=` | P0 |
| FR-DASH-04 | System MUST provide search filtering by device ID | Text input maps to API `?search=` parameter (substring match) | P0 |
| FR-DASH-05 | System MUST allow filtering by connection status | Dropdown: ONLINE / STALE / OFFLINE mapped to API `?connection=` parameter | P0 |
| FR-DASH-06 | System MUST allow filtering by equipment status | Dropdown: Running / Idle / Maintenance mapped to API `?equipment_status=` parameter | P0 |
| FR-DASH-07 | System MUST allow sorting any column ascending or descending | Click column header toggles ASC/DESC; sorts via API `?sort=<column>&dir=asc\|desc` | P1 |
| FR-DASH-08 | System MUST support configurable page sizes | Dropdown selects 5, 10, or 15 items per page; resets page to 1 on change | P1 |
| FR-DASH-09 | System MUST render smart pagination UI | Shows first page, last page, current page, prev/next buttons, and ellipsis (`...`) for large gaps | P1 |
| FR-DASH-10 | System MUST visualize fuel levels with color bars in table cells | Red (< 10%), Orange (10–20%), Yellow (20–50%), Green (> 50%) background colors | P1 |
| FR-DASH-11 | System MUST indicate device connection status visually with color badges | ONLINE = 🟢 green badge with pulse CSS animation; STALE = 🟡 yellow; OFFLINE = 🔴 red | P0 |
| FR-DASH-12 | System MUST export visible filtered devices as CSV | Generates browser-downloadable CSV with columns: Device ID, Name, Site, Fuel %, Temp, Flow, Equipment Status, Connection Status, Last Seen | P1 |

### 5.3 Device Detail Page

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-DEV-01 | System MUST display real-time device metrics | Fetches `GET /api/devices/{id}` every 5 seconds; displays fuel %, fuel level, temperature, flow rate, equipment status, connection status | P0 |
| FR-DEV-02 | System MUST render two interactive charts using Chart.js | Fuel Percentage History (green area chart), Temperature History (blue line chart) | P0 |
| FR-DEV-03 | System MUST allow time range selection for historical data | Buttons: 1h, 6h, 24h, 7d; maps to API `?range=1h\|6h\|24h\|7d` parameter | P0 |
| FR-DEV-04 | System MUST fetch historical telemetry with limit | Calls `GET /api/devices/{id}/telemetry?range=&limit=500`; returns maximum 500 data points | P0 |
| FR-DEV-05 | System MUST update charts on WebSocket telemetry arrival | Listens to CustomEvent `intecs:telemetry`; filters by matching `device_id`; appends to chart data | P1 |
| FR-DEV-06 | System MUST display live status indicator alongside charts | Pulsing green dot while WebSocket telemetry arrives for device; transitions to gray when only polling | P1 |
| FR-DEV-07 | System MUST export historical telemetry as CSV | Generates downloadable file named `{deviceId}_{rangeLabel}_{dateString}.csv` with columns: Timestamp, Fuel %, Fuel Level, Temperature °C, Flow Rate L/min | P1 |
| FR-DEV-08 | System MUST show device metadata panel | Displays: Site name, device ID, last seen timestamp, equipment status, connection status | P1 |
| FR-DEV-09 | System MUST handle API errors gracefully | Shows descriptive error message with retry button; tracks consecutive failures (3 strikes threshold) | P1 |

### 5.4 Alert Management

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-ALERT-01 | System MUST classify alerts into five types | `LOW_FUEL`, `CRITICAL_FUEL`, `HIGH_TEMPERATURE`, `CRITICAL_TEMPERATURE`, `DEVICE_OFFLINE` | P0 |
| FR-ALERT-02 | System MUST assign severity levels to alerts | `warning` (warn threshold breach), `critical` (critical threshold breach), `medium` (offline detection) | P0 |
| FR-ALERT-03 | System MUST persist all alerts to database | Stored in `alerts` table with UUID primary key, device_id, type, severity, message, status, created_at, resolved_at, acknowledged_by, solved_by, solve_notes | P0 |
| FR-ALERT-04 | System MUST support alert acknowledgment | POST to `/api/alerts/{id}/acknowledge` sets `status='acknowledged'`, populates `acknowledged_by` with user email from `X-User-Email` header; broadcasts update via WebSocket | P0 |
| FR-ALERT-05 | System MUST provide three separate tabs for alert states | Active, Acknowledged, Solved — each with independent pagination, page size, and data store in `tabsData` object | P0 |
| FR-ALERT-06 | System MUST display relative human-readable timestamps | Utilities convert absolute timestamps to: "Just now", "Xm ago", "Xh ago", "Xd ago" based on difference from current time | P1 |
| FR-ALERT-07 | System MUST color-code alerts by severity | Critical = 🟥 red / orange styling; Warning = 🟧 orange; Medium = 🟦 blue | P0 |
| FR-ALERT-08 | System MUST support independent pagination per tab | Each tab (`tabsData.active`, `tabsData.acknowledged`, `tabsData.solved`) maintains its own `page`, `totalPages`, `totalItems`, and `data` array | P0 |
| FR-ALERT-09 | System MUST export alerts as CSV with per-tab options | Separate export buttons for Active, Acknowledged, and Solved tabs; filename includes prefix indicating alert state | P1 |
| FR-ALERT-10 | System MUST request browser notification permission | Calls `Notification.requestPermission()` on component mount; grants desktop notifications for new alerts | P2 |
| FR-ALERT-11 | System MUST re-fetch alerts on WebSocket alert events | Listens to CustomEvent `intecs:alert`; triggers relevant `loadAlerts()` or `loadHistory()` call to refresh data | P1 |
| FR-ALERT-12 | System MUST support solving alerts with resolution notes | Solving modal dialog requires non-empty `notes` field; POST to `/api/alerts/{id}/solve` with body `{ notes, user_email }`; sets `status='solved'`, `resolved_at=NOW()`, `solved_by`, `solve_notes` | P0 |
| FR-ALERT-13 | System MUST display severity count aggregates per tab | Counts fetched from `GET /api/alerts/severity-counts` which returns `{ active: {...}, acknowledged: {...}, solved: {...} }` with severity breakdowns | P1 |

### 5.5 Alert Engine

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-ENG-01 | System MUST evaluate LOW_FUEL condition | Triggered when `fuel_percentage < warn_fuel_threshold`; generates alert with `type="LOW_FUEL"`, `severity="warning"` | P0 |
| FR-ENG-02 | System MUST evaluate CRITICAL_FUEL condition | Triggered when `fuel_percentage < crit_fuel_threshold`; generates alert with `type="CRITICAL_FUEL"`, `severity="critical"` | P0 |
| FR-ENG-03 | System MUST evaluate HIGH_TEMPERATURE condition | Triggered when `temperature > warn_temp_threshold`; generates alert with `type="HIGH_TEMPERATURE"`, `severity="warning"` | P0 |
| FR-ENG-04 | System MUST evaluate CRITICAL_TEMPERATURE condition | Triggered when `temperature > crit_temp_threshold`; generates alert with `type="CRITICAL_TEMPERATURE"`, `severity="critical"` | P0 |
| FR-ENG-05 | System MUST enforce cooldown deduplication mechanism | In-memory `sync.Map` keyed by `"deviceID:alertType"` prevents duplicate alerts within configured `cooldown_seconds` (default 30s); entry cleaned up after cooldown expires | P0 |
| FR-ENG-06 | System MUST support per-device threshold configuration | Stored in `device_thresholds` table; upserted via `POST /api/thresholds`; one row per device | P0 |
| FR-ENG-07 | System MUST fall back to system-wide defaults when no per-device config exists | Defaults: `warn_temp=85`, `crit_temp=90`, `warn_fuel=20`, `crit_fuel=10`, `cooldown=30` | P0 |
| FR-ENG-08 | System MUST broadcast new alerts to all connected clients via WebSocket | Sends `{ device_id, type, severity, message }` through hub `Broadcast(MessageTypeAlert, payload)` | P0 |
| FR-ENG-09 | System MUST reset cooldown when alert is acknowledged or solved | Alert resolution resets the cooldown map entry allowing immediate re-triggering if condition persists | P1 |

### 5.6 WebSocket Connectivity

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-WS-01 | System MUST accept WebSocket upgrades at `/api/ws` | Gorilla websocket upgrader with `CheckOrigin` returning true (allow all origins in development) | P0 |
| FR-WS-02 | System MUST send initial MQTT status immediately on connect | After registration, sends `SendMessage(MessageTypeMQTTStatus, {connected, last_msg_at})` to newly connected client | P0 |
| FR-WS-03 | System MUST send PingMessage frames every 30 seconds | Via ticker in `writePump` goroutine in Hub | P0 |
| FR-WS-04 | Server MUST reset `ReadDeadline` on each successful read | Prevents idle timeout disconnection; deadline set to 90 seconds from last activity in `readPump` | P0 |
| FR-WS-05 | System MUST handle client disconnects gracefully | `readPump` calls `hub.RemoveClient(client)` and `client.close()` on any read error (including timeout) | P0 |
| FR-WS-06 | System MUST broadcast incoming MQTT telemetry to all connected clients | Hub drains `broadcast` channel and fan-outs to every registered client's `send` channel | P0 |
| FR-WS-07 | Slow clients MUST NOT block broadcasting to other clients | Timeout fallback of 100ms during broadcast (`time.After`) prevents indefinite blocking; slow client message dropped silently | P0 |
| FR-WS-08 | Client MUST implement exponential backoff reconnection strategy | Delay formula: `min(1000 × 2^attempt, 60000)` milliseconds; caps at maximum 20 total attempts | P0 |
| FR-WS-09 | Client MUST send periodic heartbeat pings to keep connection alive | JSON `{ type: "ping" }` sent every 30 seconds when `ws.readyState === WebSocket.OPEN` | P0 |
| FR-WS-10 | Client MUST properly clean up resources on close | Stops heartbeat interval timer, sets WebSocket ref to `null`, schedules reconnect | P0 |
| FR-WS-11 | Client MUST parse and dispatch server messages by type | Routes by `msg.type`: `mqtt_status` → theme status bar update, `telemetry` → dispatch CustomEvent `intecs:telemetry`, `alert` → dispatch CustomEvent `intecs:alert`, `error` → console.error | P0 |

### 5.7 MQTT Broker Integration

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-MQTT-01 | System MUST subscribe to wildcard topic `intecs/site/+/device/+/telemetry` | Eclipse Paho MQTT Go client with `Client.Subscribe()` on successful broker connect | P0 |
| FR-MQTT-02 | System MUST re-subscribe on every broker reconnect | `SetOnConnectHandler` callback executes re-subscription on each reconnect event | P0 |
| FR-MQTT-03 | System MUST extract `site_id` and `device_id` from incoming topic | Splits topic string by `/` delimiter; `parts[2]` = site_id, `parts[4]` = device_id | P0 |
| FR-MQTT-04 | System MUST automatically create site record on first device message | Upserts into `sites` table with `name = "{site_id}-Site"` (e.g., "sangatta-Site") | P0 |
| FR-MQTT-05 | System MUST create device record on first telemetry message | Inserts into `devices` table with `type = "Industrial Equipment"` default value | P0 |
| FR-MQTT-06 | System MUST skip malformed or unparsable JSON payloads | Logs parsing error; continues processing next message without disrupting pipeline | P0 |
| FR-MQTT-07 | System MUST compute device connection status based on `last_seen` timestamp | ONLINE if `last_seen` < 1 minute ago; STALE if 1–5 minutes ago; OFFLINE if > 5 minutes ago | P0 |

### 5.8 Theme Management

| ID | Requirement | Implementation Detail | Priority |
|----|-------------|---------------------|----------|
| FR-THEME-01 | System MUST support dark and light themes | CSS custom properties swap values via `[data-theme="dark"]` selector on `<html>` element | P1 |
| FR-THEME-02 | System MUST persist theme preference across sessions | Stored in `localStorage.intecs-theme` as string `"dark"` or `"light"` | P1 |
| FR-THEME-03 | System MUST apply theme immediately on mount | Reads from localStorage on layout mount; calls `applyTheme()` to set `document.documentElement.setAttribute("data-theme", ...)` | P1 |

---

## 6. Non-Functional Requirements

### 6.1 Performance

| ID | Requirement | Target Metric | Test Method |
|----|-------------|--------------|-------------|
| NFR-PERF-01 | Dashboard initial load time | < 3 seconds from route navigation to fully rendered view | Browser DevTools Network + Lighthouse |
| NFR-PERF-02 | API response time (p95) | < 200 ms for standard queries (dashboard, devices list) | APM tool or nginx access log analysis |
| NFR-PERF-03 | WebSocket message delivery latency | < 2 seconds from MQTT publish to client receipt end-to-end | Compare server broadcast timestamp vs. client handler timestamp |
| NFR-PERF-04 | Chart rendering time | < 2 seconds for full dataset (500 data points) | Browser DevTools Performance tab; measure paint completion |
| NFR-PERF-05 | Concurrent WebSocket client capacity | Support minimum 50 simultaneous connections without degradation | Load test with automated client simulator |
| NFR-PERF-06 | Telemetry ingestion throughput | Handle 100 devices × 12 messages/min = 1,200 msg/min sustained | MQTT broker publish counter vs. DB insert counter |
| NFR-PERF-07 | CSV export generation time | < 3 seconds for datasets up to 10,000 rows | Measure time from export click to download start |

### 6.2 Reliability

| ID | Requirement | Target | Verification |
|----|-------------|--------|-------------|
| NFR-REL-01 | WebSocket reconnection success rate | > 95% of attempted reconnects succeed within 60 seconds | Log reconnect attempts and outcomes over 7-day period |
| NFR-REL-02 | MQTT message delivery guarantee | QoS 1 ensures at-least-once delivery; broker handles duplicates via deduplication map | Compare published message count vs. processed message count |
| NFR-REL-03 | Database query safety | Zero panics/crashes on unexpected schema or NULL values; uses `sql.NullFloat64` for nullable numeric fields | Unit tests with edge-case data; chaos testing |
| NFR-REL-04 | Graceful degradation under WebSocket failure | HTTP polling (5-second interval) serves as functional fallback for all real-time features | Disconnect WebSocket; verify polling maintains operational parity |
| NFR-REL-05 | Memory leak prevention | Every closed WebSocket cleanup releases both `readPump` and `writePump` goroutines AND the `send` channel | Profile memory over 24-hour run with 100+ connect/disconnect cycles |
| NFR-REL-06 | Telemetry data durability | Zero data loss for valid telemetry during normal operation; all messages persisted to PostgreSQL | Audit trail comparing MQTT received count vs. `SELECT COUNT(*)` in telemetry table |

### 6.3 Security

| ID | Requirement | Implementation | Priority |
|----|-------------|---------------|----------|
| NFR-SEC-01 | Origin validation on WebSocket | `CheckOrigin` currently returns true (development mode); SHOULD restrict to specific domains in production | P1 |
| NFR-SEC-02 | Secret management | All secrets (database URL, MQTT credentials) loaded exclusively from environment variables via `godotenv`; never hardcoded or logged | P0 |
| NFR-SEC-03 | Token validation | Client-side base64 decode with 24-hour expiry check before granting dashboard access; no server-side validation in Phase 1 | P0 |
| NFR-SEC-04 | HTTPS/WSS readiness | Nginx reverse proxy configuration supports TLS termination; upgrade headers (`Upgrade`, `Connection`) configured for WebSocket passthrough | P1 |
| NFR-SEC-05 | SQL injection prevention | All database queries use parameterized placeholders (`$1`, `$2`, `$3`) via `database/sql`; zero string concatenation for query construction | P0 |
| NFR-SEC-06 | Input validation | Malformed MQTT JSON payloads skipped gracefully; API parameters validated for expected types/ranges | P1 |

### 6.4 Maintainability

| ID | Requirement | Implementation | Verification |
|----|-------------|---------------|-------------|
| NFR-MAINT-01 | Code structure discipline | Single Go package per functional domain (`api/`, `alert/`, `database/`, `device/`, `mqtt/`, `websocket/`); no circular dependencies | Go build succeeds; `go mod graph` shows clean dependency tree |
| NFR-MAINT-02 | Structured logging | Log output includes context: WebSocket open/close codes, MQTT connection errors, database query failures with query strings | Review log output during error scenarios |
| NFR-MAINT-03 | Environment-based configuration | All external addresses, ports, credentials via `.env` variables loaded by `godotenv.Parse()` on application startup | Change `.env` values and verify runtime behavior without code changes |
| NFR-MAINT-04 | Automatic database migrations | Schema created/updated on startup via `database.Migrate()` function; seed data inserted conditionally (if no existing `sangatta` site) | Drop database; restart backend; verify full schema recreation |
| NFR-MAINT-05 | Clear error messages | API handlers return appropriate HTTP status codes (400, 404, 500) with descriptive JSON error bodies | Test invalid inputs and verify response format |

### 6.5 Usability

| ID | Requirement | Implementation | Priority |
|----|-------------|---------------|----------|
| NFR-USABLE-01 | Responsive design breakpoints | Layout adapts at: 480px (mobile), 768px (tablet), 1024px (desktop); tested via Chrome DevTools device emulation | P1 |
| NFR-USABLE-02 | Empty state messaging | Descriptive messages with actionable CTAs (e.g., "No results found. Clear Search" with clickable clear button) | P1 |
| NFR-USABLE-03 | Loading feedback | Spinner overlays during data fetching; disabled button states during ongoing operations (acknowledge, solve, export) | P1 |
| NFR-USABLE-04 | Error recovery UX | Retry buttons on failed API calls; manual reload links on persistent failures; toast-style error notifications | P1 |
| NFR-USABLE-05 | Consistent visual hierarchy | Unified typography scale, spacing system, color palette defined in global CSS custom properties | P2 |

### 6.6 Scalability

| ID | Requirement | Consideration | Future Path |
|----|-------------|--------------|-------------|
| NFR-SCALE-01 | Horizontal device scaling | Database indexes support efficient per-device queries (`idx_telemetry_device_timestamp ON (device_id, timestamp)`) | Partition `telemetry` table by date for > 1M rows |
| NFR-SCALE-02 | Broadcast fan-out optimization | Non-blocking channel send with `time.After(100ms)` timeout during broadcast prevents single slow client from blocking entire hub | Shard WebSocket hub by client role or site |
| NFR-SCALE-03 | Future sharding path | Site-level partitioning possible by adding `site_id` filters to all queries; natural boundary for horizontal split | Deploy site-specific backend instances |
| NFR-SCALE-04 | Chart data limits | Telemetry API caps at 500 points (`limit=500`); prevents browser overload from large payloads | Implement server-side downsampling (LTTB algorithm) for > 1,000 points |

### 6.7 Portability

| ID | Requirement | Implementation | Verification |
|----|-------------|---------------|-------------|
| NFR-PORT-01 | Containerization | All services defined in `docker-compose.yml` with explicit images, ports, environment variables, healthchecks, and dependencies | `docker compose up -d` brings up complete stack on any Docker-compatible host |
| NFR-PORT-02 | Cross-platform build | Go modules compile for linux/amd64; frontend bundles static assets independently behind Nginx | Build on different host OS yields identical docker image layers |
| NFR-PORT-03 | Zero external service dependency | Everything runs locally: PostgreSQL, HiveMQ, Backend, Simulator, Frontend — no third-party cloud services required | Disconnect external network; verify all services operate correctly |

---

## 7. Database Schema

### 7.1 Sites Table

Stores organizational site hierarchy for multi-location operations.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal surrogate key |
| `name` | TEXT | NOT NULL | Human-readable name (e.g., "Sangatta Site") |
| `site_id` | TEXT | UNIQUE, NOT NULL | Business key used in MQTT topics (e.g., "sangatta") |
| `created_at` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Record creation timestamp |

**Indexes:** None beyond PK and UNIQUE constraint on `site_id`.

**Seed Data:** On migration, if no site with `site_id = 'sangatta'` exists, one is automatically created with `name = 'Sangatta Site'`.

### 7.2 Devices Table

Tracks all monitored equipment across all sites.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal surrogate key |
| `device_id` | TEXT | UNIQUE, NOT NULL | MQTT-derived identifier (e.g., "DT-001") |
| `site_id` | UUID | FK → `sites(id)` | Associated site reference |
| `name` | TEXT | NULLABLE | Human-readable device name |
| `type` | TEXT | NOT NULL, DEFAULT `'Industrial Equipment'` | Device classification |
| `status` | TEXT | DEFAULT `'OFFLINE'` | Runtime status indicator (overridden by `last_seen` computation for display) |
| `last_seen` | TIMESTAMP WITH TIME ZONE | NULLABLE | Most recent telemetry receipt timestamp |
| `created_at` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Record creation timestamp |

**Indexes:** `CREATE INDEX IF NOT EXISTS idx_alerts ON devices(device_id);`

### 7.3 Telemetry Table

Time-series storage for all sensor readings.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal surrogate key |
| `device_id` | TEXT | NOT NULL | Reference to originating device |
| `timestamp` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Time the reading was taken |
| `fuel_percentage` | NUMERIC(5,2) | NULLABLE | Fuel level as percentage (0–100) |
| `fuel_level` | NUMERIC(10,2) | NULLABLE | Actual fuel volume in liters |
| `temperature` | NUMERIC(5,2) | NULLABLE | Current temperature in Celsius |
| `flow_rate` | NUMERIC(8,2) | NULLABLE | Fluid consumption rate in L/min |
| `equipment_status` | TEXT | NULLABLE | Operating state: running / idle / maintenance |

**Indexes:**
- `CREATE INDEX IF NOT EXISTS idx_telemetry_device_timestamp ON telemetry(device_id, timestamp);` — Optimizes time-series queries per device

**Retention:** No automatic deletion policy in Phase 1; consider partitioning or archival for long-term growth.

### 7.4 Alerts Table

Manages the full alert lifecycle including acknowledgment and resolution tracking.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal surrogate key |
| `device_id` | TEXT | NOT NULL | Affected equipment device |
| `type` | TEXT | NOT NULL | Alert classification: `LOW_FUEL`, `CRITICAL_FUEL`, `HIGH_TEMPERATURE`, `CRITICAL_TEMPERATURE`, `DEVICE_OFFLINE` |
| `severity` | TEXT | NOT NULL | Priority level: `warning`, `critical`, `medium` |
| `message` | TEXT | NOT NULL | Human-readable alert description |
| `status` | TEXT | DEFAULT `'active'` | Lifecycle state: `active` → `acknowledged` → `solved` |
| `acknowledged_by` | TEXT | NULLABLE | Email of user who acknowledged |
| `solved_by` | TEXT | NULLABLE | Email of user who solved |
| `solve_notes` | TEXT | NULLABLE | Resolution explanation provided by solver |
| `created_at` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Alert creation timestamp |
| `resolved_at` | TIMESTAMP WITH TIME ZONE | NULLABLE | When alert was solved (set to `NOW()` on solve) |

**Indexes:**
- `CREATE INDEX IF NOT EXISTS idx_alerts_device ON alerts(device_id);` — Device alert lookup
- `CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);` — Active/alert filtering by status

### 7.5 Device Thresholds Table

Per-device override of default alert thresholds.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| `id` | UUID | PK, DEFAULT `gen_random_uuid()` | Internal surrogate key |
| `device_id` | TEXT | UNIQUE, NOT NULL, FK → `devices(device_id)` | One configuration row per device |
| `warn_temp_threshold` | NUMERIC(5,2) | DEFAULT 85 | Warning temperature threshold (°C) |
| `crit_temp_threshold` | NUMERIC(5,2) | DEFAULT 90 | Critical temperature threshold (°C) |
| `warn_fuel_threshold` | NUMERIC(5,2) | DEFAULT 20 | Warning low fuel threshold (%) |
| `crit_fuel_threshold` | NUMERIC(5,2) | DEFAULT 10 | Critical low fuel threshold (%) |
| `cooldown_seconds` | INT | DEFAULT 30 | Minimum interval between repeated alerts (seconds) |
| `created_at` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Configuration creation timestamp |
| `updated_at` | TIMESTAMP WITH TIME ZONE | DEFAULT `NOW()` | Last modification timestamp |

**UPSERT Behavior:** `POST /api/thresholds` performs upsert: if `device_id` already exists, update values; otherwise insert new row.

### 7.6 Entity Relationship Overview

```
sites (1) ────┬──── (N) devices
                  │
                  ├─── (N) telemetry
                  ├─── (N) alerts
                  └─── (1) device_thresholds
```

---

## 8. API Reference

### 8.1 Base URLs and Authentication

| Environment | Base URL | WebSocket URL |
|-------------|----------|--------------|
| Local Development | `http://localhost:8080/api` | `ws://localhost:8080/api/ws` |
| Production | `https://{domain}/api` | `wss://{domain}/api/ws` |

**Authentication:** Single admin session token stored in `localStorage.token`. Passed implicitly via `X-User-Email` header on alert actions.

### 8.2 Dashboard Endpoints

#### GET `/api/dashboard`

Returns summary statistics and current device list with latest telemetry.

**Response 200 OK:**
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

### 8.3 Device Endpoints

#### GET `/api/devices`

List devices with pagination, search, filter, and sort capabilities.

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number (1-indexed) |
| `page_size` | integer | No | 5 | Items per page (minimum: 5) |
| `search` | string | No | "" | Filter by device_id substring |
| `connection` | string | No | "" | Filter: `ONLINE`, `STALE`, `OFFLINE` |
| `equipment_status` | string | No | "" | Filter: `running`, `idle`, `maintenance` |
| `sort` | string | No | "" | Sort column (e.g., `device_id`, `fuel_percent`, `temperature`) |
| `dir` | string | No | `desc` | Sort direction: `asc` or `desc` |

**Response 200 OK:**
```json
{
  "devices": [/* Device objects */],
  "total_items": 10,
  "total_pages": 2,
  "current_page": 1,
  "page_size": 5
}
```

#### GET `/api/devices/{id}`

Retrieve single device detail by internal UUID.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Device internal identifier |

**Response 200 OK:** Device object with full telemetry snapshot.

**Response 404 Not Found:** Device not found.

#### GET `/api/devices/{id}/telemetry`

Retrieve historical telemetry data points for chart visualization.

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `range` | string | No | 24h | Time window: `1h`, `6h`, `24h`, `7d` |
| `limit` | integer | No | 500 | Maximum data points returned |

**Response 200 OK:**
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

### 8.4 Alert Endpoints

#### GET `/api/alerts`

List alerts with pagination and status filtering. Retrieves data ordered by `created_at DESC` (newest first).

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `page_size` | integer | No | 10 | Items per page |
| `status` | string | No | active | Filter: `active`, `acknowledged`, `solved` |

**Special Handling for `status=acknowledged`:** Query adds `AND resolved_at IS NULL` clause to distinguish acknowledged (active status + acknowledged state marker) from solved.

**Response 200 OK:**
```json
{
  "alerts": [
    {
      "id": "uuid-here",
      "device_id": "DT-001",
      "type": "CRITICAL_FUEL",
      "severity": "critical",
      "message": "Fuel level critical: 8.5%",
      "status": "active",
      "created_at": "2026-09-25T15:30:00Z",
      "resolved_at": null
    }
  ],
  "total_items": 5,
  "total_pages": 1,
  "current_page": 1,
  "page_size": 10
}
```

#### GET `/api/alerts/history`

Retrieve full alert history (acknowledged and solved alerts) with pagination.

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `page` | integer | No | 1 | Page number |
| `page_size` | integer | No | 10 | Items per page |
| `device_id` | string | No | "" | Filter by device identifier |
| `status` | string | No | "" | Filter by status: `acknowledged`, `solved` |

**Response:** Same structure as `/api/alerts`.

#### GET `/api/alerts/severity-counts`

Return aggregated severity counts grouped by alert status (`active`, `acknowledged`, `solved`). Used by frontend to display severity totals on each alert tab.

**Response 200 OK:**
```json
{
  "active": {
    "critical": 2,
    "warning": 5,
    "medium": 1
  },
  "acknowledged": {
    "critical": 1,
    "warning": 3
  },
  "solved": {
    "critical": 10,
    "warning": 25,
    "medium": 8
  }
}
```

**SQL Logic:** Groups by `status, severity`; distributes counts into `active`, `acknowledged`, or `solved` map based on status value.

#### POST `/api/alerts/{id}/acknowledge`

Mark an alert as acknowledged. Transitions alert from `active` to `acknowledged` state.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Alert internal identifier |

**Required Headers:**

| Header | Description |
|--------|-------------|
| `X-User-Email` | Email of acknowledging operator (defaults to `admin@email.com` if omitted) |

**Request Body:** None.

**Response:** No content (204 equivalent via redirect to `/alerts`).

**Database Effect:**
```sql
UPDATE alerts 
SET status = 'acknowledged', 
    acknowledged_by = $1 
WHERE id = $2 AND status = 'active'
```

**WebSocket Effect:** Broadcasts `{ id, status: 'acknowledged', updated }` to all connected clients.

#### POST `/api/alerts/{id}/solve`

Mark an alert as solved with resolution documentation. Transitions alert to `solved` state and sets `resolved_at` timestamp.

**Path Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `id` | UUID | Alert internal identifier |

**Required Headers:**

| Header | Description |
|--------|-------------|
| `X-User-Email` | Email of solving operator (defaults to `admin@email.com` if omitted) |

**Request Body:**
```json
{
  "notes": "Fuel tank refilled. Valve inspected and confirmed sealing properly.",
  "user_email": "operator@intecs.com"
}
```

**Response:** Redirect to `/alerts` (302).

**Database Effect:**
```sql
UPDATE alerts 
SET status = 'solved', 
    resolved_at = NOW(), 
    solved_by = $1, 
    solve_notes = $2 
WHERE id = $3 AND status IN ('active', 'acknowledged')
```

**WebSocket Effect:** Broadcasts `{ id, status: 'solved', updated }` to all connected clients. Resets cooldown map entry for affected `device_id:alert_type`.

### 8.5 Threshold Endpoints

#### GET `/api/thresholds`

List per-device threshold configurations.

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `device_id` | string | No | "" | Filter for specific device config |

**Response 200 OK (single device):**
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

**Response 200 OK (all devices — when `device_id` param omitted):** Array of threshold objects.

#### POST `/api/thresholds`

Create or update per-device threshold configuration (upsert semantics).

**Request Body:**
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

**Response 200 OK:** Updated threshold object with `created_at` and `updated_at` timestamps.

### 8.6 WebSocket Endpoint

#### GET `/api/ws`

Upgrade HTTP connection to WebSocket protocol.

**Handshake Flow:**
1. Client sends `GET /api/ws` with `Upgrade: websocket` and `Connection: Upgrade` headers
2. Server responds `101 Switching Protocols`
3. Both parties transition to full-duplex message exchange
4. Server immediately sends `mqtt_status` message with current broker state
5. Client begins sending periodic `{ type: "ping" }` heartbeats

**Max Message Size:** 32 KB per frame (configured via `SetReadLimit`).

### 8.7 MQTT Status Endpoint

#### GET `/api/mqtt/status`

Check MQTT broker connection health.

**Response 200 OK:**
```json
{
  "connected": true,
  "last_msg_at": "2026-09-25T15:30:00Z"
}
```

**Response 200 OK (disconnected):**
```json
{
  "connected": false,
  "last_msg_at": "2026-09-25T14:00:00Z"
}
```

---

## 9. Configuration

### 9.1 Environment Variables

| Variable | Type | Required | Default | Service(s) | Description |
|----------|------|----------|---------|------------|-------------|
| `DATABASE_URL` | string | Yes | — | Backend | PostgreSQL connection string format: `postgresql://user:password@host:5432/dbname?sslmode=disable` |
| `MQTT_BROKER_URL` | string | Yes | — | Backend, Simulator | MQTT broker address: `tcp://host:port` |
| `MQTT_USERNAME` | string | No | — | Backend, Simulator | Broker authentication username |
| `MQTT_PASSWORD` | string | No | — | Backend, Simulator | Broker authentication password |
| `DEVICE_COUNT` | integer | No | 10 | Simulator | Number of simulated devices (DT-001 to DT-N) |
| `PUBLISH_INTERVAL` | duration | No | 5s | Simulator | Publishing cadence: Golang duration string (e.g., `5s`, `10s`, `1m`) |
| `SITE_ID` | string | No | sangatta | Backend, Simulator | Default site identifier used in MQTT topic path |
| `HIGH_TEMP_THRESHOLD` | float | No | 90 | Backend | System-wide critical temperature alert threshold in °C |

### 9.2 Default Alert Thresholds

Used when no per-device configuration exists in `device_thresholds` table.

| Parameter | Value | Unit | Condition Type | Alert Type Generated |
|-----------|-------|------|---------------|---------------------|
| `warn_temp_threshold` | 85 | °C | `temperature > 85` | `HIGH_TEMPERATURE` (warning) |
| `crit_temp_threshold` | 90 | °C | `temperature > 90` | `CRITICAL_TEMPERATURE` (critical) |
| `warn_fuel_threshold` | 20 | % | `fuel_percentage < 20` | `LOW_FUEL` (warning) |
| `crit_fuel_threshold` | 10 | % | `fuel_percentage < 10` | `CRITICAL_FUEL` (critical) |
| `cooldown_seconds` | 30 | seconds | N/A | Min interval between repeated alerts |

### 9.3 WebSocket Timeout Configuration

| Setting | Value | Location | Purpose |
|---------|-------|----------|---------|
| `WriteWait` | 10 seconds | `websocket_handler.go` | Max time before write to client |
| `ReadWait` | 90 seconds | `hub.go` (via `SetReadDeadline`) | Max idle time before disconnection |
| `PingPeriod` | 30 seconds | Derived from `ReadWait` and `WriteWait` | How often to send PING frames |
| `MaxMessageSize` | 32 KB | `websocket_handler.go` (`SetReadLimit`) | Largest allowed incoming message |

### 9.4 Nginx WebSocket Proxy Settings

Required for production deployments behind reverse proxy.

| Directive | Value | Purpose |
|-----------|-------|---------|
| `proxy_http_version` | `1.1` | Required for WebSocket upgrade handshake |
| `proxy_set_header Upgrade` | `$http_upgrade` | Passes `Upgrade: websocket` header to backend |
| `proxy_set_header Connection` | `"upgrade"` | Signals WebSocket upgrade intent |
| `proxy_read_timeout` | `3600s` | Prevents proxy from closing long-lived WebSocket connections |
| `proxy_send_timeout` | `3600s` | Same as above |

---

## 10. Deployment

### 10.1 Docker Compose Topology

| Service | Image / Build | Ports | Environment | Depends On | Health Check |
|---------|--------------|-------|-------------|------------|--------------|
| `postgres` | `postgres:16-alpine` | `5432:5432` | `POSTGRES_USER=intecs`, `POSTGRES_PASSWORD=intecs123`, `POSTGRES_DB=intecs` | None | `pg_isready -U intecs` (interval: 10s, timeout: 5s, retries: 5) |
| `hivemq` | `hivemq/hivemq4:latest` | `1883:1883`, `8888:8888`, `8000:8000`, `9091:8080` | `HIVEMQ_ALLOW_ALL_CLIENTS=false` | None | `curl -sf http://localhost:8080/v2/login` (interval: 15s, timeout: 10s, retries: 5, start_period: 60s) |
| `backend` | Build from `./backend` | `8080:8080` | All variables from Section 9.1 | postgres (healthy), hivemq (started) | None configured |
| `simulator` | Build from `./simulator` | (internal — no exposed ports) | MQTT_BROKER_URL, MQTT_USERNAME, MQTT_PASSWORD, DEVICE_COUNT, PUBLISH_INTERVAL, SITE_ID | hivemq (started) | None configured |
| `frontend` | Build from `./frontend` | `5173:80` | `VITE_API_URL=http://localhost:8080/api` | backend | None configured |

### 10.2 Startup Sequence

```
Timeline    Action
──────      ──────
T+0s        docker compose up -d initiated
T+0-10s     postgres container starts; pg_isready passes health check
T+0-25s     hivemq container starts; accepts MQTT connections; admin UI available
T+10-30s    backend container starts; runs database.Migrate()
                ├─ Creates tables if missing
                ├─ Creates indexes if missing
                └─ Seeds 'sangatta' site if not exists
T+10-30s    Backend connects to PostgreSQL and establishes MQTT subscription
T+30-35s    simulator container starts; begins publishing telemetry
T+35s+      All services operational; dashboard accessible at http://localhost:5173
```

### 10.3 Volume Mounts and Persistence

| Volume Name | Mount Point (Container) | Purpose | Persistence Across Restarts |
|-------------|------------------------|---------|----------------------------|
| `postgres_data` | `/var/lib/postgresql/data` | PostgreSQL WAL and table files | Yes — survives `docker compose down` (without `-v`) |

**Transient Data (Lost on Container Removal):**
- Backend in-memory cooldown map (reset on restart)
- WebSocket hub client registry (connections re-establish)
- Frontend build artifacts (rebuilt on next build)

### 10.4 Production Hardening Checklist

- [ ] Replace hardcoded credentials with secrets management (HashiCorp Vault, AWS Secrets Manager)
- [ ] Restrict WebSocket `CheckOrigin` to known frontend domains
- [ ] Enable HTTPS/WSS with valid TLS certificates (Let's Encrypt or internal CA)
- [ ] Add `backend` service health check endpoint (e.g., `GET /api/health`)
- [ ] Configure log rotation for all containers (`logging.driver` in docker-compose)
- [ ] Set resource limits per container (`deploy.resources.limits` in Compose v3+)
- [ ] Add database backup strategy (pg_dump cron job or continuous archiving)
- [ ] Implement proper authentication backend (JWT signing with RS256, bcrypt password hashing)
- [ ] Add rate limiting on REST endpoints (e.g., nginx limit_req or middleware)
- [ ] Enable CORS explicitly for known frontend origins
- [ ] Enable MQTT over TLS (change `tcp://` to `ssl://` in broker config)
- [ ] Set up Prometheus metrics endpoints and Grafana dashboards
- [ ] Configure centralized logging (ELK stack or Loki + Grafana)

---

## 11. Appendices

### Appendix A: WebSocket Hub Architecture Details

The WebSocket hub implements a concurrent-safe fan-out pattern:

```
                    ┌──────────────┐
                    │   WebSocket  │
                    │     Hub      │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼───┐  ┌────▼───┐  ┌────▼───┐
         │ Client │  │ Client │  │ Client │
         │   A    │  │   B    │  │   C    │
         └────┬───┘  └────┬───┘  └────┬───┘
              │            │            │
         ┌────▼───┐  ┌────▼───┐  ┌────▼───┐
         │readPump│  │readPump│  │readPump│
         │writePump│ │writePump│ │writePump│
         └────────┘  └────────┘  └────────┘
```

**Channel-Based Communication:**
- `client.send` channel (buffer size 256): Queues messages for writePump to flush
- `hub.broadcast` channel: Fan-out destination; readPump writes here for all-client broadcasts
- `hub.register` channel: Client registration requests
- `hub.unsubscribe` channel: Client removal requests

**Concurrency Safety:**
- Hub struct locked via mutex for `clients` map modifications
- Each client managed by isolated goroutine pair (readPump + writePump)
- No shared mutable state between client goroutines

### Appendix B: Alert State Machine

```
         ┌──────────┐
         │  ACTIVE  │◄──────────────────────────┐
         └────┬─────┘                           │
              │                                  │
    POST /acknowledge                   Condition
    status='acknowledged'               re-triggers
              │                          (after cooldown
              ▼                           reset)
         ┌──────────────┐                ┌──────────┐
         │ ACKNOWLEDGED │                │  SOLVED  │
         └──────┬───────┘                └──────────┘
                │
       POST /solve
       status='solved'
                │
                ▼
         ┌──────────────┐
         │ RESOLVED     │
         │ (final state)│
         └──────────────┘
```

**Transition Rules:**
- `active` → `acknowledged`: Allowed; requires `X-User-Email` header; WHERE clause enforces `status='active'`
- `active` → `solved`: Allowed directly (bypass acknowledgment); requires notes and user email
- `acknowledged` → `solved`: Allowed; represents post-acknowledgment resolution
- `solved` → any: Not allowed; solved is terminal/final state

### Appendix C: Telemetry Field Validation

| Field | Expected Type | Validation Rules | Default if Missing |
|-------|--------------|------------------|-------------------|
| `device_id` | string, non-empty | Must match existing or new device pattern | Reject message |
| `fuel_percentage` | float, 0–100 | Numeric check; range optional | Persisted as NULL |
| `fuel_level` | float, ≥ 0 | Numeric check | Persisted as NULL |
| `temperature` | float | Any numeric value accepted | Persisted as NULL |
| `flow_rate` | float, ≥ 0 | Numeric check | Persisted as NULL |
| `equipment_status` | enum | Must be one of: running, idle, maintenance | Persisted as NULL |
| `timestamp` | ISO 8601 | Parseable datetime; timezone-aware preferred | Uses `NOW()` (server time) |

### Appendix D: Smart Pagination Algorithm

For large result sets (> 100 pages), the frontend renders compressed page ranges:

```
Example: total_pages = 200, current_page = 42

Rendered: « ‹ 1 ... 40 41 [42] 43 44 ... 200 › »

Rules:
- Always show first page (1) and last page (total_pages)
- Show ±1 pages around current_page
- Insert ellipsis (⋯) when gap exceeds 1 page
- Hide numeric buttons; replace with "Previous"/"Next" when > 100 pages
```
