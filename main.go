package main

import (
	"log"
	"net/http"
	"time"

	"monitoring-app/internal/config"
	"monitoring-app/internal/database"
	"monitoring-app/internal/handler"
	"monitoring-app/internal/metrics"
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

	// Jalankan scheduler SSH check di background (goroutine terpisah)
	go startScheduler(cfg, db)

	// Setup web server
	h, err := handler.New(db, "web/templates/index.html", cfg.Server.Host)
	if err != nil {
		log.Fatalf("Gagal load template: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Index)
	mux.HandleFunc("GET /api/recent", h.APIRecent)
	mux.HandleFunc("GET /api/latest", h.APILatest)

	log.Printf("✅ Dashboard jalan di http://localhost:%s", cfg.WebPort)
	log.Printf("🔍 Monitoring server: %s (cek tiap %ds)", cfg.Server.Host, cfg.CheckIntervalSeconds)
	log.Fatal(http.ListenAndServe(":"+cfg.WebPort, mux))
}

// startScheduler menjalankan pengecekan SSH secara berkala sesuai interval di config
func startScheduler(cfg *config.Config, db *database.DB) {
	interval := time.Duration(cfg.CheckIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// jalankan sekali di awal, jangan tunggu tick pertama
	checkServer(cfg, db)

	for range ticker.C {
		checkServer(cfg, db)
	}
}

// checkServer melakukan 1 siklus: konek SSH -> jalankan command -> parse -> simpan ke DB
func checkServer(cfg *config.Config, db *database.DB) {
	client, err := sshclient.Connect(cfg.Server)
	if err != nil {
		log.Printf("⚠️  Gagal konek SSH: %v", err)
		return
	}
	defer client.Close()

	cpuOut, err := sshclient.RunCommand(client, `top -bn1 | grep "Cpu(s)"`)
	if err != nil {
		log.Printf("⚠️  Gagal ambil CPU: %v", err)
		return
	}

	memOut, err := sshclient.RunCommand(client, "free -m")
	if err != nil {
		log.Printf("⚠️  Gagal ambil memory: %v", err)
		return
	}

	diskOut, err := sshclient.RunCommand(client, "df -h /")
	if err != nil {
		log.Printf("⚠️  Gagal ambil disk: %v", err)
		return
	}

	loadOut, err := sshclient.RunCommand(client, "cat /proc/loadavg")
	if err != nil {
		log.Printf("⚠️  Gagal ambil load average: %v", err)
		return
	}

	cpuPercent := metrics.ParseCPU(cpuOut)
	memUsed, memTotal, memPercent := metrics.ParseMem(memOut)
	diskPercent := metrics.ParseDisk(diskOut)
	loadAvg := metrics.ParseLoadAvg(loadOut)

	record := database.MetricRecord{
		CPUPercent:  cpuPercent,
		MemPercent:  memPercent,
		MemUsedMB:   memUsed,
		MemTotalMB:  memTotal,
		DiskPercent: diskPercent,
		LoadAvg:     loadAvg,
	}

	if err := db.InsertMetric(record); err != nil {
		log.Printf("⚠️  Gagal simpan metric: %v", err)
		return
	}

	log.Printf("✔ CPU: %.1f%% | Mem: %.1f%% | Disk: %.1f%% | Load: %.2f", cpuPercent, memPercent, diskPercent, loadAvg)
}
