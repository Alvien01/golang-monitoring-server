# Server Monitoring (Go + SSH + SQLite)

Aplikasi monitoring 1 server via SSH. Tinggal isi credential di `config.yaml`, jalankan, buka dashboard di browser.

## Struktur Project

```
monitoring-app/
├── main.go                        # entry point: scheduler + web server
├── config.yaml                    # isi credential server di sini
├── go.mod
├── internal/
│   ├── config/config.go           # load config.yaml
│   ├── sshclient/client.go        # koneksi & jalankan command SSH
│   ├── metrics/parser.go          # parse output command jadi angka
│   ├── database/sqlite.go         # simpan & ambil history metrics
│   └── handler/dashboard.go       # HTTP handler (halaman + API JSON)
├── web/templates/index.html       # dashboard (chart.js, auto-refresh 5 detik)
└── data/                          # folder database SQLite akan dibuat otomatis
```

## Langkah 1: Install Go

Pastikan Go sudah terinstall (`go version`). Kalau belum, download di https://go.dev/dl/

## Langkah 2: Download dependency

Dari dalam folder project ini, jalankan:

```bash
go mod tidy
```

Ini akan otomatis download 3 library yang dipakai:
- `golang.org/x/crypto/ssh` — untuk koneksi SSH
- `modernc.org/sqlite` — driver SQLite pure Go (tanpa CGO / tanpa butuh gcc)
- `gopkg.in/yaml.v3` — untuk baca file config.yaml

> **Catatan:** Project ini menggunakan driver `modernc.org/sqlite` yang merupakan implementasi pure Go, sehingga dapat berjalan langsung di Windows tanpa memerlukan compiler C (gcc/MinGW) ataupun mengaktifkan CGO (`CGO_ENABLED=0` tetap bisa berjalan).

## Langkah 3: Siapkan config.yaml

Salin file contoh konfigurasi lalu sesuaikan isinya:

```bash
# Di Windows PowerShell:
cp config.yaml.example config.yaml

# Atau di Linux/macOS:
cp config.yaml.example config.yaml
```

Buka `config.yaml`, isi sesuai server kamu:

```yaml
server:
  host: "192.168.1.100"
  port: "22"
  username: "root"
  password: "your_password_here"
  private_key_path: ""   # atau isi path private key kalau pakai key-based auth
```

## Langkah 4: Jalankan

```bash
go run main.go
```

Kalau berhasil, akan muncul log:
```
✅ Dashboard jalan di http://localhost:8080
🔍 Monitoring server: 192.168.1.100 (cek tiap 30s)
✔ CPU: 12.3% | Mem: 45.6% | Disk: 38.0% | Load: 0.52
```

Buka browser ke **http://localhost:8080** untuk lihat dashboard.

## Langkah 5 (opsional): Compile jadi 1 binary

```bash
go build -o monitoring-app.exe main.go
```

Lalu jalankan `monitoring-app.exe` — tetap butuh file `config.yaml` dan folder `web/` ada di direktori yang sama.

## Cara Kerja Singkat

1. `main.go` jalankan 2 hal secara paralel: **scheduler** (goroutine, cek server tiap interval) dan **web server** (serve dashboard + API).
2. Scheduler konek SSH → jalankan command (`top`, `free`, `df`, `cat /proc/loadavg`) → parse hasilnya → simpan ke SQLite.
3. Dashboard (browser) polling API `/api/latest` dan `/api/recent` tiap 5 detik untuk update angka & grafik.

## Yang Perlu Disesuaikan / Dikembangkan Lagi

- **Keamanan:** saat ini `HostKeyCallback: ssh.InsecureIgnoreHostKey()` dipakai supaya simpel — untuk pemakaian jangka panjang sebaiknya verifikasi host key asli, dan sebaiknya pakai **private key** daripada password.
- **Password di config.yaml** masih plain text — kalau mau lebih aman, bisa dienkripsi atau dipindah ke environment variable.
- **Alerting:** belum ada notifikasi kalau CPU/disk mendekati penuh — bisa ditambah pengecekan threshold di `checkServer()` lalu kirim ke Telegram/email/Slack.
- **Multi-server:** kalau nanti mau monitoring lebih dari 1 server, struktur `config.yaml` dan tabel `metrics` perlu ditambah kolom/field `server_id`.
