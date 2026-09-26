# Business Requirements Document (BRD)
## INTECS IoT Monitoring Web Application

| Field | Details |
|-------|---------|
| **Project Name** | INTECS IoT Monitoring Web |
| **Version** | 1.0 |
| **Status** | Draft |
| **Date** | 2026-09-26 |

---

## 1. Executive Summary

INTECS requires a real-time web-based monitoring platform for industrial equipment telemetry data collected via MQTT. The system ingests telemetry from simulated/real industrial devices, processes alert conditions, and presents operational dashboards to plant operators for continuous monitoring, historical analysis, and incident management.

## 2. Business Objectives

| # | Objective | Priority |
|---|-----------|----------|
| 1 | Provide real-time visibility into equipment health metrics (fuel, temperature, flow rate) across all monitored assets | P0 |
| 2 | Reduce equipment downtime by enabling proactive alerting on threshold violations | P0 |
| 3 | Enable operators to review historical telemetry trends for maintenance planning | P1 |
| 4 | Support scalable device onboarding without backend code changes | P1 |
| 5 | Maintain operational continuity through automatic recovery from connection failures | P1 |

## 3. Stakeholders

| Role | Responsibility |
|------|----------------|
| Plant Operators | Monitor dashboards, acknowledge alerts, export reports |
| Maintenance Teams | Review historical data, plan preventive maintenance |
| System Administrators | Manage infrastructure, configure thresholds, monitor system health |
| IoT Engineers | Design device firmware, manage MQTT broker, validate telemetry accuracy |

## 4. Business Use Cases

### UC-01: Real-Time Fleet Monitoring
- **Actor:** Plant Operator
- **Trigger:** Operator logs into the dashboard
- **Flow:** Dashboard displays live status of all connected devices including fuel percentage, temperature, flow rate, and connectivity state. Auto-refreshes every 5 seconds with WebSocket push for instant updates.
- **Business Value:** Operators maintain situational awareness of entire equipment fleet without manual checks.

### UC-02: Alert Detection and Response
- **Actor:** Plant Operator
- **Trigger:** Telemetry crosses defined thresholds
- **Flow:** System generates alert notification pushed via WebSocket. Operator views alert details, acknowledges the alert, and initiates response protocol. Alerts persist until resolved.
- **Business Value:** Reduces mean time to detection (MTTD) and resolution (MTTR) for critical equipment conditions.

### UC-03: Historical Trend Analysis
- **Actor:** Maintenance Team
- **Trigger:** Operator selects a specific device and time range
- **Fuel Percentage History:** Line chart showing fuel consumption over selected period (1h, 6h, 24h, 7d)
- **Temperature History:** Line chart showing thermal profile over selected period
- **CSV Export:** All telemetry data exportable for offline analysis
- **Business Value:** Enables predictive maintenance by identifying degradation patterns before failures occur.

### UC-04: Device Health Reporting
- **Actor:** System Administrator
- **Trigger:** Periodic review or event-driven investigation
- **Flow:** Full paginated device inventory with search, filter by connection/equipment status, client-side sorting. Bulk CSV export of current device states.
- **Business Value:** Supports compliance reporting and asset inventory management.

### UC-05: Customizable Alert Thresholds
- **Actor:** IoT Engineer / System Administrator
- **Trigger:** Different equipment types require different alert criteria
- **Flow:** Per-device configurable thresholds for high temperature (warn/critical), low fuel (warn/critical), and alert cooldown periods.
- **Business Value:** Eliminates false positives from one-size-fits-all alert rules.

## 5. Scope

### In Scope
- Real-time telemetry ingestion via MQTT
- PostgreSQL persistence for telemetry, devices, alerts, and thresholds
- RESTful API for data retrieval and configuration
- SvelteKit-based responsive web dashboard
- WebSocket-based real-time notifications
- Device detail page with interactive charts and CSV export
- Alert management with acknowledgment and history
- Multi-site support (site abstraction layer)
- Dark/light theme support
- Docker Compose deployment

### Out of Scope (Phase 1)
- User role management beyond single admin
- Email/push notification beyond browser notifications
- Automated device provisioning workflow
- Integration with external CMMS/EAM systems
- Mobile application
- Machine learning anomaly detection
- Multi-tenant architecture

## 6. High-Level Requirements

| ID | Requirement | Type | Priority |
|----|-------------|------|----------|
| HR-01 | System shall ingest telemetry from MQTT-enabled industrial devices | Functional | P0 |
| HR-02 | System shall display real-time dashboard with aggregate statistics | Functional | P0 |
| HR-03 | System shall generate alerts based on configurable thresholds | Functional | P0 |
| HR-04 | System shall provide per-device historical data visualization | Functional | P1 |
| HR-05 | System shall support CSV data export | Functional | P1 |
| HR-06 | System shall maintain WebSocket connections for live updates | Non-Functional | P0 |
| HR-07 | System shall auto-recover from WebSocket disconnections | Non-Functional | P1 |

## 7. Constraints

- Database: PostgreSQL only (self-hosted)
- MQTT Broker: HiveMQ
- No third-party cloud services
- Deployment via Docker Compose on-premise
- Single administrator account model

## 8. Assumptions

- Devices publish telemetry at regular intervals (approximately 5 seconds)
- Network connectivity between devices and MQTT broker is available
- Server has stable internet or internal network access

## 9. Success Criteria

- Dashboard loads within 3 seconds on initial visit
- WebSocket reconnection successful within 60 seconds after any disconnect
- Alert generation latency under 5 seconds from threshold breach
- Zero data loss for telemetry during normal operation
- Chart rendering completes within 2 seconds for any selected time range
