package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type MetricRecord struct {
	ID          int64
	Timestamp   string
	CPUPercent  float64
	MemPercent  float64
	MemUsedMB   float64
	MemTotalMB  float64
	DiskPercent float64
	LoadAvg     float64
}

type DB struct {
	conn *sql.DB
}

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
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		cpu_percent REAL,
		mem_percent REAL,
		mem_used_mb REAL,
		mem_total_mb REAL,
		disk_percent REAL,
		load_avg REAL
	);
	CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
	`
	if _, err := conn.Exec(schema); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) InsertMetric(m MetricRecord) error {
	_, err := db.conn.Exec(
		`INSERT INTO metrics (cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		m.CPUPercent, m.MemPercent, m.MemUsedMB, m.MemTotalMB, m.DiskPercent, m.LoadAvg,
	)
	return err
}

func (db *DB) GetRecent(limit int) ([]MetricRecord, error) {
	rows, err := db.conn.Query(
		`SELECT id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg
		 FROM metrics ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []MetricRecord
	for rows.Next() {
		var m MetricRecord
		if err := rows.Scan(&m.ID, &m.Timestamp, &m.CPUPercent, &m.MemPercent, &m.MemUsedMB, &m.MemTotalMB, &m.DiskPercent, &m.LoadAvg); err != nil {
			return nil, err
		}
		results = append(results, m)
	}

	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results, nil
}

func (db *DB) GetLatest() (*MetricRecord, error) {
	var m MetricRecord
	row := db.conn.QueryRow(
		`SELECT id, timestamp, cpu_percent, mem_percent, mem_used_mb, mem_total_mb, disk_percent, load_avg
		 FROM metrics ORDER BY id DESC LIMIT 1`,
	)
	err := row.Scan(&m.ID, &m.Timestamp, &m.CPUPercent, &m.MemPercent, &m.MemUsedMB, &m.MemTotalMB, &m.DiskPercent, &m.LoadAvg)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
