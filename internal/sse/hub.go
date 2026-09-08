package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Hub struct {
	mu         sync.RWMutex
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
}

func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 256),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client <- message:
				default:
					// Klien lambat, skip buffer
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast mengirim data dalam format SSE ke semua klien aktif
func (h *Hub) Broadcast(eventName string, payload interface{}) {
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("⚠️ Gagal marshal SSE event %s: %v", eventName, err)
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, string(dataBytes))
	h.broadcast <- []byte(msg)
}

// ServeHTTP menangani request koneksi SSE dari browser
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan []byte, 32)
	h.register <- clientChan
	defer func() {
		h.unregister <- clientChan
	}()

	// Kirim pesan sambutan awal
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	// Keepalive ping ticker setiap 15 detik
	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-pingTicker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			_, err := w.Write(msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
