Saya ingin membuat prototype website IoT Fuel & Equipment Monitoring untuk technical assignment/interview.

IMPORTANT:
Project ini hanya memiliki waktu pengerjaan sekitar 2 hari. Jangan over-engineer dan jangan membuat microservices atau infrastructure production yang kompleks.

TUJUAN:
Membuat prototype end-to-end yang menunjukkan alur:

IoT Simulator
↓ MQTT
HiveMQ
↓ MQTT Subscribe
Go Backend
↓
PostgreSQL
↓ REST API
Svelte Dashboard

Gunakan arsitektur sederhana dengan SATU Go backend yang menangani:

1. MQTT consumer
2. REST API
3. Basic alert logic
4. PostgreSQL access

JANGAN membuat Go ingestion sebagai service terpisah untuk prototype ini.

TECH STACK:

- Backend: Go
- MQTT Broker: HiveMQ
- MQTT Client: library MQTT yang mature untuk Go
- Database: PostgreSQL
- Frontend: Svelte
- API: REST
- Communication: MQTT
- Development environment: Docker Compose
- Authentication: sederhana jika waktu memungkinkan
- Frontend styling: gunakan library/component system yang sederhana dan clean

JANGAN gunakan:

- Kubernetes
- Kafka
- Redis
- TimescaleDB
- Microservices
- MQTT cluster
- WebSocket pada MVP
- complex event-driven architecture
- unnecessary abstractions

Semua teknologi tersebut hanya boleh disebut sebagai future scalability consideration, bukan diimplementasikan.

==================================================

1. # BUSINESS USE CASE

Buat sistem monitoring fuel dan equipment untuk lingkungan industrial/mining.

Sistem harus dapat memonitor:

- Fuel percentage
- Fuel level
- Temperature
- Flow rate
- Equipment status
- Device connectivity
- Basic alerts

Contoh equipment:

DT-001
DT-002
DT-003
dst.

Prototype cukup mensimulasikan 10 device.

# ================================================== 2. MQTT TOPIC

Gunakan topic:

intecs/site/{site_id}/device/{device_id}/telemetry

Contoh:

intecs/site/sangatta/device/DT-001/telemetry

Payload:

{
"device_id": "DT-001",
"timestamp": "2026-09-25T15:30:00Z",
"fuel_percentage": 72.5,
"fuel_level": 7250,
"temperature": 81.2,
"flow_rate": 35.2,
"equipment_status": "running"
}

# ================================================== 3. IoT SIMULATOR

Buat Go IoT simulator.

Simulator harus dapat menjalankan 10 virtual devices.

Setiap device publish telemetry setiap 5 detik.

Contoh:

DT-001
DT-002
...
DT-010

Nilai telemetry harus berubah secara realistis, jangan random ekstrem.

Contoh:
fuel_percentage perlahan turun,
temperature berubah dalam range tertentu,
flow_rate berubah,
equipment_status dapat berubah.

Simulator harus dapat dikonfigurasi:

DEVICE_COUNT=10
PUBLISH_INTERVAL=5s

Jika memungkinkan tambahkan:
SIMULATION_SPEED
SITE_ID

# ================================================== 4. GO BACKEND

Go backend harus memiliki:

A. MQTT Consumer

Subscribe:

intecs/site/+/device/+/telemetry

Flow:

MQTT message
→ decode JSON
→ validate
→ save PostgreSQL
→ update current device state
→ evaluate basic alert

B. REST API

Implementasikan minimal:

GET /api/dashboard
GET /api/devices
GET /api/devices/:id
GET /api/devices/:id/telemetry
GET /api/alerts

Jika memungkinkan:

POST /api/alerts/:id/acknowledge

# ================================================== 5. DATABASE

Gunakan PostgreSQL.

Buat schema sederhana.

Table:

sites

- id
- name
- created_at

devices

- id
- device_id
- site_id
- name
- type
- status
- last_seen
- created_at

telemetry

- id
- device_id
- timestamp
- fuel_percentage
- fuel_level
- temperature
- flow_rate
- equipment_status

alerts

- id
- device_id
- type
- severity
- message
- status
- created_at
- resolved_at

Gunakan UUID atau struktur ID yang konsisten.

Tambahkan index yang relevan, terutama:

telemetry(device_id, timestamp)

# ================================================== 6. DEVICE STATUS

Device status dihitung berdasarkan last_seen.

Contoh:

ONLINE:
last_seen < 1 minute

STALE:
1–5 minutes

OFFLINE:

> 5 minutes

Threshold harus mudah diubah.

# ================================================== 7. ALERT

Implementasikan basic alert:

LOW_FUEL:
fuel_percentage < 20

CRITICAL_FUEL:
fuel_percentage < 10

HIGH_TEMPERATURE:
temperature > configurable threshold

DEVICE_OFFLINE:
device tidak mengirim telemetry melewati threshold

Jangan membuat alert engine terpisah.

Basic logic di Go backend sudah cukup.

Hindari duplicate alert.

Contoh:
Jika fuel tetap 15% selama 20 detik, jangan membuat alert baru setiap telemetry.

# ================================================== 8. SVELTE DASHBOARD

Buat dashboard yang clean dan professional.

Halaman utama:

Dashboard

Tampilkan:

Total Devices
Online Devices
Offline Devices
Active Alerts

Kemudian tabel:

Device
Site
Fuel
Temperature
Flow Rate
Equipment Status
Connection Status

Contoh:

DT-001 | Sangatta | 72% | 81°C | 35 L/min | Running | Online

Gunakan visual indicator untuk:
Online
Offline
Low Fuel
Critical Fuel

# ================================================== 9. DEVICE DETAIL

Ketika user klik device:

/devices/:id

Tampilkan:

Device information
Current fuel
Current temperature
Current flow rate
Equipment status
Connection status
Last seen

Tambahkan historical chart sederhana:

Fuel percentage over time
Temperature over time

Gunakan REST API.

Tidak perlu WebSocket untuk MVP.

Dashboard boleh melakukan polling setiap 5 detik untuk mendapatkan data terbaru.

# ================================================== 10. AUTHENTICATION

Karena hanya memiliki waktu 2 hari, authentication sederhana saja.

Jika memungkinkan buat:

POST /api/auth/login

User:

admin@example.com

Password:
gunakan environment variable atau seed database.

Jika authentication mengganggu penyelesaian core feature, prioritaskan MQTT → backend → database → dashboard terlebih dahulu.

Jangan membuat authentication system yang kompleks.

# ================================================== 11. DOCKER

Buat docker-compose.yml untuk:

- HiveMQ
- PostgreSQL
- Go backend
- Svelte frontend jika memungkinkan

Namun development harus tetap mudah dijalankan secara lokal.

Sediakan:

.env.example

Contoh:

DATABASE_URL=
MQTT_BROKER_URL=
MQTT_USERNAME=
MQTT_PASSWORD=
DEVICE_COUNT=
PUBLISH_INTERVAL=

Jangan memasukkan secret asli ke Git.

# ================================================== 12. PROJECT STRUCTURE

Gunakan struktur yang sederhana dan mudah dipahami.

Contoh:

project/
├── backend/
│ ├── cmd/
│ ├── internal/
│ │ ├── mqtt/
│ │ ├── api/
│ │ ├── database/
│ │ ├── device/
│ │ └── alert/
│ ├── migrations/
│ └── Dockerfile
│
├── simulator/
│ ├── main.go
│ └── Dockerfile
│
├── frontend/
│ ├── src/
│ └── Dockerfile
│
├── docker-compose.yml
├── .env.example
└── README.md

Jangan membuat abstraction layer yang tidak diperlukan.

# ================================================== 13. ERROR HANDLING

Backend harus menangani:

- MQTT connection failure
- PostgreSQL connection failure
- invalid MQTT payload
- invalid device_id
- malformed JSON
- duplicate alert
- device reconnect

MQTT consumer harus melakukan reconnect jika broker disconnect.

# ================================================== 14. SECURITY

Untuk prototype implementasikan basic security:

- Environment variables untuk credentials
- MQTT authentication
- MQTT ACL jika mudah dikonfigurasi
- Input validation
- Password tidak boleh hardcoded
- Database tidak perlu expose ke public internet pada production architecture

Tidak perlu implementasi enterprise security.

Di README jelaskan bahwa production version dapat menambahkan:

- MQTT TLS
- mTLS
- device certificates
- RBAC
- secret manager
- network segmentation

# ================================================== 15. PRODUCTION SCALABILITY

Jangan implementasikan scalability infrastructure.

Tetapi dokumentasikan di README:

Prototype:

IoT Simulator
→ HiveMQ
→ Go Backend
→ PostgreSQL
→ Svelte

Production dapat dikembangkan menjadi:

IoT Devices
→ MQTT Cluster
→ Ingestion Service
→ Message Queue
→ Time-series Database
→ API Cluster
→ Dashboard

Jelaskan bahwa pemisahan ingestion dan API dilakukan jika workload meningkat.

Jelaskan bahwa PostgreSQL dapat diganti/diperluas dengan TimescaleDB untuk telemetry dalam volume besar.

# ================================================== 16. README

README harus menjelaskan:

1. Project overview
2. Architecture
3. Business use case
4. Tech stack
5. MQTT topic
6. Payload format
7. Database schema
8. How to run
9. How to start simulator
10. API endpoints
11. Dashboard features
12. Alert rules
13. Security consideration
14. Production scalability consideration
15. Known limitations

Tambahkan architecture diagram dalam Mermaid.

# ================================================== 17. DEVELOPMENT PRIORITY

Prioritaskan pengerjaan berdasarkan urutan:

P0:

1. HiveMQ
2. PostgreSQL
3. Go backend
4. MQTT consumer
5. IoT simulator
6. REST API
7. Svelte dashboard
8. Device monitoring
9. Fuel monitoring

P1: 10. Alerts 11. Historical chart 12. Device status

P2: 13. Authentication 14. Docker improvement 15. UI polish

Jangan mengerjakan P2 sebelum P0 selesai.

# ================================================== 18. IMPORTANT ENGINEERING PRINCIPLE

Prototype harus:

- simple
- runnable
- understandable
- easy to demonstrate
- easy to explain during interview

Jangan over-engineer.

Jika ada beberapa pilihan architecture, pilih solusi yang paling sederhana yang tetap menunjukkan konsep IoT dengan benar.

Jangan menambahkan teknologi hanya untuk terlihat sophisticated.

Sebelum menulis kode, jelaskan secara singkat:

1. Architecture
2. Database schema
3. MQTT flow
4. API design
5. Project structure
6. Implementation steps

Setelah itu implementasikan secara bertahap dan pastikan setiap tahap dapat dijalankan sebelum lanjut ke tahap berikutnya.

# ================================================== 19. GIT WORKFLOW — WAJIB

Gunakan Git secara disiplin.

PRINSIP UTAMA:

Selesaikan SATU feature terlebih dahulu, kemudian:

1. Implementasikan feature
2. Jalankan application/build/test yang relevan
3. Pastikan feature berhasil
4. Review perubahan
5. git status
6. git diff
7. git add
8. git commit
9. git push
10. Baru lanjut ke feature berikutnya

JANGAN mengerjakan banyak feature sekaligus sebelum commit.

Setiap commit harus merepresentasikan SATU logical feature atau perubahan yang jelas.

Gunakan Conventional Commits.

Format:

<type>(<scope>): <description>

Contoh:

feat(mqtt): add telemetry consumer
feat(simulator): add device telemetry simulator
feat(database): add telemetry schema
feat(api): add device endpoints
feat(alert): add low fuel alert
feat(frontend): add device dashboard
feat(frontend): add device telemetry chart
fix(mqtt): handle broker reconnect
fix(alert): prevent duplicate alerts
docs(readme): add architecture documentation

==================================================
GIT BRANCH
==================================================

Gunakan branch:

feature/iot-monitoring

Jika branch belum ada:

git checkout -b feature/iot-monitoring

Jangan langsung bekerja di main/master kecuali repository memang mengharuskan demikian.

==================================================
FEATURE-BY-FEATURE WORKFLOW
==================================================

Kerjakan dalam urutan berikut.

FEATURE 01
Repository & project structure

Implement:

- folder structure
- .gitignore
- .env.example
- basic README
- initial project setup

Test:

- project dapat di-build

Commit:

chore(init): initialize project structure

Push:

git push -u origin feature/iot-monitoring

FEATURE 02
PostgreSQL setup

Implement:

- PostgreSQL Docker container
- database connection
- migration
- sites table
- devices table
- telemetry table
- alerts table

Test:

- database dapat dijalankan
- migration berhasil
- backend dapat connect ke PostgreSQL

Commit:

feat(database): add PostgreSQL schema

FEATURE 03
HiveMQ setup

Implement:

- HiveMQ Docker/container setup
- MQTT connection configuration
- MQTT credentials jika digunakan
- basic broker configuration

Test:

- HiveMQ dapat berjalan
- MQTT client dapat connect

Commit:

feat(mqtt): setup HiveMQ broker

FEATURE 04
IoT Simulator

Implement:

- Go simulator
- 10 virtual devices
- telemetry generation
- configurable publish interval
- MQTT publishing

Topic:

intecs/site/{site_id}/device/{device_id}/telemetry

Test:

- simulator berhasil connect ke HiveMQ
- telemetry berhasil dipublish
- payload valid

Commit:

feat(simulator): add IoT telemetry simulator

FEATURE 05
MQTT Consumer

Implement:

- MQTT subscription
- subscribe:

intecs/site/+/device/+/telemetry

- JSON decoding
- payload validation
- PostgreSQL insertion

Test:
Simulator
→ HiveMQ
→ Go Backend
→ PostgreSQL

harus berhasil.

Commit:

feat(mqtt): add telemetry consumer

FEATURE 06
Device Current State

Implement:

- device registration/update
- last_seen
- current status
- ONLINE / STALE / OFFLINE calculation

Test:

- device muncul di database
- last_seen berubah ketika telemetry masuk
- status berubah sesuai threshold

Commit:

feat(device): add device status tracking

FEATURE 07
REST API

Implement:

GET /api/dashboard
GET /api/devices
GET /api/devices/:id
GET /api/devices/:id/telemetry
GET /api/alerts

Test:

- endpoint dapat diakses
- response JSON valid
- error handling basic tersedia

Commit:

feat(api): add monitoring endpoints

FEATURE 08
Low Fuel Alert

Implement:

LOW_FUEL:
fuel_percentage < 20

CRITICAL_FUEL:
fuel_percentage < 10

HIGH_TEMPERATURE:
temperature > configurable threshold

DEVICE_OFFLINE:
device melewati offline threshold

Hindari duplicate alert.

Test:

- simulator menghasilkan kondisi low fuel
- alert masuk database
- API mengembalikan alert
- alert tidak dibuat berulang pada setiap telemetry

Commit:

feat(alert): add equipment monitoring alerts

FEATURE 09
Svelte Dashboard

Implement:

- dashboard layout
- summary cards
- device table
- fuel percentage
- temperature
- flow rate
- equipment status
- connection status
- active alerts

Test:

- frontend dapat mengambil data dari API
- device list tampil
- dashboard summary tampil

Commit:

feat(frontend): add monitoring dashboard

FEATURE 10
Device Detail

Implement:

/devices/:id

Tampilkan:

- device information
- current fuel
- temperature
- flow rate
- equipment status
- connection status
- last seen

Commit:

feat(frontend): add device detail page

FEATURE 11
Historical Telemetry

Implement:

- fuel history
- temperature history
- simple chart
- API query berdasarkan time range

Test:

- telemetry history dapat ditampilkan
- chart tidak error jika data kosong

Commit:

feat(frontend): add telemetry history chart

FEATURE 12
Dashboard Polling

Implement:

- frontend polling setiap 5 detik
- refresh device state
- refresh alerts

Jangan implement WebSocket pada tahap ini.

Commit:

feat(frontend): add dashboard polling

FEATURE 13
Authentication

HANYA kerjakan jika seluruh P0 dan P1 sudah selesai.

Implement basic login.

Endpoint:

POST /api/auth/login

Gunakan environment variable atau seed user.

Commit:

feat(auth): add basic authentication

FEATURE 14
Docker Integration

Jika seluruh aplikasi sudah stabil:

- backend Dockerfile
- simulator Dockerfile
- frontend Dockerfile jika diperlukan
- docker-compose integration

Pastikan seluruh stack dapat dijalankan dengan:

docker compose up

Commit:

feat(devops): add application containers

FEATURE 15
Documentation

Update README:

- architecture
- setup
- environment variables
- MQTT topic
- payload
- database schema
- API
- alert rules
- simulator
- troubleshooting
- production scalability

Commit:

docs(readme): document IoT monitoring system

# ================================================== 20. GIT SAFETY RULES

WAJIB:

Sebelum setiap commit:

git status
git diff

Pastikan hanya perubahan yang berhubungan dengan feature tersebut yang masuk commit.

JANGAN:

- git add .
  jika terdapat perubahan unrelated yang belum selesai

Lebih baik gunakan:

git add <specific-files>

Jika perubahan memang seluruhnya berasal dari feature tersebut, git add . diperbolehkan setelah melakukan git diff.

Jangan commit:

- .env
- password
- API key
- private key
- secret
- credential
- generated temporary files
- node_modules
- binary/build artifacts

Pastikan .gitignore sudah benar.

# ================================================== 21. COMMIT QUALITY

Setiap commit harus:

- buildable jika memungkinkan
- tidak meninggalkan kode yang rusak
- memiliki scope yang jelas
- memiliki commit message yang deskriptif
- tidak mencampurkan unrelated changes

Hindari commit seperti:

"update"
"fix"
"changes"
"final"
"done"

Gunakan Conventional Commit.

# ================================================== 22. PUSH RULE

Setelah commit berhasil:

git push

Jika branch pertama kali dipush:

git push -u origin feature/iot-monitoring

Setelah push berhasil, tampilkan:

FEATURE COMPLETED

Feature:
<feature name>

Commit:
<commit hash>

Commit message:
<commit message>

Push:
SUCCESS

Kemudian baru lanjut ke feature berikutnya.

# ================================================== 23. FAILURE RULE

Jika test/build gagal:

JANGAN commit.

Perbaiki terlebih dahulu.

Workflow:

Implement
→ Test
→ FAIL
→ Debug
→ Test ulang
→ PASS
→ git diff
→ Commit
→ Push

Jika masalah membutuhkan perubahan architecture besar, STOP dan jelaskan masalahnya sebelum melakukan perubahan besar.

# ================================================== 24. DO NOT OVER-IMPLEMENT

Jika feature sudah memenuhi requirement, STOP.

Jangan menambahkan:

- unnecessary abstraction
- unnecessary library
- extra database
- unnecessary service
- unnecessary endpoint
- unnecessary UI
- unnecessary authentication
- unnecessary infrastructure

Prioritaskan working end-to-end system daripada jumlah fitur.

# ================================================== 25. FINAL DEMO CHECK

Sebelum menyatakan project selesai, pastikan flow berikut berhasil:

Go IoT Simulator
↓
MQTT
↓
HiveMQ
↓
Go MQTT Consumer
↓
PostgreSQL
↓
REST API
↓
Svelte Dashboard

Dan demonstrasikan:

1. Device mengirim telemetry
2. Telemetry masuk PostgreSQL
3. Device muncul di dashboard
4. Fuel berubah
5. Temperature berubah
6. Device status berubah
7. Low fuel menghasilkan alert
8. Historical telemetry dapat dilihat

Setelah semuanya berhasil:

git status

Pastikan working tree bersih.

Kemudian:

git log --oneline --decorate -n 15

Tampilkan daftar commit feature yang sudah dibuat.
