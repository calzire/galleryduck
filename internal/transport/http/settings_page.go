package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"galleryduck/internal/app/gallery"
	"galleryduck/internal/netinfo"
	"galleryduck/internal/qr"
	webpages "galleryduck/internal/web/pages"
)

const (
	settingsAuthCookieName     = "galleryduck_settings_auth"
	settingsAuthCookieLifetime = 24 * time.Hour
)

func (s *Server) settingsHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}
	cfg := s.gallery.Config()
	requireAuth := isSettingsPasswordConfigured(cfg)

	switch r.Method {
	case http.MethodGet:
		if requireAuth && !s.isSettingsAuthorized(r, cfg) {
			s.renderSettingsLoginPage(w, r, "")
			return
		}
		s.renderSettingsPage(w, r, "", false)
	case http.MethodPost:
		if requireAuth && !s.isSettingsAuthorized(r, cfg) {
			s.renderSettingsLoginPage(w, r, "sign in to access settings")
			return
		}
		s.handleSettingsSave(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) settingsLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	cfg := s.gallery.Config()
	if !isSettingsPasswordConfigured(cfg) {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.renderSettingsLoginPage(w, r, "invalid login form")
		return
	}

	password := strings.TrimSpace(r.FormValue("password"))
	if !verifySettingsPassword(cfg, password) {
		s.clearSettingsAuthCookie(w)
		s.renderSettingsLoginPage(w, r, "incorrect password")
		return
	}

	s.setSettingsAuthCookie(w, cfg)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *Server) settingsLogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}

	s.clearSettingsAuthCookie(w)
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
	passwordChanged := false
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
	newPassword := strings.TrimSpace(r.FormValue("settings_password"))
	confirmPassword := strings.TrimSpace(r.FormValue("settings_password_confirm"))
	if newPassword != "" || confirmPassword != "" {
		if newPassword == "" || confirmPassword == "" {
			s.renderSettingsPage(w, r, "both password fields are required", false)
			return
		}
		if len(newPassword) < 6 {
			s.renderSettingsPage(w, r, "settings password must be at least 6 characters", false)
			return
		}
		if newPassword != confirmPassword {
			s.renderSettingsPage(w, r, "password confirmation does not match", false)
			return
		}

		salt, hash, err := hashSettingsPassword(newPassword)
		if err != nil {
			s.renderSettingsPage(w, r, "failed to update settings password", false)
			return
		}
		cfg.SettingsPasswordSalt = salt
		cfg.SettingsPasswordHash = hash
		passwordChanged = true
	}

	if err := s.gallery.UpdateConfig(cfg); err != nil {
		s.renderSettingsPage(w, r, err.Error(), false)
		return
	}
	if passwordChanged {
		s.setSettingsAuthCookie(w, cfg)
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
		RequireAuth:     false,
		PasswordSet:     isSettingsPasswordConfigured(cfg),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.SettingsPage(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render settings", http.StatusInternalServerError)
	}
}

func (s *Server) renderSettingsLoginPage(w http.ResponseWriter, r *http.Request, errMessage string) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}
	cfg := s.gallery.Config()
	data := webpages.SettingsPageData{
		ConfigPath:   s.gallery.ConfigPath(),
		Theme:        cfg.Theme,
		RequireAuth:  true,
		HasError:     errMessage != "",
		ErrorMessage: errMessage,
		PasswordSet:  true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.SettingsPage(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render settings", http.StatusInternalServerError)
	}
}

func isSettingsPasswordConfigured(cfg gallery.Config) bool {
	return strings.TrimSpace(cfg.SettingsPasswordSalt) != "" && strings.TrimSpace(cfg.SettingsPasswordHash) != ""
}

func hashSettingsPassword(password string) (string, string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", "", err
	}
	salt := base64.RawURLEncoding.EncodeToString(saltBytes)
	return salt, hashWithSalt(password, salt), nil
}

func hashWithSalt(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func verifySettingsPassword(cfg gallery.Config, password string) bool {
	if !isSettingsPasswordConfigured(cfg) || password == "" {
		return false
	}
	expected := cfg.SettingsPasswordHash
	actual := hashWithSalt(password, cfg.SettingsPasswordSalt)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func settingsAuthKey(cfg gallery.Config) []byte {
	key := sha256.Sum256([]byte(cfg.SettingsPasswordSalt + ":" + cfg.SettingsPasswordHash))
	return key[:]
}

func makeSettingsAuthValue(cfg gallery.Config, expiry int64) string {
	payload := strconv.FormatInt(expiry, 10)
	mac := hmac.New(sha256.New, settingsAuthKey(cfg))
	_, _ = mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func verifySettingsAuthValue(cfg gallery.Config, value string) bool {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return false
	}

	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix() > expiry {
		return false
	}

	expected := makeSettingsAuthValue(cfg, expiry)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(value)) == 1
}

func (s *Server) isSettingsAuthorized(r *http.Request, cfg gallery.Config) bool {
	if !isSettingsPasswordConfigured(cfg) {
		return true
	}
	cookie, err := r.Cookie(settingsAuthCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	return verifySettingsAuthValue(cfg, cookie.Value)
}

func (s *Server) setSettingsAuthCookie(w http.ResponseWriter, cfg gallery.Config) {
	expiry := time.Now().Add(settingsAuthCookieLifetime)
	http.SetCookie(w, &http.Cookie{
		Name:     settingsAuthCookieName,
		Value:    makeSettingsAuthValue(cfg, expiry.Unix()),
		Path:     "/settings",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(settingsAuthCookieLifetime.Seconds()),
		Expires:  expiry,
	})
}

func (s *Server) clearSettingsAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     settingsAuthCookieName,
		Value:    "",
		Path:     "/settings",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
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
