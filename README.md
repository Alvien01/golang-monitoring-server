# PulseNode - Multi-Server Monitoring (Go + SSH + SQLite)

Aplikasi monitoring multi-server via SSH dengan antarmuka web modern, streaming data real-time via Server-Sent Events (SSE), inspeksi proses, proteksi Basic Auth, dan housekeeping data otomatis di SQLite.

## Fitur Utama

- 🌐 **Multi-Server Monitoring**: Monitor banyak server secara bersamaan dengan scheduler paralel per-server.
- ⚡ **Real-Time SSE Streaming**: Data telemetri langsung ter-push ke browser (*zero polling latency*).
- 🚀 **Optimasi SSH (Compound Command)**: Menggabungkan perintah monitoring ke dalam 1 sesi eksekusi SSH tunggal (hemat bandwidth dan 5x lebih cepat).
- 🔥 **Inspeksi Top 5 Processes**: Menampilkan proses yang mengonsumsi CPU/RAM tertinggi beserta PID & user.
- ⏱️ **Server Uptime & Resource Cards**: CPU %, RAM (used/total), Disk (root partition), Load Average, dan status server.
- 🔒 **Keamanan & Kredensial**:
  - Dukungan ekspansi Environment Variable `${SSH_PASSWORD}` di `config.yaml`.
  - Dukungan Private Key authentication (termasuk passphrase).
  - Verifikasi sidik jari SSH Host Key asli via `known_hosts`.
  - HTTP Basic Auth opsional untuk melindungi dashboard & API.
- 🧹 **Data Retention Policy**: Otomatis membersihkan riwayat metrik lama di SQLite (`retention_days`).
- 💎 **Pure Go SQLite**: Tidak membutuhkan compiler C (gcc/MinGW) maupun CGO di Windows/Linux/macOS.

---

## Struktur Project

```
monitoring-app/
├── main.go                        # entry point: multi-server scheduler + SSE + web server
├── config.yaml                    # konfigurasi server, auth, dan retensi (di-ignore git)
├── config.yaml.example            # contoh konfigurasi untuk git
├── go.mod
├── internal/
│   ├── config/config.go           # load & validasi config.yaml + env vars
│   ├── sshclient/client.go        # SSH client, auth methods, host key verification
│   ├── metrics/parser.go          # compound command generator & regex parser
│   ├── database/sqlite.go         # SQLite pure Go, auto-migration & housekeeping
│   ├── sse/hub.go                 # Server-Sent Events thread-safe broadcast hub
│   └── handler/dashboard.go       # HTTP handler, API endpoints, dan Basic Auth
├── web/templates/index.html       # modern dark dashboard (Chart.js + SSE client)
└── data/                          # folder database SQLite otomatis dibuat
```

---

## Langkah 1: Persiapan Konfigurasi

Salin file contoh konfigurasi lalu sesuaikan isinya:

```bash
# Di Windows PowerShell:
cp config.yaml.example config.yaml

# Di Linux / macOS:
cp config.yaml.example config.yaml
```

Buka `config.yaml`, sesuaikan daftar server yang ingin Anda monitor:

```yaml
servers:
  - id: "vps-main"
    name: "Production VPS"
    host: "103.163.139.113"
    port: "22"
    username: "root"
    password: "your_password_here" # atau pakai ${SSH_PASSWORD}
    private_key_path: ""
    private_key_passphrase: ""
    insecure_ignore_host_key: true
    known_hosts_path: ""

# Keamanan Dashboard Web (HTTP Basic Auth)
auth:
  enabled: false # set true jika ingin dashboard diproteksi login
  username: "admin"
  password: "password123"

# Retensi data otomatis (hari)
retention_days: 7

# Interval pengecekan (detik)
check_interval_seconds: 30

web_port: "8080"
database_path: "./data/monitoring.db"
```

---

## Langkah 2: Jalankan Aplikasi

```bash
go run main.go
```

Output di terminal:
```text
✅ Dashboard jalan di http://localhost:8080
🔍 Memonitor 1 server (cek tiap 30s, data retention 7 hari)
   -> [vps-main] Production VPS (103.163.139.113:22)
✔ [vps-main] CPU: 12.3% | Mem: 45.6% | Disk: 38.0% | Load: 0.52 | Up: 4 days, 2 hours
```

Buka browser ke **http://localhost:8080** untuk mengakses dashboard.

---

## Endpoint API yang Tersedia

| Endpoint | Method | Keterangan |
|---|---|---|
| `/` | GET | Halaman web dashboard interaktif |
| `/api/servers` | GET | Daftar seluruh server yang dikonfigurasi beserta status terkini |
| `/api/latest?server_id=ID` | GET | Snapshot metrik terbaru server tertentu |
| `/api/recent?server_id=ID&limit=60` | GET | Riwayat metrik untuk grafik chart |
| `/api/events` | GET | Server-Sent Events (SSE) stream untuk update real-time |

---

## Rencana Pengembangan Selanjutnya

- **Alerting System**: Notifikasi via Telegram Bot, Discord Webhook, Slack, atau Email ketika CPU/RAM/Disk melebihi batas toleransi.
- **Service & Container Monitoring**: Memantau status service `systemd` (Nginx, Docker, PostgreSQL, MySQL) atau kontainer Docker yang aktif.
- **Bandwidth / Network I/O**: Memantau grafik kecepatan download/upload secara real-time via `/proc/net/dev`.
