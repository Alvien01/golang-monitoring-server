package main

import (
	"log"
	"net/http"
	"time"

	"monitoring-app/internal/config"
	"monitoring-app/internal/database"
	"monitoring-app/internal/handler"
	"monitoring-app/internal/metrics"
	"monitoring-app/internal/sse"
	"monitoring-app/internal/sshclient"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Gagal load config.yaml: %v", err)
	}

	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Gagal init database: %v", err)
	}
	defer db.Close()

	// Inisialisasi SSE Hub untuk push data real-time ke browser
	sseHub := sse.NewHub()

	// Jalankan scheduler housekeeping (data retention)
	go startHousekeeping(cfg.RetentionDays, db)

	// Jalankan scheduler SSH monitor untuk setiap server secara paralel
	for _, s := range cfg.Servers {
		go startServerScheduler(s, cfg.CheckIntervalSeconds, db, sseHub)
	}

	// Setup web server & handler
	h, err := handler.New(db, "web/templates/index.html", cfg)
	if err != nil {
		log.Fatalf("Gagal load template: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("GET /api/servers", h.APIServers)
	mux.HandleFunc("GET /api/recent", h.APIRecent)
	mux.HandleFunc("GET /api/latest", h.APILatest)
	mux.Handle("GET /api/events", sseHub)

	// Terapkan middleware Basic Auth jika diaktifkan di config
	var rootHandler http.Handler = mux
	if cfg.Auth.Enabled {
		log.Printf("🔒 Web Dashboard dilindungi Basic Auth (User: %s)", cfg.Auth.Username)
		rootHandler = h.WithAuth(mux)
	}

	log.Printf("✅ Dashboard jalan di http://localhost:%s", cfg.WebPort)
	log.Printf("🔍 Memonitor %d server (cek tiap %ds, data retention %d hari)", len(cfg.Servers), cfg.CheckIntervalSeconds, cfg.RetentionDays)
	for _, s := range cfg.Servers {
		log.Printf("   -> [%s] %s (%s:%s)", s.ID, s.Name, s.Host, s.Port)
	}

	log.Fatal(http.ListenAndServe(":"+cfg.WebPort, rootHandler))
}

// startServerScheduler memonitor satu server secara berkala
func startServerScheduler(s config.ServerConfig, intervalSec int, db *database.DB, hub *sse.Hub) {
	interval := time.Duration(intervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Jalankan pengecekan pertama segera
	checkServer(s, db, hub)

	for range ticker.C {
		checkServer(s, db, hub)
	}
}

// checkServer menjalankan single compound command via SSH lalu menyimpan & membroadcast hasilnya
func checkServer(s config.ServerConfig, db *database.DB, hub *sse.Hub) {
	client, err := sshclient.Connect(s)
	if err != nil {
		log.Printf("⚠️  [%s] Gagal konek SSH: %v", s.ID, err)
		hub.Broadcast("server_status", map[string]interface{}{
			"server_id": s.ID,
			"online":    false,
			"error":     err.Error(),
		})
		return
	}
	defer client.Close()

	// Eksekusi seluruh command monitoring sekaligus dalam 1 sesi SSH (Compound Command)
	out, err := sshclient.RunCommand(client, metrics.CompoundSSHCommand())
	if err != nil {
		log.Printf("⚠️  [%s] Gagal eksekusi command SSH: %v", s.ID, err)
		hub.Broadcast("server_status", map[string]interface{}{
			"server_id": s.ID,
			"online":    false,
			"error":     err.Error(),
		})
		return
	}

	// Parse seluruh metrics dari compound output
	data := metrics.ParseCompoundOutput(out)

	record := database.MetricRecord{
		ServerID:     s.ID,
		CPUPercent:   data.CPUPercent,
		MemPercent:   data.MemPercent,
		MemUsedMB:    data.MemUsedMB,
		MemTotalMB:   data.MemTotalMB,
		DiskPercent:  data.DiskPercent,
		LoadAvg:      data.LoadAvg,
		Uptime:       data.Uptime,
		TopProcesses: metrics.EncodeProcessesToJSON(data.TopProcesses),
	}

	if err := db.InsertMetric(record); err != nil {
		log.Printf("⚠️  [%s] Gagal simpan metric: %v", s.ID, err)
		return
	}

	// Ambil kembali snapshot dengan timestamp dari DB untuk konsistensi SSE
	latestRecord, err := db.GetLatest(s.ID)
	if err == nil && latestRecord != nil {
		record = *latestRecord
	}

	// Broadcast snapshot baru ke seluruh browser yang sedang terhubung via SSE
	hub.Broadcast("metric", record)

	log.Printf("✔ [%s] CPU: %.1f%% | Mem: %.1f%% | Disk: %.1f%% | Load: %.2f | Up: %s",
		s.ID, record.CPUPercent, record.MemPercent, record.DiskPercent, record.LoadAvg, record.Uptime)
}

// startHousekeeping membersihkan data metrics lama di SQLite
func startHousekeeping(retentionDays int, db *database.DB) {
	if retentionDays <= 0 {
		return
	}

	// Jalankan sekali saat startup
	if deleted, err := db.CleanupOldMetrics(retentionDays); err == nil && deleted > 0 {
		log.Printf("🧹 [Housekeeping] Membersihkan %d record metrik lama (> %d hari)", deleted, retentionDays)
	}

	// Bersihkan setiap 6 jam sekali
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		if deleted, err := db.CleanupOldMetrics(retentionDays); err == nil && deleted > 0 {
			log.Printf("🧹 [Housekeeping] Membersihkan %d record metrik lama (> %d hari)", deleted, retentionDays)
		}
	}
}
