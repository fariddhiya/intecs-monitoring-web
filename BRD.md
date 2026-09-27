# Business Requirements Document (BRD)

## INTECS IoT Monitoring Web Application

| Field | Details |
|-------|---------|
| **Project Name** | INTECS IoT Monitoring Web |
| **Version** | 2.0 |
| **Status** | Approved |
| **Date** | 2026-09-27 |
| **Author** | Technical Team |
| **Classification** | Internal Use |

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Business Objectives](#2-business-objectives)
3. [Stakeholders](#3-stakeholders)
4. [Business Problem & Opportunity](#4-business-problem--opportunity)
5. [Business Use Cases](#5-business-use-cases)
6. [Scope](#6-scope)
7. [High-Level Requirements](#7-high-level-requirements)
8. [Success Criteria & KPIs](#8-success-criteria--kpis)
9. [Constraints](#9-constraints)
10. [Assumptions](#10-assumptions)
11. [Risks & Mitigations](#11-risks--mitigations)
12. [Cost-Benefit Analysis](#12-cost-benefit-analysis)
13. [Timeline & Milestones](#13-timeline--milestones)
14. [Appendices](#14-appendices)

---

## 1. Executive Summary

INTECS requires a real-time web-based monitoring platform for industrial equipment telemetry data collected via MQTT. The system ingests telemetry from simulated and physical industrial devices, processes alert conditions based on configurable thresholds, and presents operational dashboards to plant operators for continuous monitoring, historical analysis, and incident management.

This document outlines the business requirements for Phase 1 of the INTECS IoT Monitoring Web Application, which establishes foundational capabilities including real-time fleet monitoring, intelligent alerting, historical trend visualization, and alert lifecycle management through acknowledgment and resolution workflows.

---

## 2. Business Objectives

| # | Objective | Priority | Measurable Outcome |
|---|-----------|----------|-------------------|
| 1 | Provide real-time visibility into equipment health metrics (fuel, temperature, flow rate) across all monitored assets | P0 | Dashboard loads within 3 seconds; WebSocket updates delivered within 2 seconds |
| 2 | Reduce equipment downtime by enabling proactive alerting on threshold violations | P0 | Alert generation latency under 5 seconds from threshold breach |
| 3 | Enable operators to review historical telemetry trends for maintenance planning | P1 | Chart rendering completes within 2 seconds for any selected time range (up to 500 data points) |
| 4 | Support scalable device onboarding without backend code changes | P1 | New devices automatically registered on first telemetry message via MQTT wildcard subscription |
| 5 | Maintain operational continuity through automatic recovery from connection failures | P1 | WebSocket reconnection success rate > 95% within 60 seconds after disconnect |
| 6 | Standardize alert response process through structured acknowledge/solve workflow | P1 | 100% of critical alerts acknowledged within defined SLA |
| 7 | Facilitate compliance reporting through CSV data export | P2 | Export generation completes within 3 seconds for datasets up to 10,000 rows |

### Strategic Alignment

These objectives directly support:
- **Operational Excellence**: Reducing unplanned downtime through predictive insights
- **Asset Optimization**: Extending equipment lifespan via condition-based maintenance
- **Safety Compliance**: Ensuring thermal and fuel safety thresholds are continuously monitored
- **Data-Driven Decisions**: Providing actionable analytics rather than reactive firefighting

---

## 3. Stakeholders

| Role | Responsibility | System Access | Decision Authority |
|------|----------------|---------------|-------------------|
| **Plant Operators** | Monitor dashboards, acknowledge alerts, initiate response protocols, solve alerts with resolution notes | Full dashboard access, alert management | Acknowledge/solve decisions |
| **Maintenance Engineers** | Review historical data, plan preventive maintenance, analyze degradation patterns | Device detail pages, chart visualizations, CSV export | Maintenance scheduling |
| **System Administrators** | Manage infrastructure, configure system-wide thresholds, monitor system health, manage sites | Configuration endpoints, system status, database migrations | Infrastructure configuration |
| **IoT Engineers** | Design device firmware, manage MQTT broker, validate telemetry accuracy, troubleshoot ingestion pipelines | MQTT broker admin, API testing tools, log access | Technical architecture decisions |
| **Management/Leadership** | Review aggregate KPIs, allocate resources, approve budget for scaling | Executive summary reports (future phase) | Budget and priority approvals |

---

## 4. Business Problem & Opportunity

### Current State

Industrial operations currently rely on manual or semi-automated approaches to monitor equipment conditions:
- **Siloed Data**: Telemetry information exists in isolated device controllers with no centralized view
- **Reactive Response**: Equipment failures discovered only after catastrophic malfunction, causing extended downtime
- **Limited Historical Context**: No systematic collection of historical sensor data for trend analysis
- **Alert Fatigue**: Threshold alarms exist but lack prioritization, acknowledgment tracking, or resolution workflows
- **Reporting Burden**: Manual compilation of equipment status reports consumes significant operator time

### Pain Points Quantified

| Pain Point | Impact | Estimated Cost |
|------------|--------|---------------|
| Unplanned equipment downtime | Production loss, emergency repair costs | $5,000–$50,000 per incident |
| Delayed fault detection | Secondary damage, safety risks | $2,000–$10,000 per delayed response hour |
| Manual reporting overhead | Operator time diverted from critical tasks | 4–8 hours per operator per week |
| Lack of maintenance planning | Over-maintenance or under-maintenance of assets | 15–30% excess maintenance spending |

### Opportunity

Implementing a centralized, real-time monitoring platform enables:
- **Proactive Maintenance**: Identify degradation patterns before failures occur
- **Reduced MTTR** (Mean Time To Resolution): Structured alert acknowledgment and solve workflows
- **Improved MTTF** (Mean Time To Failure): Early warning on critical threshold breaches
- **Scalable Operations**: Support additional sites and devices without proportional staffing increases
- **Regulatory Compliance**: Audit trail of alert events, acknowledgments, and resolutions

---

## 5. Business Use Cases

### UC-01: Real-Time Fleet Monitoring

| Attribute | Description |
|-----------|-------------|
| **Use Case ID** | UC-01 |
| **Actor** | Plant Operator |
| **Priority** | P0 (Critical) |
| **Trigger** | Operator logs into the dashboard |

**Main Flow:**
1. Operator authenticates with credentials
2. Dashboard displays live status of all connected devices
3. Aggregate statistics show: Total Devices, Online Count, Offline Count, Active Alerts
4. Device table renders with current telemetry: fuel %, temperature, flow rate, equipment status
5. Connection status indicators auto-update every 5 seconds (polling) + instant updates via WebSocket
6. Device detail page accessible by clicking any row, showing interactive charts and historical trends

**Business Value:** Operators maintain situational awareness of entire equipment fleet without manual checks or switching between disparate systems. Enables immediate identification of offline or distressed assets.

**Success Metric:** Dashboard fully operational within 3 seconds of login; device status reflects MQTT state within 2 seconds.

---

### UC-02: Alert Detection and Response

| Attribute | Description |
|-----------|-------------|
| **Use Case ID** | UC-02 |
| **Actor** | Plant Operator |
| **Priority** | P0 (Critical) |
| **Trigger** | Telemetry crosses defined thresholds or device goes offline |

**Main Flow:**
1. System evaluates incoming telemetry against configured thresholds (fuel, temperature)
2. If condition met, alert generated with severity classification (Warning/Critical/Medium)
3. Alert pushed to all connected browsers via WebSocket in real-time
4. Browser notification displayed (if permission granted)
5. Operator views alert details from Active tab in Alert Management
6. Operator clicks "Acknowledge" button to claim responsibility
7. Alert moves to Acknowledged tab, awaiting resolution
8. Operator investigates root cause, takes corrective action
9. Operator clicks "Solve", provides resolution notes
10. Alert moves to Solved tab with timestamp and solver attribution

**Postconditions:** Complete audit trail created: who acknowledged, who solved, when, and why.

**Business Value:** Reduces Mean Time To Detection (MTTD) and Mean Time To Resolution (MTTR). Structured workflow prevents alerts from being overlooked or forgotten. Accountability tracked through user attribution.

**Success Metric:** Alert generated within 5 seconds of threshold breach; 100% of alerts acknowledged within defined SLA (to be established post-deployment).

---

### UC-03: Historical Trend Analysis

| Attribute | Attribute | Description |
|-----------|-----------|-------------|
| **Use Case ID** | UC-03 |
| **Actor** | Maintenance Engineer |
| **Priority** | P1 (High) |
| **Trigger** | Engineer selects specific device and time range |

**Main Flow:**
1. Engineer navigates to Device Detail page for target asset
2. System retrieves latest telemetry via WebSocket stream + polling fallback
3. Engineer selects time range: 1h, 6h, 24h, or 7d
4. System fetches historical telemetry (up to 500 data points) via REST API
5. Interactive charts render: Fuel Percentage History (area chart), Temperature History (line chart)
6. Engineer analyzes trends: gradual fuel depletion, temperature spikes, correlations between metrics
7. Engineer exports data as CSV for offline analysis or reporting

**Business Value:** Enables predictive maintenance by identifying degradation patterns before failures occur. Fuel consumption trending reveals inefficiencies. Temperature trending detects bearing wear or cooling system deterioration. Supports data-driven maintenance scheduling rather than fixed-interval schedules.

**Success Metric:** Chart rendering completes within 2 seconds; CSV export generates correctly formatted file within 3 seconds.

---

### UC-04: Device Health Reporting and Inventory Management

| Attribute | Attribute | Description |
|-----------|-----------|-------------|
| **Use Case ID** | UC-04 |
| **Actor** | System Administrator |
| **Priority** | P1 (High) |
| **Trigger** | Periodic review or event-driven investigation |

**Main Flow:**
1. Administrator accesses Dashboard device inventory view
2. Applies filters: connection status (ONLINE/STALE/OFFLINE), equipment status (Running/Idle/Maintenance)
3. Searches by device ID or name
4. Sorts columns ascending/descending
5. Reviews paginated results (configurable page size: 5/10/15)
6. Exports current filtered view as CSV for compliance reporting or asset inventory
7. Uses exported data for audit trails, insurance documentation, regulatory compliance

**Business Value:** Supports compliance reporting and asset inventory management. Eliminates manual spreadsheet compilation. Filter and sort capabilities enable targeted investigations (e.g., "Show me all offline devices that were running yesterday").

**Success Metric:** Export contains accurate, complete data matching filtered view. Paginated navigation handles datasets of 100+ devices efficiently.

---

### UC-05: Customizable Alert Thresholds

| Attribute | Attribute | Description |
|-----------|-----------|-------------|
| **Use Case ID** | UC-05 |
| **Actor** | IoT Engineer / System Administrator |
| **Priority** | P1 (High) |
| **Trigger** | Different equipment types require different alert criteria |

**Main Flow:**
1. Engineer identifies new device type with unique threshold requirements
2. Configures per-device thresholds via REST API: warn_temp, crit_temp, warn_fuel, crit_fuel, cooldown_seconds
3. System persists configuration in `device_thresholds` table
4. Alert engine uses per-device thresholds instead of defaults for this device
5. Engineer verifies alert behavior via test telemetry or simulation

**Business Value:** Eliminates false positives from one-size-fits-all alert rules. Different equipment operating parameters (e.g., older generators vs. newer pumps) can have distinct threshold profiles. Cooldown tuning prevents alert storms during transient conditions.

**Success Metric:** Per-device thresholds applied correctly within next evaluation cycle (< 5 seconds after API call). Default values used as fallback when no per-device config exists.

---

### UC-06: Multi-Site Operations Support

| Attribute | Attribute | Description |
|-----------|-----------|-------------|
| **Use Case ID** | UC-06 |
| **Actor** | Regional Manager |
| **Priority** | P2 (Medium) |
| **Trigger** | Organization operates across multiple geographic locations |

**Main Flow:**
1. System abstracts site hierarchy via `sites` table
2. Each site has unique `site_id` referenced in MQTT topic structure: `intecs/site/{site_id}/device/...`
3. Dashboard displays aggregated statistics across all sites
4. Future enhancement: Site-specific filtering and drill-down views

**Business Value:** Foundation for distributed operations. Supports organizational structure where multiple sites report to central management. Each site operates independently yet contributes to global visibility.

**Success Metric:** New sites auto-created on first telemetry message. No manual site provisioning required.

---

## 6. Scope

### In Scope (Phase 1)

| Capability | Description |
|------------|-------------|
| **Telemetry Ingestion** | Real-time MQTT subscription, parsing, validation, and persistence |
| **Device Management** | Automatic upsert, status computation (ONLINE/STALE/OFFLINE), last_seen tracking |
| **Site Abstraction** | Hierarchical site grouping for multi-location operations |
| **Real-Time Dashboard** | Aggregate stats, paginated device table, color-coded indicators |
| **Alert Engine** | Threshold evaluation, severity classification, cooldown deduplication |
| **Alert Lifecycle** | Three-state workflow: Active → Acknowledged → Solved with user attribution |
| **WebSocket Notifications** | Push-based updates for telemetry, alerts, and broker status |
| **Historical Visualization** | Chart.js interactive charts with time-range selectors |
| **CSV Export** | Export functionality for devices, telemetry, and alerts |
| **Dark/Light Theme** | User preference persisted in localStorage |
| **Authentication** | Single admin account with token-based session management |
| **Containerized Deployment** | Docker Compose orchestration for PostgreSQL, HiveMQ, Backend, Simulator, Frontend |

### Out of Scope (Phase 1) — Deferred to Future Phases

| Feature | Rationale | Target Phase |
|---------|-----------|--------------|
| Multi-user role management | Complex authentication/RBAC exceeds Phase 1 scope | Phase 2 |
| Email/Push notifications beyond browser | Requires third-party services (SMTP, Firebase) | Phase 3 |
| Automated device provisioning workflow | Manual first-contact registration acceptable initially | Phase 3 |
| Integration with CMMS/EAM systems | External API integration requires vendor selection | Phase 3 |
| Mobile application | Desktop-first approach sufficient for control room operations | Phase 3 |
| Machine learning anomaly detection | Requires historical dataset and ML engineering effort | Phase 4 |
| Multi-tenant architecture | Single organization deployment is Phase 1 requirement | Phase 4 |
| Database query performance optimization | Current PostgreSQL setup adequate for 10–100 devices | Phase 3 (if scale demands) |

---

## 7. High-Level Requirements

### Functional Requirements

| ID | Requirement | Source Use Case | Priority |
|----|-------------|-----------------|----------|
| HR-01 | System shall ingest telemetry from MQTT-enabled industrial devices at configurable intervals | UC-01, UC-02 | P0 |
| HR-02 | System shall display real-time dashboard with aggregate statistics and device list | UC-01 | P0 |
| HR-03 | System shall generate alerts based on configurable thresholds with severity classification | UC-02, UC-05 | P0 |
| HR-04 | System shall provide per-device historical data visualization with interactive charts | UC-03 | P1 |
| HR-05 | System shall support CSV data export for devices, telemetry, and alerts | UC-04 | P1 |
| HR-06 | System shall maintain WebSocket connections for live updates | UC-01, UC-02 | P0 |
| HR-07 | System shall auto-recover from WebSocket disconnections with exponential backoff | UC-01 | P1 |
| HR-08 | System shall support alert acknowledgment and resolution workflow with user attribution | UC-02 | P1 |
| HR-09 | System shall persist all telemetry, devices, alerts, and thresholds to PostgreSQL | All | P0 |
| HR-10 | System shall apply per-device or system-default threshold configurations | UC-05 | P1 |
| HR-11 | System shall prevent duplicate alerts within configurable cooldown periods | UC-02 | P1 |
| HR-12 | System shall support dark/light theme toggle with persisted preference | UC-01 | P2 |

### Non-Functional Requirements

| ID | Requirement | Type | Priority |
|----|-------------|------|----------|
| H-NFR-01 | Dashboard initial load completes within 3 seconds | Performance | P0 |
| H-NFR-02 | WebSocket message delivery latency under 2 seconds | Performance | P0 |
| H-NFR-03 | System supports minimum 50 concurrent WebSocket clients | Scalability | P1 |
| H-NFR-04 | System handles 100 devices × 12 messages/min = 1200 msg/min sustained throughput | Scalability | P1 |
| H-NFR-05 | Zero data loss for telemetry during normal operation | Reliability | P0 |
| H-NFR-06 | WebSocket reconnection success rate > 95% within 60 seconds | Reliability | P1 |
| H-NFR-07 | All queries use parameterized statements (SQL injection prevention) | Security | P0 |
| H-NFR-08 | Secrets loaded from environment variables, never committed or logged | Security | P0 |
| H-NFR-09 | Responsive layout adapts at breakpoints: 480px, 768px, 1024px | Usability | P1 |
| H-NFR-10 | Containerized deployment via Docker Compose on-premise only | Portability | P0 |

---

## 8. Success Criteria & KPIs

### Key Performance Indicators

| KPI | Target | Measurement Method |
|-----|--------|-------------------|
| **Dashboard Load Time** | < 3 seconds from route navigation | Browser DevTools Network tab |
| **Alert Generation Latency** | < 5 seconds from threshold breach | Compare MQTT publish timestamp vs. alert.created_at in DB |
| **WebSocket Delivery Latency** | < 2 seconds end-to-end | Compare server broadcast timestamp vs. client receipt timestamp |
| **Chart Rendering Time** | < 2 seconds for full dataset (500 points) | Browser DevTools Performance tab |
| **WebSocket Reconnection Rate** | > 95% success within 60 seconds | Client-side reconnect attempt logging |
| **Telemetry Ingestion Throughput** | ≥ 1200 messages/minute sustained | MQTT broker publish counter |
| **Zero Data Loss** | 100% of valid telemetry persisted | Compare MQTT published count vs. DB row count |
| **API Response Time (p95)** | < 200 ms for standard queries | APM tool or Nginx access logs |

### Business Outcome Metrics

| Metric | Baseline (Target) | Phase 1 Goal |
|--------|------------------|--------------|
| Unplanned downtime per month | Industry average: 8–12 hours | Reduce by 30% within 6 months |
| Mean Time To Detection (MTTD) | Reactive (hours) | Proactive (< 5 minutes) |
| Mean Time To Resolution (MTTR) | Untracked | Track via alert resolved_at - created_at delta |
| Operator reporting time | 4–8 hours/week | Eliminate manual reporting via automated export |
| False alarm rate | Unknown | Reduce via configurable cooldown and per-device thresholds |

---

## 9. Constraints

| Constraint | Description | Impact |
|------------|-------------|--------|
| **Database Platform** | PostgreSQL only (self-hosted via Docker) | Limits time-series optimizations; future migration to TimescaleDB possible |
| **MQTT Broker** | HiveMQ 4.x | Vendor-locked to HiveMQ ecosystem; Enterprise features not available |
| **No Third-Party Cloud Services** | Fully on-premise deployment | No managed authentication, SMS, email, or CDN services |
| **Deployment Model** | Docker Compose on-premise | Limited to single-node or simple cluster; no Kubernetes orchestration |
| **Authentication Model** | Single administrator account (Phase 1) | No role-based access control; all users share same credentials |
| **Language/Framework** | Go 1.23+ backend, SvelteKit frontend | Requires specific skill set; limits talent pool compared to Java/.NET ecosystems |
| **Network Topology** | Devices communicate via MQTT over TCP | Requires network connectivity < 50ms RTT for reliable operation |
| **Token Storage** | Client-side localStorage with base64 encoding | No server-side session management; potential security vulnerability if tokens stolen |

---

## 10. Assumptions

| # | Assumption | Validation Method |
|---|-----------|------------------|
| 1 | Devices publish telemetry at regular intervals (approximately 5 seconds) | Simulator publishes at configurable interval; production devices must comply |
| 2 | Network connectivity between devices and MQTT broker is available (< 50ms RTT) | Network infrastructure assessment prior to deployment |
| 3 | Server retains at least 4 GB RAM and 2 CPU cores for production use | Hardware specification confirmed with IT operations |
| 4 | Initial deployment monitors 10–50 devices across 1–3 sites | Capacity planning assumes linear scaling; revisit if growth exceeds projection |
| 5 | Alert thresholds follow industry standards (fuel < 10% critical, temp > 90°C critical) | Validate with engineering team; adjust per device type |
| 6 | Operators will acknowledge alerts within reasonable SLA (to be defined post-deployment) | Track via resolved_at - created_at metric; refine process as needed |
| 7 | No network partitions exceeding 30 seconds between devices and broker | Network reliability assessment; plan for retry logic if partitions occur |

---

## 11. Risks & Mitigations

| Risk | Probability | Impact | Mitigation Strategy |
|------|-----------|--------|-------------------|
| MQTT broker failure causes telemetry ingestion halt | Medium | High | Implement broker clustering; add local buffer for temporary outages |
| PostgreSQL data loss from disk failure | Low | Critical | Daily pg_dump backups; point-in-time recovery (WAL archiving) |
| Single admin credential shared among operators reduces accountability | High | Medium | Phase 1 workaround: X-User-Email header on alert actions; Phase 2: full auth backend |
| WebSocket disconnections cause missed real-time updates | Medium | Medium | Polling fallback (5-second interval); exponential backoff reconnection |
| Alert fatigue from excessive notifications | Medium | High | Configurable cooldown periods (default 30 seconds); severity classification reduces noise |
| Memory leak in WebSocket hub over extended uptime | Low | Medium | Structured cleanup on disconnect (readPump/writePump goroutine termination, channel close) |
| Device count growth exceeds database query performance | Low | Medium | Existing indexes optimize common queries; partitioning available if needed |
| HiveMQ license cost escalates with device growth | Medium | Medium | Evaluate open-source alternatives (EMQX, Mosquitto) if HiveMQ becomes cost-prohibitive |

---

## 12. Cost-Benefit Analysis

### Implementation Costs (Phase 1)

| Cost Category | Estimate | Notes |
|---------------|----------|-------|
| Development (Engineering effort) | Already completed | Codebase exists; minor enhancements ongoing |
| Infrastructure (On-premise hardware) | $2,000–$5,000 | Dedicated server or VM allocation |
| Licensing (HiveMQ Community Edition) | $0 | Open-source edition sufficient for Phase 1 |
| PostgreSQL (Self-hosted) | $0 | Open-source |
| Training (Operator onboarding) | $500–$1,000 | 2–4 hours per operator |
| Change Management | $1,000–$2,000 | Process documentation, SOP updates |
| **Total Phase 1** | **$3,500–$8,000** | One-time implementation cost |

### Expected Benefits

| Benefit | Annual Value | Calculation Basis |
|---------|-------------|-------------------|
| Reduced unplanned downtime | $30,000–$150,000 | 30% reduction × average incident cost ($5k–$50k) × incidents/year |
| Labor savings (manual reporting eliminated) | $8,000–$16,000 | 4–8 hours/week × $20–$40/hour × 52 weeks |
| Optimized maintenance scheduling | $10,000–$30,000 | 15–30% reduction in excess maintenance spending |
| Extended equipment lifespan | $20,000–$50,000 | Proactive intervention prevents secondary damage |
| **Total Annual Benefits** | **$68,000–$246,000** | Conservative to optimistic estimates |

### ROI Projection

| Metric | Value |
|--------|-------|
| **First Year Net Benefit** | $60,000–$238,000 (Benefits minus implementation) |
| **Payback Period** | 1–2 months |
| **3-Year ROI** | 800%–2,000% |

---

## 13. Timeline & Milestones

| Milestone | Target Date | Deliverables | Dependencies |
|-----------|-------------|--------------|--------------|
| **M1: Codebase Completion** | Q3 2026 | Backend API, Frontend Dashboard, Simulator — all functional | None (already complete) |
| **M2: Pilot Deployment** | Q4 2026 | Deploy to 1–3 devices at Sangatta Site; Operator training | Infrastructure ready, hardware provisioned |
| **M3: Scale to 10 Devices** | Q1 2027 | Expand to full simulator capacity; Validate 1200 msg/min throughput | Successful pilot, feedback incorporated |
| **M4: Production Hardening** | Q2 2027 | Authentication overhaul, backup strategy, monitoring | Phase 1 stable operation, budget approved |
| **M5: Advanced Features** | Q3 2027+ | Mobile app, ML predictions, CMMS integration | Phase 4 deferred projects |

---

## 14. Appendices

### Appendix A: Glossary

| Term | Definition |
|------|-----------|
| **MQTT** | Message Queuing Telemetry Transport — lightweight publish/subscribe messaging protocol for IoT |
| **Telemetry** | Time-series sensor data: fuel percentage, fuel level, temperature, flow rate, equipment status |
| **Threshold** | Configurable boundary value that triggers an alert when breached |
| **Cooldown** | Minimum time interval between repeated alerts for the same device and alert type |
| **Acknowledged** | Alert state indicating operator has reviewed and claimed responsibility |
| **Solved** | Alert state indicating issue resolved and documented with resolution notes |
| **ONLINE** | Device connectivity status: telemetry received within last 1 minute |
| **STALE** | Device connectivity status: telemetry delayed 1–5 minutes |
| **OFFLINE** | Device connectivity status: no telemetry reported for > 5 minutes |
| **Hub** | WebSocket connection multiplexer managing client connections and message routing |
| **ReadPump / WritePump** | Goroutine pairs handling inbound/outbound WebSocket traffic per client |

### Appendix B: Referenced Documents

- `SRS.md` — Software Requirements Specification (detailed technical specifications)
- `rules.md` — Project development rules and conventions
- `.env.example` — Environment variable definitions and defaults
- `docker-compose.yml` — Deployment orchestration configuration
- `backend/internal/database/database.go` — Database schema definitions and migration logic

### Appendix C: Default Threshold Values

| Parameter | Value | Unit | Trigger Condition | Severity |
|-----------|-------|------|-------------------|----------|
| `warn_fuel_threshold` | 20 | % | Fuel drops below | Warning |
| `crit_fuel_threshold` | 10 | % | Fuel drops below | Critical |
| `warn_temp_threshold` | 85 | °C | Temperature exceeds | Warning |
| `crit_temp_threshold` | 90 | °C | Temperature exceeds | Critical |
| `cooldown_seconds` | 30 | seconds | Min interval between repeated alerts | N/A |
| `offline_threshold_minutes` | 5 | minutes | No telemetry received for duration | Medium |

### Appendix D: Device Types and Classifications

| Device Type | Classification | Typical Parameters |
|-------------|---------------|-------------------|
| Industrial Generator | Heavy Equipment | Fuel-heavy operation, high thermal output |
| Mining Vehicle | Mobile Asset | Variable duty cycles, environmental exposure |
| Processing Pump | Stationary Equipment | Steady-state operation, pressure-sensitive |
| Conveyor System | Material Handling | Continuous operation, motor temperature monitoring |

*Note: Device type is currently stored as `"Industrial Equipment"` default. Future enhancement to classify by type for threshold customization.*
