package handler

import (
	"crypto/subtle"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"monitoring-app/internal/config"
	"monitoring-app/internal/database"
)

type Handler struct {
	db   *database.DB
	tmpl *template.Template
	cfg  *config.Config
}

type ServerView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Host string `json:"host"`
	Port string `json:"port"`
}

func New(db *database.DB, templatePath string, cfg *config.Config) (*Handler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}
	return &Handler{db: db, tmpl: tmpl, cfg: cfg}, nil
}

// Index merender halaman dashboard HTML dengan daftar server
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	var servers []ServerView
	for _, s := range h.cfg.Servers {
		servers = append(servers, ServerView{
			ID:   s.ID,
			Name: s.Name,
			Host: s.Host,
			Port: s.Port,
		})
	}

	data := map[string]interface{}{
		"Servers": servers,
		"DefaultServerID": func() string {
			if len(servers) > 0 {
				return servers[0].ID
			}
			return "default"
		}(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.Execute(w, data)
}

// APIServers mengembalikan daftar server terkonfigurasi beserta status terkininya
func (h *Handler) APIServers(w http.ResponseWriter, r *http.Request) {
	type ServerStatus struct {
		ID     string                 `json:"id"`
		Name   string                 `json:"name"`
		Host   string                 `json:"host"`
		Port   string                 `json:"port"`
		Latest *database.MetricRecord `json:"latest,omitempty"`
	}

	var results []ServerStatus
	for _, s := range h.cfg.Servers {
		latest, _ := h.db.GetLatest(s.ID)
		results = append(results, ServerStatus{
			ID:     s.ID,
			Name:   s.Name,
			Host:   s.Host,
			Port:   s.Port,
			Latest: latest,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// APIRecent mengembalikan history metrics untuk serverID tertentu
func (h *Handler) APIRecent(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" && len(h.cfg.Servers) > 0 {
		serverID = h.cfg.Servers[0].ID
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	records, err := h.db.GetRecent(serverID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []database.MetricRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// APILatest mengembalikan snapshot metrics terbaru untuk serverID tertentu
func (h *Handler) APILatest(w http.ResponseWriter, r *http.Request) {
	serverID := r.URL.Query().Get("server_id")
	if serverID == "" && len(h.cfg.Servers) > 0 {
		serverID = h.cfg.Servers[0].ID
	}

	record, err := h.db.GetLatest(serverID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

// WithAuth membungkus handler dengan proteksi HTTP Basic Auth jika diaktifkan di config
func (h *Handler) WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.cfg.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(h.cfg.Auth.Username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(h.cfg.Auth.Password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Monitoring Dashboard"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
