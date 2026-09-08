package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type MetricRecord struct {
	ID           int64   `json:"id"`
	ServerID     string  `json:"server_id"`
	Timestamp    string  `json:"timestamp"`
	CPUPercent   float64 `json:"cpu_percent"`
	MemPercent   float64 `json:"mem_percent"`
	MemUsedMB    float64 `json:"mem_used_mb"`
	MemTotalMB   float64 `json:"mem_total_mb"`
	DiskPercent  float64 `json:"disk_percent"`
	LoadAvg      float64 `json:"load_avg"`
	Uptime       string  `json:"uptime"`
	TopProcesses string  `json:"top_processes"` // JSON string representation
}

type DB struct {
	conn *sql.DB
}

// Init membuka database SQLite, membuat tabel dan migrasi kolom jika belum ada
func Init(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id TEXT NOT NULL DEFAULT 'default',
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		cpu_percent REAL,
		mem_percent REAL,
		mem_used_mb REAL,
		mem_total_mb REAL,
		disk_percent REAL,
		load_avg REAL,
		uptime TEXT DEFAULT '',
		top_processes TEXT DEFAULT '[]'
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_server_timestamp ON metrics(server_id, timestamp);
	`
	if _, err := conn.Exec(schema); err != nil {
		return nil, err
	}

	// Migrasi otomatis jika tabel lama belum memiliki kolom baru
	if err := migrateColumns(conn); err != nil {
		return nil, fmt.Errorf("gagal migrasi kolom database: %w", err)
	}

	return &DB{conn: conn}, nil
}

func migrateColumns(conn *sql.DB) error {
	rows, err := conn.Query("PRAGMA table_info(metrics)")
	if err != nil {
		return err
	}
	defer rows.Close()

	existingCols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
			existingCols[strings.ToLower(name)] = true
		}
	}

	if !existingCols["server_id"] {
		if _, err := conn.Exec("ALTER TABLE metrics ADD COLUMN server_id TEXT NOT NULL DEFAULT 'default'"); err != nil {
			return err
		}
	}
	if !existingCols["uptime"] {
		if _, err := conn.Exec("ALTER TABLE metrics ADD COLUMN uptime TEXT DEFAULT ''"); err != nil {
			return err
		}
	}
	if !existingCols["top_processes"] {
		if _, err := conn.Exec("ALTER TABLE metrics ADD COLUMN top_processes TEXT DEFAULT '[]'"); err != nil {
			return err
		}
	}

	// Buat index gabungan server_id & timestamp
	conn.Exec("CREATE INDEX IF NOT EXISTS idx_metrics_server_timestamp ON metrics(server_id, timestamp)")
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// InsertMetric menyimpan satu snapshot metrics baru
func (db *DB) InsertMetric(m MetricRecord) error {
	if m.ServerID == "" {
		m.ServerID = "default"
	}
	if m.TopProcesses == "" {
		m.TopProcesses = "[]"
	}
	_, err := db.conn.Exec(
		`INSERT INTO metrics (server_id, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg, uptime, top_processes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ServerID, m.CPUPercent, m.MemPercent, m.MemUsedMB, m.MemTotalMB, m.DiskPercent, m.LoadAvg, m.Uptime, m.TopProcesses,
	)
	return err
}

// GetRecent mengambil record metrics terbaru berdasarkan serverID
func (db *DB) GetRecent(serverID string, limit int) ([]MetricRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	var rows *sql.Rows
	var err error

	if serverID != "" {
		rows, err = db.conn.Query(
			`SELECT id, server_id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg, coalesce(uptime, ''), coalesce(top_processes, '[]')
			 FROM metrics WHERE server_id = ? ORDER BY id DESC LIMIT ?`, serverID, limit,
		)
	} else {
		rows, err = db.conn.Query(
			`SELECT id, server_id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg, coalesce(uptime, ''), coalesce(top_processes, '[]')
			 FROM metrics ORDER BY id DESC LIMIT ?`, limit,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MetricRecord
	for rows.Next() {
		var m MetricRecord
		if err := rows.Scan(&m.ID, &m.ServerID, &m.Timestamp, &m.CPUPercent, &m.MemPercent, &m.MemUsedMB, &m.MemTotalMB, &m.DiskPercent, &m.LoadAvg, &m.Uptime, &m.TopProcesses); err != nil {
			return nil, err
		}
		results = append(results, m)
	}

	// Balik urutan supaya dari lama -> baru (untuk chart)
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
}

// GetLatest mengambil snapshot metrics paling baru untuk server tertentu
func (db *DB) GetLatest(serverID string) (*MetricRecord, error) {
	var row *sql.Row
	if serverID != "" {
		row = db.conn.QueryRow(
			`SELECT id, server_id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg, coalesce(uptime, ''), coalesce(top_processes, '[]')
			 FROM metrics WHERE server_id = ? ORDER BY id DESC LIMIT 1`, serverID,
		)
	} else {
		row = db.conn.QueryRow(
			`SELECT id, server_id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg, coalesce(uptime, ''), coalesce(top_processes, '[]')
			 FROM metrics ORDER BY id DESC LIMIT 1`,
		)
	}

	var m MetricRecord
	err := row.Scan(&m.ID, &m.ServerID, &m.Timestamp, &m.CPUPercent, &m.MemPercent, &m.MemUsedMB, &m.MemTotalMB, &m.DiskPercent, &m.LoadAvg, &m.Uptime, &m.TopProcesses)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CleanupOldMetrics menghapus record metrics yang lebih tua dari N hari
func (db *DB) CleanupOldMetrics(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	query := fmt.Sprintf("DELETE FROM metrics WHERE timestamp < datetime('now', '-%d days')", retentionDays)
	res, err := db.conn.Exec(query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
