package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"galleryduck/internal/app/gallery"
)

type mediaDTO struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	SubType string `json:"sub_type"`
	Date    string `json:"date"`
	ModTime string `json:"mod_time"`
	Src     string `json:"src"`
}

func (s *Server) listMediaHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	query := gallery.Query{
		Type:     strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type"))),
		SubTypes: normalizeSubTypes(r.URL.Query()["sub_type"]),
		Year:     gallery.ParseQueryInt(r.URL.Query().Get("year"), 0),
		Date:     strings.TrimSpace(r.URL.Query().Get("date")),
		Search:   strings.TrimSpace(r.URL.Query().Get("search")),
		Sort:     strings.TrimSpace(r.URL.Query().Get("sort")),
		Order:    strings.TrimSpace(r.URL.Query().Get("order")),
		Page:     gallery.ParseQueryInt(r.URL.Query().Get("page"), 1),
		PageSize: gallery.ParseQueryInt(r.URL.Query().Get("page_size"), 24),
	}

	items, total := s.gallery.QueryMedia(query)
	respItems := make([]mediaDTO, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, mediaDTO{
			Path:    item.Path,
			Name:    item.Name,
			Type:    item.Type,
			SubType: item.SubType,
			Date:    item.Date.Format("2006-01-02"),
			ModTime: item.ModTime.Format(time.RFC3339),
			Src:     "/api/media/file?path=" + url.QueryEscape(item.Path),
		})
	}

	response := map[string]any{
		"items":     respItems,
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func normalizeSubTypes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		subType := strings.ToLower(strings.TrimSpace(value))
		if subType == "" {
			continue
		}
		if _, ok := seen[subType]; ok {
			continue
		}
		seen[subType] = struct{}{}
		out = append(out, subType)
	}
	return out
}

func (s *Server) rebuildIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	if err := s.gallery.Rescan(); err != nil {
		http.Error(w, "failed to rebuild index", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"media_count": s.gallery.MediaCount(),
	})
}

func (s *Server) mediaFileHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if err := s.ensurePathAllowed(absPath); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if _, err := os.Stat(absPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "cannot read file", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, absPath)
}

func (s *Server) ensurePathAllowed(absPath string) error {
	roots := s.gallery.LibraryPaths()
	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && rel != "..") {
			return nil
		}
	}
	return errors.New("path outside configured roots")
}
