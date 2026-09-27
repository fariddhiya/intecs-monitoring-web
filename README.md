# INTECS IoT Monitoring Web Application

Real-time industrial equipment monitoring system for tracking fuel levels, temperature, flow rates, and equipment status across multiple devices using MQTT telemetry ingestion and a Svelte-based web dashboard.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Tech Stack](#tech-stack)
- [MQTT Protocol](#mqtt-protocol)
- [System Requirements](#system-requirements)
- [Quick Start (Docker Compose)](#quick-start-docker-compose-recommended)
- [Local Development Setup](#local-development-setup)
- [Project Structure](#project-structure)
- [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Alert Rules & Thresholds](#alert-rules--thresholds)
- [Dashboard Screens](#dashboard-screens)
- [Security Considerations](#security-considerations)
- [Known Limitations](#known-limitations)
- [Production Roadmap](#production-roadmap)

---

## Overview

INTECS is an end-to-end monitoring platform designed for industrial and mining environments. It ingests real-time sensor data from equipment via MQTT, evaluates threshold conditions to generate alerts, and provides operators with live dashboards, historical trend analysis, and CSV reporting capabilities.

### Key Capabilities

- **Real-Time Fleet Monitoring** — Live dashboard showing all connected devices with fuel %, temperature, flow rate, and connectivity status
- **Intelligent Alerting** — Automatic detection of low fuel, high temperature, and offline device conditions with configurable thresholds and cooldown deduplication
- **Historical Visualization** — Interactive time-series charts (Chart.js) with multi-range selectors (1h, 6h, 24h, 7d)
- **WebSocket Push Notifications** — Instant alert and telemetry updates to all connected browsers
- **Multi-Site Support** — Abstracted site hierarchy supporting distributed operations
- **CSV Export** — One-click export for device inventory, telemetry history, and alert records

---

## Architecture

```mermaid
graph LR
    subgraph IoT_Layer["IoT Layer"]
        A[Simulated Devices<br/>DT-001 → DT-010] -->|MQTT QoS 1| B[HiveMQ Broker<br/>port 1883]
    end

    subgraph Backend_Layer["Backend Layer (Go)"]
        B -->|Subscribe: intecs/site/+/device/+/telemetry| C[MQTT Client]
        C --> D[Device Manager]
        C --> E[Alert Engine]
        D --> F[(PostgreSQL 16)]
        E --> F
        D --> F
        C --> F
        G[REST API Server<br/>port 8080] --> F
        H[WebSocket Hub] --> F
    end

    subgraph Frontend_Layer["Frontend Layer (SvelteKit)"]
        I[Web Dashboard<br/>port 5173/Nginx] -->|REST JSON| G
        I -->|WebSocket /api/ws| H
    end

    style IoT_Layer fill:#e1f5fe
    style Backend_Layer fill:#fff3e0
    style Frontend_Layer fill:#e8f5e9
```

### Data Flow

1. **Simulated/Physical Devices** publish telemetry payloads to HiveMQ every 5 seconds
2. **Backend MQTT Client** subscribes to wildcard topic `intecs/site/+/device/+/telemetry`
3. **Message Processing Pipeline:**
   - Extract `site_id` and `device_id` from topic path
   - Upsert site and device records in PostgreSQL
   - Persist telemetry reading to `telemetry` table
   - Evaluate threshold conditions against alert rules
   - Insert alert if triggered (with cooldown deduplication)
   - Broadcast event via WebSocket to all connected clients
4. **Frontend Dashboard** receives updates via WebSocket and auto-refreshes UI components
5. **Operators** interact through REST API for filtering, paginating, exporting, and alert management

---

## Features

### Dashboard (`/`)
- Aggregate statistics cards: Total Devices, Online, Offline, Active Alerts
- Paginated device table with client-side search and server-side filters
- Connection status indicators: 🟢 ONLINE (< 1 min), 🟡 STALE (1–5 min), 🔴 OFFLINE (> 5 min)
- Equipment status badges: Running, Idle, Maintenance
- Color-coded fuel gauge bars (Red <10%, Orange 10-20%, Yellow 20-50%, Green >50%)
- Multi-column sorting (click headers)
- Configurable page sizes: 5, 10, 15 items
- Smart pagination with ellipsis for large datasets
- Bulk CSV export of current view
- Auto-polling every 5 seconds + WebSocket live updates
- MQTT broker connection status bar (connected/disconnected + last message timestamp)

### Device Detail (`/devices/[id]`)
- Real-time metrics display: Fuel %, Fuel Level, Temperature °C, Flow Rate L/min
- Two interactive Chart.js visualizations:
  - Fuel Percentage History (green area chart)
  - Temperature History (blue area chart)
- Time range selector: 1h, 6h, 24h, 7d
- Historical polling (15s interval) + WebSocket stream overlay
- Live pulsing indicator dot while streaming
- CSV export with auto-generated filename `{deviceId}_{range}_{date}.csv`
- Error recovery with retry button after 3 consecutive failures

### Alert Management (`/alerts`)
- Three-state tabs: Active, Acknowledged, Solved
- Severity classification: Critical (🔴 red), Warning (🟠 orange), Medium (🔵 blue)
- Per-tab pagination independent of each other
- Severity filter chips per tab (Critical/Warning/Medium)
- Relative timestamps: "Just now", "2m ago", "1h ago", "3d ago"
- Acknowledge action: POST to mark as acknowledged (requires user email)
- Solve action: POST with required resolution notes
- Browser notification support via `Notification.requestPermission()`
- Auto-refresh: Active (10s), Acknowledged (10s), Solved (60s)
- Severity count aggregation per tab (fetched from `/api/alerts/severity-counts`)
- Separate CSV export buttons per tab state

### Authentication
- Single admin account: `admin@email.com` / `pass`
- JWT-like token stored in `localStorage` with 24-hour expiry
- Route guard on all pages redirecting to `/login` if expired
- Logout clears session storage and redirects to login

### Theme Management
- Dark/Light theme toggle persisted in `localStorage.intecs-theme`
- CSS custom properties swap values via `[data-theme="dark"]` selector
- Theme applied immediately on mount from stored preference

---

## Tech Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Backend API | Go | 1.23+ |
| MQTT Broker | HiveMQ | 4.x |
| Database | PostgreSQL | 16 |
| Frontend Framework | SvelteKit | 2.x |
| Frontend UI | Svelte | 4.x |
| Build Tool | Vite | 5.x |
| Charts | Chart.js | 4.x |
| Containerization | Docker Compose | — |
| HTTP Client | Gorilla WebSocket | — |
| MQTT Client | Eclipse Paho MQTT Go | — |
| Router | Go Mux | — |

---

## MQTT Protocol

### Topic Pattern

```
Publish:     intecs/site/{site_id}/device/{device_id}/telemetry
Subscribe:   intecs/site/+/device/+/telemetry
```

**Example:**
```
intecs/site/sangatta/device/DT-001/telemetry
```

### Message Format

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

### MQTT Client Configuration

| Parameter | Value | Description |
|-----------|-------|-------------|
| Client ID | `intecs-backend` | Identifier for backend subscriber |
| Keep Alive | 30 seconds | Heartbeat interval |
| Auto Reconnect | Enabled | Automatic reconnection on disconnect |
| Clean Session | true | Start fresh on reconnect |
| QoS | 1 (At least once) | Delivery guarantee |

### WebSocket Messages (Server → Client)

| Type | Payload |
|------|---------|
| `telemetry` | `{ device_id, timestamp, fuel_percent, fuel_level, temperature, flow_rate, equipment_status }` |
| `alert` | `{ device_id, type, severity, message }` |
| `mqtt_status` | `{ connected: boolean, last_msg_at: string }` |
| `error` | `{ message: string }` |

### WebSocket Client Behavior
- Heartbeat ping every 30 seconds (`{ type: "ping" }`)
- Exponential backoff reconnection: `min(1000 × 2^attempt, 60000)` ms, max 20 attempts
- Message routing by `msg.type`: `mqtt_status` → status bar, `telemetry` → CustomEvent dispatch, `alert` → CustomEvent dispatch

---

## System Requirements

### Hardware (Minimum)

| Component | Specification |
|-----------|--------------|
| CPU | 2 vCPU |
| RAM | 4 GB |
| Disk | 20 GB |
| Network | Internal VLAN (< 50ms RTT recommended) |

### Software Dependencies

| Dependency | Purpose |
|------------|---------|
| Docker | Container runtime |
| Docker Compose | Multi-container orchestration |
| Git | Version control (for local dev build) |

---

## Quick Start (Docker Compose, Recommended)

### Prerequisites

- Docker 24+ installed
- Docker Compose v2+ installed
- Port availability: 5173, 8080, 5432, 1883, 8000

### Steps

```bash
# 1. Clone repository
git clone <repo-url>
cd intecs-monitoring-web

# 2. Copy environment configuration
cp .env.example .env

# 3. Start all services
docker compose up -d

# 4. Verify services
docker compose ps
```

### Service Access Points

| Service | URL | Port |
|---------|-----|------|
| **Dashboard** | http://localhost:5173 | 5173 |
| **REST API** | http://localhost:8080/api | 8080 |
| **HiveMQ Admin** | http://localhost:8000 | 8000 |
| **HiveMQ MQTT** | tcp://localhost:1883 | 1883 |
| **PostgreSQL** | postgresql://intecs:intecs123@localhost:5432/intecs | 5432 |

### Default Login

- **Email:** `admin@email.com`
- **Password:** `pass`

### Stop Services

```bash
docker compose down          # Remove containers
docker compose down -v       # Remove containers + volumes (destructive)
```

---

## Local Development Setup

### 1. Start Infrastructure Only

```bash
docker compose up postgres hivemq
```

Wait for health checks to pass (~30 seconds).

### 2. Start Backend

```bash
cd backend
go mod download
go run ./cmd/main.go
```

Backend starts on `http://localhost:8080`.

### 3. Start Simulator

```bash
cd simulator
go mod download
go run ./main.go
```

Simulator publishes telemetry for 10 devices at 5-second intervals.

### 4. Start Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend serves on `http://localhost:5173` with hot module replacement.

### 5. Run All Four Services Together

```bash
# Terminal 1: Infrastructure
docker compose up postgres hivemq

# Terminal 2: Backend
cd backend && go run ./cmd/main.go

# Terminal 3: Simulator
cd simulator && go run ./main.go

# Terminal 4: Frontend
cd frontend && npm run dev
```

---

## Project Structure

```
intecs-monitoring-web/
├── backend/                      # Go backend service
│   ├── cmd/
│   │   └── main.go              # Application entry point
│   ├── internal/
│   │   ├── api/                 # REST API handlers
│   │   │   ├── handler.go       # HTTP request handlers (alerts, devices, dashboard)
│   │   │   └── websocket_handler.go  # WebSocket upgrade handler
│   │   ├── alert/               # Alert evaluation engine
│   │   │   └── manager.go       # Threshold checking, cooldown deduplication
│   │   ├── database/            # PostgreSQL operations
│   │   │   └── database.go      # Migration, seeding, schema definitions
│   │   ├── device/              # Device state management
│   │   │   └── manager.go       # Upsert, status computation, last_seen tracking
│   │   ├── mqtt/                # MQTT client implementation
│   │   │   ├── handler.go       # Message processing pipeline
│   │   │   └── types.go         # Telemetry payload structures
│   │   └── websocket/           # WebSocket hub
│   │       └── hub.go           # readPump/writePump, broadcast, client management
│   └── Dockerfile
├── frontend/                     # SvelteKit dashboard
│   ├── src/
│   │   ├── lib/
│   │   │   └── components/      # Reusable UI components
│   │   │       ├── AlertList.svelte    # Alert management with tabs/filters
│   │   │       ├── Dashboard.svelte    # Main dashboard with device table
│   │   │       └── DeviceDetail.svelte # Device detail page with charts
│   │   └── routes/
│   │       ├── +layout.svelte     # Global layout, auth guard, theme
│   │       ├── +page.svelte       # Dashboard route (/)
│   │       ├── login/             # Login page
│   │       ├── devices/           # Device list/detail routes
│   │       └── alerts/            # Alert management route
│   └── Dockerfile
├── simulator/                    # IoT device simulator
│   ├── main.go                  # Simulates 10 industrial devices
│   └── Dockerfile
├── docker-compose.yml           # Multi-service orchestration
├── .env.example                 # Environment variable template
├── BRD.md                       # Business Requirements Document
├── SRS.md                       # Software Requirements Specification
└── README.md                    # This file
```

---

## Environment Variables

### Docker Compose Services

```yaml
backend:
  DATABASE_URL: postgresql://intecs:intecs123@postgres:5432/intecs?sslmode=disable
  MQTT_BROKER_URL: tcp://hivemq:1883
  MQTT_USERNAME: intecs
  MQTT_PASSWORD: intecs123
  DEVICE_COUNT: "10"
  PUBLISH_INTERVAL: 5s
  SITE_ID: sangatta
  HIGH_TEMP_THRESHOLD: "90"

simulator:
  MQTT_BROKER_URL: tcp://hivemq:1883
  MQTT_USERNAME: intecs
  MQTT_PASSWORD: intecs123
  DEVICE_COUNT: "10"
  PUBLISH_INTERVAL: 5s
  SITE_ID: sangatta

frontend:
  VITE_API_URL: http://localhost:8080/api
```

### Variable Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `MQTT_BROKER_URL` | Yes | — | MQTT broker address (tcp://host:port) |
| `MQTT_USERNAME` | No | — | Broker authentication username |
| `MQTT_PASSWORD` | No | — | Broker authentication password |
| `DEVICE_COUNT` | No | `10` | Number of simulated devices |
| `PUBLISH_INTERVAL` | No | `5s` | Simulator publish cadence |
| `SITE_ID` | No | `sangatta` | Default site identifier |
| `HIGH_TEMP_THRESHOLD` | No | `90` | System-wide critical temperature threshold |

---

## API Reference

### Base URLs

- **REST API:** `http://localhost:8080/api`
- **WebSocket:** `ws://localhost:8080/api/ws`
- **MQTT Status:** `http://localhost:8080/api/mqtt/status`

### Endpoints

#### Dashboard

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

#### Devices

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/devices` | List devices with pagination, search, filter, sort |
| GET | `/api/devices/{id}` | Get single device detail |
| GET | `/api/devices/{id}/telemetry` | Get historical telemetry points |

**Query Parameters (`GET /api/devices`):**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | integer | 1 | Page number (1-indexed) |
| `page_size` | integer | 5 | Items per page (valid: 5, 10, 15+) |
| `search` | string | "" | Filter by device_id substring |
| `connection` | string | "" | Filter: ONLINE, STALE, OFFLINE |
| `equipment_status` | string | "" | Filter: running, idle, maintenance |
| `sort` | string | "" | Sort column name |
| `dir` | string | desc | Sort direction: asc, desc |

**Query Parameters (`GET /api/devices/{id}/telemetry`):**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `range` | string | 24h | Time window: 1h, 6h, 24h, 7d |
| `limit` | integer | 500 | Maximum data points returned |

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

#### Alerts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/alerts` | List active/acknowledged/solved alerts with pagination |
| GET | `/api/alerts/history` | Full alert history with pagination |
| GET | `/api/alerts/severity-counts` | Get severity counts per status (active/acknowledged/solved) |
| POST | `/api/alerts/{id}/acknowledge` | Mark alert as acknowledged (header: `X-User-Email`) |
| POST | `/api/alerts/{id}/solve` | Mark alert as solved (requires resolution notes in body) |

**Query Parameters (`GET /api/alerts`):**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `page` | integer | 1 | Page number |
| `page_size` | integer | 10 | Items per page |
| `status` | string | active | Filter: active, acknowledged, solved |

**Request Body (`POST /api/alerts/{id}/solve`):**
```json
{
  "notes": "Fuel refilled and valve checked",
  "user_email": "admin@email.com"
}
```

**Severity Counts Response:**
```json
{
  "active": { "critical": 2, "warning": 5 },
  "acknowledged": { "critical": 1, "warning": 3 },
  "solved": { "critical": 10, "warning": 25 }
}
```

#### Thresholds

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/thresholds` | List per-device or global thresholds |
| POST | `/api/thresholds` | Create or update per-device thresholds |

**Query Parameters (`GET /api/thresholds`):**

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `device_id` | string | "" | Filter for specific device config |

**Request Body (`POST /api/thresholds`):**
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

#### WebSocket

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/ws` | Upgrade to WebSocket connection |

#### MQTT Status

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/mqtt/status` | Check MQTT broker connection status |

**Response:**
```json
{
  "connected": true,
  "last_msg_at": "2026-09-25T15:30:00Z"
}
```

---

## Database Schema

### ER Diagram

```mermaid
erDiagram
    SITES ||--o{ DEVICES : has
    DEVICES ||--o{ TELEMETRY : generates
    DEVICES ||--o{ ALERTS : triggers
    DEVICES ||--o{ DEVICE_THRESHOLDS : configures

    SITES {
        UUID id PK
        TEXT name
        TEXT site_id UK
        TIMESTAMP created_at
    }

    DEVICES {
        UUID id PK
        TEXT device_id UK
        UUID site_id FK
        TEXT name
        TEXT type
        TEXT status
        TIMESTAMP last_seen
        TIMESTAMP created_at
    }

    TELEMETRY {
        UUID id PK
        TEXT device_id
        TIMESTAMP timestamp
        NUMERIC fuel_percentage
        NUMERIC fuel_level
        NUMERIC temperature
        NUMERIC flow_rate
        TEXT equipment_status
    }

    ALERTS {
        UUID id PK
        TEXT device_id
        TEXT type
        TEXT severity
        TEXT message
        TEXT status
        TEXT acknowledged_by
        TEXT solved_by
        TEXT solve_notes
        TIMESTAMP created_at
        TIMESTAMP resolved_at
    }

    DEVICE_THRESHOLDS {
        UUID id PK
        TEXT device_id UK
        NUMERIC warn_temp_threshold
        NUMERIC crit_temp_threshold
        NUMERIC warn_fuel_threshold
        NUMERIC crit_fuel_threshold
        INT cooldown_seconds
        TIMESTAMP created_at
        TIMESTAMP updated_at
    }
```

### Indexes

| Table | Index | Purpose |
|-------|-------|---------|
| `telemetry` | `idx_telemetry_device_timestamp ON (device_id, timestamp)` | Optimized time-series queries |
| `alerts` | `idx_alerts_device ON (device_id)` | Device alert lookup |
| `alerts` | `idx_alerts_status ON (status)` | Active alert filtering |
| `devices` | `idx_alerts ON (device_id)` | Device reference lookups |

### Alert States

| State | `status` Column | `resolved_at` Column |
|-------|-----------------|---------------------|
| **Active** | `'active'` | `NULL` | New alerts just created |
| **Acknowledged** | `'acknowledged'` | `NULL` | Operator clicked "Acknowledge" |
| **Solved** | `'solved'` | Set to current timestamp | Operator clicked "Solve" with notes |

---

## Alert Rules & Thresholds

### Default Alert Conditions

| Alert Type | Condition | Severity | Default Threshold | Cooldown |
|------------|-----------|----------|-------------------|----------|
| `LOW_FUEL` | `fuel_percentage < warn_fuel_threshold` | Warning | 20% | 30 seconds |
| `CRITICAL_FUEL` | `fuel_percentage < crit_fuel_threshold` | Critical | 10% | 30 seconds |
| `HIGH_TEMPERATURE` | `temperature > warn_temp_threshold` | Warning | 85°C | 30 seconds |
| `CRITICAL_TEMPERATURE` | `temperature > crit_temp_threshold` | Critical | 90°C | 30 seconds |
| `DEVICE_OFFLINE` | `last_seen > 5 minutes ago` | Medium | N/A | N/A |

### Device Connectivity Status Rules

| Status | Condition | Time Window | Visual Indicator |
|--------|-----------|-------------|-----------------|
| **ONLINE** | Recent telemetry received | `< 1 minute` ago | 🟢 Green badge with pulse |
| **STALE** | Telemetry delayed | `1–5 minutes` ago | 🟡 Yellow badge |
| **OFFLINE** | No telemetry reported | `> 5 minutes` ago | 🔴 Red badge |

### Threshold Deduplication Logic

- In-memory map keyed by `"deviceID:alertType"` prevents duplicate alerts
- Cooldown period configured per device (default: 30 seconds)
- Reset cooldown when alert is acknowledged or solved
- Falls back to system defaults if no per-device thresholds exist

---

## Dashboard Screens

### Login Page (`/login`)
- Minimal form with email and password fields
- Hardcoded credentials: `admin@email.com` / `pass`
- Token stored in localStorage on successful auth

### Dashboard Home (`/`)
- Stats cards row: Total Devices, Online, Offline, Active Alerts
- MQTT status bar: Connected/Disconnected indicator with last message timestamp
- Search input for device filtering
- Filter dropdowns: Connection Status, Equipment Status
- Paginated device table with sortable columns
- Fuel level color bars embedded in table
- Export CSV button

### Device Detail (`/devices/[id]`)
- Header with device metadata: Name, Site, Device ID, Last Seen
- Current readings panel: Fuel %, Fuel Level, Temperature, Flow Rate, Equipment Status
- Connection status badge
- Two Chart.js charts side-by-side: Fuel History, Temperature History
- Time range selector buttons: 1h, 6h, 24h, 7d
- Export telemetry CSV button

### Alert Management (`/alerts`)
- Tab navigation: Active, Acknowledged, Solved
- Severity filter chips per tab: Critical, Warning, Medium
- Alert list with severity icons, device info, relative timestamps
- Acknowledge button (Active tab only)
- Solve modal dialog (requires notes input)
- Separate pagination controls per tab

### Theme Toggle
- Dark/Light mode switch in global header
- Icon toggles between sun/moon
- Preference persisted in localStorage

---

## Security Considerations

### Current Implementation (Development)

- Environment variables for all credentials
- Input validation on all REST endpoints
- Parameterized SQL queries (prevents SQL injection)
- Origin validation on WebSocket enabled but returns true (allow all origins)

### Production Hardening Checklist

- [ ] Replace hardcoded credentials with secrets management (Vault, AWS Secrets Manager)
- [ ] Restrict WebSocket `CheckOrigin` to known domains
- [ ] Enable HTTPS/WSS with valid TLS certificates
- [ ] Add `backend` service health check endpoint
- [ ] Configure log rotation for all containers
- [ ] Set resource limits (CPU/memory) per container
- [ ] Add database backup strategy (pg_dump cron)
- [ ] Implement proper authentication backend (JWT signing, password hashing with bcrypt)
- [ ] Add rate limiting on REST endpoints
- [ ] Enable CORS explicitly for known frontend origins
- [ ] MQTT over TLS/mTLS for encrypted broker communication
- [ ] Device certificate-based authentication
- [ ] RBAC for admin/API access
- [ ] Audit logging for all alert state changes
- [ ] Network segmentation between MQTT and API layers

---

## Known Limitations

| Limitation | Impact | Future Solution |
|------------|--------|-----------------|
| Single-node deployment | No high availability | PostgreSQL streaming replication + multi-instance backend |
| Memory-only alert deduplication | Dedup state lost on restart | Persistent cooldown map in database |
| Basic authentication | No user management | Full auth backend with JWT, bcrypt, role assignment |
| PostgreSQL for telemetry | Not optimized for time-series | Migrate to TimescaleDB or InfluxDB |
| Single MQTT broker | Broker failure blocks ingestion | HiveMQ cluster or EMQX HA setup |
| No mobile application | Desktop-only interface | Progressive Web App (PWA) or native mobile app |
| Browser notifications only | Limited reach | Email, SMS, and push notification integration |
| Manual device provisioning | No automated onboarding | Device registration workflow with QR codes |
| No ML anomaly detection | Threshold-based only only | Predictive maintenance models |
| No CMMS/EAM integration | Siloed alert data | Integration with ServiceNow, IBM Maximo, etc. |

---

## Production Roadmap

### Phase 2: Production Readiness

1. **Authentication Overhaul**
   - JWT token signing with secret key
   - Password hashing with bcrypt
   - Role-based access control (Admin, Operator, Viewer)
   - Session management and refresh tokens

2. **Infrastructure Hardening**
   - PostgreSQL streaming replication for failover
   - MQTT broker clustering (HiveMQ Enterprise or EMQX)
   - Load balancer for horizontal API scaling
   - Redis cache for frequently accessed data
   - Prometheus metrics + Grafana dashboards

3. **Monitoring & Observability**
   - Distributed tracing (Jaeger/OpenTelemetry)
   - Structured logging (JSON format with correlation IDs)
   - Alerting on system health (uptime, error rates)
   - Database query performance monitoring

### Phase 3: Advanced Features

1. **Time-Series Database Migration**
   - Replace PostgreSQL telemetry table with TimescaleDB
   - Continuous aggregates for dashboard stats
   - Automated data retention policies

2. **Notification Channels**
   - Email notifications via SMTP
   - SMS alerts via Twilio
   - Slack webhook integration

3. **Device Provisioning**
   - Self-registration portal
   - QR code scan setup
   - Firmware OTA update capability

4. **Analytics**
   - Predictive maintenance with ML models
   - Fuel consumption optimization recommendations
   - Equipment lifetime estimation
   - Custom report builder
