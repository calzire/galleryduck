package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

type themeRequest struct {
	Theme string `json:"theme"`
}

func (s *Server) themeHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req themeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	theme := strings.ToLower(strings.TrimSpace(req.Theme))
	if theme != "light" && theme != "dark" && theme != "system" {
		http.Error(w, "invalid theme", http.StatusBadRequest)
		return
	}

	cfg := s.gallery.Config()
	cfg.Theme = theme
	if err := s.gallery.UpdateConfig(cfg); err != nil {
		http.Error(w, "failed to save theme", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "theme": theme})
}
