package handler

import (
	"encoding/json"
	"html/template"
	"net/http"

	"monitoring-app/internal/database"
)

type Handler struct {
	db        *database.DB
	tmpl      *template.Template
	serverTag string
}

func New(db *database.DB, templatePath string, serverTag string) (*Handler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}
	return &Handler{db: db, tmpl: tmpl, serverTag: serverTag}, nil
}

// Index menampilkan halaman dashboard HTML
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{"ServerTag": h.serverTag}
	h.tmpl.Execute(w, data)
}

// APIRecent mengembalikan history metrics dalam format JSON (dipakai chart.js di frontend)
func (h *Handler) APIRecent(w http.ResponseWriter, r *http.Request) {
	records, err := h.db.GetRecent(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// APILatest mengembalikan snapshot metrics terbaru dalam format JSON
func (h *Handler) APILatest(w http.ResponseWriter, r *http.Request) {
	record, err := h.db.GetLatest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}
