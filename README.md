# IoT Fuel & Equipment Monitoring System

## Overview

A prototype IoT monitoring system for industrial/mining environments that tracks fuel levels, temperature, flow rates, and equipment status across multiple devices in real-time.

## Architecture

```mermaid
graph LR
    subgraph IoT Layer
        A[IoT Simulator<br/>DT-001 to DT-010] -->|MQTT| B[HiveMQ Broker]
    end
    
    subgraph Backend Layer
        B -->|Subscribe| C[Go Backend<br/>MQTT Consumer + REST API]
        C -->|Read/Write| D[(PostgreSQL)]
    end
    
    subgraph Frontend Layer
        E[Svelte Dashboard] -->|REST API| C
    end
```

**Production Scale:**
```
IoT Devices → MQTT Cluster → Ingestion Service → Message Queue → Time-series DB → API Cluster → Dashboard
```

## Business Use Case

Monitor industrial equipment (mining vehicles, generators, etc.) across sites:
- **Fuel Level**: Track percentage and liters in real-time
- **Temperature**: Alert on overheating equipment
- **Flow Rate**: Monitor fluid consumption
- **Equipment Status**: Running, Idle, or Maintenance
- **Connectivity**: Online, Stale, or Offline device detection
- **Alerts**: Low fuel, critical fuel, high temperature notifications

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.23+ |
| MQTT Broker | HiveMQ 4 |
| Database | PostgreSQL 16 |
| Frontend | SvelteKit 2 |
| Containerization | Docker Compose |
| Protocol | MQTT 5.0 / REST JSON |

## MQTT Topics

**Publish Topic Pattern:**
```
intecs/site/{site_id}/device/{device_id}/telemetry
```

**Example:**
```
intecs/site/sangatta/device/DT-001/telemetry
```

**Subscribed Topic (Backend):**
```
intecs/site/+/device/+/telemetry
```

**Payload Format:**
```json
{
  "device_id": "DT-001",
  "timestamp": "2026-09-25T15:30:00Z",
  "fuel_percentage": 72.5,
  "fuel_level": 7250,
  "temperature": 81.2,
  "flow_rate": 35.2,
  "equipment_status": "running"
}
```

## Database Schema

### tables

```sql
sites (id, name, site_id, created_at)
devices (id, device_id, site_id, name, type, status, last_seen, created_at)
telemetry (id, device_id, timestamp, fuel_percentage, fuel_level, temperature, flow_rate, equipment_status)
alerts (id, device_id, type, severity, message, status, created_at, resolved_at)
```

**Indexes:**
- `telemetry(device_id, timestamp)` - optimized time series queries
- `alerts(device_id)` - device alert lookup
- `alerts(status)` - active alert filtering

## How to Run

### Option 1: Docker Compose (Recommended)

```bash
cp .env.example .env
docker compose up
```

Services available at:
- **Dashboard**: http://localhost:5173
- **REST API**: http://localhost:8080/api
- **HiveMQ Admin**: http://localhost:8000
- **PostgreSQL**: localhost:5432

### Option 2: Local Development

**1. Start infrastructure:**
```bash
docker compose up postgres hivemq
```

**2. Start backend:**
```bash
cd backend && go run ./cmd/main.go
```

**3. Start simulator:**
```bash
cd simulator && go run ./main.go
```

**4. Start frontend:**
```bash
cd frontend && npm install && npm run dev
```

## Environment Variables

```bash
DATABASE_URL=postgresql://user:pass@host:5432/db
MQTT_BROKER_URL=tcp://localhost:1883
MQTT_USERNAME=intecs
MQTT_PASSWORD=intecs123
DEVICE_COUNT=10
PUBLISH_INTERVAL=5s
SITE_ID=sangatta
HIGH_TEMP_THRESHOLD=90
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dashboard` | Dashboard summary with all device statuses |
| GET | `/api/devices` | List all devices with current telemetry |
| GET | `/api/devices/:id` | Single device detail |
| GET | `/api/devices/:id/telemetry` | Historical telemetry for a device |
| GET | `/api/alerts` | List alerts (filter by `?status=active`) |
| POST | `/api/alerts/:id/acknowledge` | Acknowledge/resolve an alert |

## Dashboard Features

- **Summary Cards**: Total, online, offline devices + active alerts count
- **Device Table**: All devices with fuel gauge, temperature, flow rate, connection status
- **Color Indicators**: 
  - 🟢 Green: Online / High fuel (>50%)
  - 🟡 Yellow: Stale / Low fuel (10-20%)
  - 🔴 Red: Offline / Critical fuel (<10%)
- **Active Alerts Panel**: Latest alerts with acknowledge functionality
- **Auto-refresh**: Polls API every 5 seconds

## Device Detail Page

Access by clicking any device in the dashboard table. Shows:
- Current readings (fuel %, temperature, flow rate)
- Visual fuel level bar with color coding
- Connection status and equipment state
- Last seen timestamp
- Historical telemetry data

## Alert Rules

| Alert Type | Condition | Severity | Cooldown |
|------------|-----------|----------|----------|
| LOW_FUEL | fuel < 20% | Warning | 30s |
| CRITICAL_FUEL | fuel < 10% | Critical | 30s |
| HIGH_TEMPERATURE | temp > threshold | High | 30s |
| DEVICE_OFFLINE | no telemetry > 5 min | Medium | N/A |

Duplicate alerts are prevented within cooldown period.

## Security Considerations

**Current Implementation:**
- Environment variables for credentials
- MQTT authentication enabled
- Input validation on all endpoints
- No hardcoded passwords

**Production Enhancements Needed:**
- MQTT over TLS/mTLS
- Device certificate-based authentication
- RBAC for admin/API access
- Secret manager integration
- Network segmentation
- Audit logging

## Production Scalability

The prototype uses a single Go process. For production:

1. **Split Services**: Separate MQTT ingestion from REST API
2. **Time-series DB**: Replace PostgreSQL with TimescaleDB for telemetry data
3. **Message Queue**: Add Kafka/RabbitMQ between ingestion and processing
4. **Horizontal Scaling**: Multiple API instances behind load balancer
5. **High Availability**: PostgreSQL streaming replication, MQTT cluster
6. **Monitoring**: Prometheus metrics, distributed tracing

## Known Limitations

- Single-node deployment (no HA)
- No WebSocket support (polling-based dashboard)
- Basic authentication only
- PostgreSQL instead of time-series database
- Single MQTT broker (not clustered)
- No device provisioning workflow
- Memory-only alert state (no persistence for deduplication)

## Project Structure

```
project/
├── backend/           # Go backend (MQTT + API)
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── mqtt/      # MQTT message handling
│   │   ├── api/       # REST API handlers
│   │   ├── database/  # PostgreSQL operations
│   │   ├── device/    # Device state management
│   │   └── alert/     # Alert evaluation logic
│   └── Dockerfile
├── simulator/         # IoT device simulator
│   ├── main.go
│   └── Dockerfile
├── frontend/          # Svelte dashboard
│   ├── src/routes/
│   └── Dockerfile
├── docker-compose.yml
├── .env.example
└── README.md
```
