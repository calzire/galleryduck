package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"galleryduck/internal/netinfo"
	"galleryduck/internal/qr"
	webpages "galleryduck/internal/web/pages"
)

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderSettingsPage(w, r, "", false)
	case http.MethodPost:
		s.handleSettingsSave(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSettingsSave(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderSettingsPage(w, r, "invalid form data", false)
		return
	}

	cfg := s.gallery.Config()
	prevPort := cfg.Port
	cfg.LibraryPaths = splitLines(r.FormValue("library_paths"))
	cfg.Port = s.port
	cfg.Theme = strings.TrimSpace(r.FormValue("theme"))
	cfg.DefaultSort = strings.TrimSpace(r.FormValue("default_sort"))
	cfg.DefaultView = strings.TrimSpace(r.FormValue("default_view"))
	cfg.PaginationMode = strings.TrimSpace(r.FormValue("pagination_mode"))
	cfg.Slideshow.Transition = strings.TrimSpace(r.FormValue("transition"))
	cfg.Slideshow.Autoplay = r.FormValue("autoplay") == "1"
	cfg.Slideshow.Loop = r.FormValue("loop") == "1"
	cfg.Slideshow.Fullscreen = r.FormValue("fullscreen") == "1"

	if speed := strings.TrimSpace(r.FormValue("speed_ms")); speed != "" {
		var parsed int
		_, _ = fmt.Sscanf(speed, "%d", &parsed)
		if parsed > 0 {
			cfg.Slideshow.SpeedMS = parsed
		}
	}
	if port := strings.TrimSpace(r.FormValue("port")); port != "" {
		parsed, err := strconv.Atoi(port)
		if err != nil || parsed < 1 || parsed > 65535 {
			s.renderSettingsPage(w, r, "port must be a number between 1 and 65535", false)
			return
		}
		cfg.Port = parsed
	}

	if err := s.gallery.UpdateConfig(cfg); err != nil {
		s.renderSettingsPage(w, r, err.Error(), false)
		return
	}

	redirectURL := "/settings?saved=1"
	if cfg.Port != prevPort {
		redirectURL += "&restart=1"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (s *Server) renderSettingsPage(w http.ResponseWriter, r *http.Request, errMessage string, saved bool) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	cfg := s.gallery.Config()
	localURL, lanURL, hasLAN := netinfo.URLs(fmt.Sprintf(":%d", s.port))
	lanQRURL := ""
	if hasLAN {
		lanQRURL = "/api/qr.svg?url=" + url.QueryEscape(lanURL)
	}

	data := webpages.SettingsPageData{
		ConfigPath:      s.gallery.ConfigPath(),
		LibraryPaths:    strings.Join(cfg.LibraryPaths, "\n"),
		Port:            cfg.Port,
		Theme:           cfg.Theme,
		DefaultSort:     cfg.DefaultSort,
		DefaultView:     cfg.DefaultView,
		Pagination:      cfg.PaginationMode,
		SpeedMS:         cfg.Slideshow.SpeedMS,
		Transition:      cfg.Slideshow.Transition,
		Autoplay:        cfg.Slideshow.Autoplay,
		Loop:            cfg.Slideshow.Loop,
		Fullscreen:      cfg.Slideshow.Fullscreen,
		MediaCount:      s.gallery.MediaCount(),
		LocalURL:        localURL,
		LANURL:          lanURL,
		LANQRURL:        lanQRURL,
		Saved:           saved || r.URL.Query().Get("saved") == "1",
		HasError:        errMessage != "",
		ErrorMessage:    errMessage,
		LibraryCount:    len(cfg.LibraryPaths),
		RestartRequired: r.URL.Query().Get("restart") == "1",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.SettingsPage(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render settings", http.StatusInternalServerError)
	}
}

func (s *Server) qrImageHandler(w http.ResponseWriter, r *http.Request) {
	payload := strings.TrimSpace(r.URL.Query().Get("url"))
	if payload == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}

	svg, err := qr.SVG(payload, 8)
	if err != nil {
		http.Error(w, "qr unavailable", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(svg))
}

func splitLines(value string) []string {
	raw := strings.Split(value, "\n")
	out := make([]string, 0, len(raw))
	for _, line := range raw {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
