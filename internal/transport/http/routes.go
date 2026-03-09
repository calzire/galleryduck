package server

import (
	"encoding/json"
	"log"
	"net/http"

	"galleryduck/internal/web"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/hello-world", s.HelloWorldHandler)
	mux.HandleFunc("/api/health", s.healthHandler)
	mux.HandleFunc("/api/media", s.listMediaHandler)
	mux.HandleFunc("/api/media/file", s.mediaFileHandler)
	mux.HandleFunc("/api/theme", s.themeHandler)
	mux.HandleFunc("/api/slideshow/items", s.slideshowItemsHandler)
	mux.HandleFunc("/api/index/rebuild", s.rebuildIndexHandler)
	mux.HandleFunc("/api/qr.svg", s.qrImageHandler)
	mux.HandleFunc("/media/list", s.mediaListFragmentHandler)
	mux.HandleFunc("/media/chunk", s.mediaChunkFragmentHandler)
	mux.HandleFunc("/media/subtypes", s.mediaSubTypesFragmentHandler)

	fileServer := http.FileServer(http.FS(web.Files))
	mux.Handle("/assets/", fileServer)
	mux.HandleFunc("/favicon.ico", s.faviconHandler)
	mux.HandleFunc("/settings", s.settingsHandler)
	mux.HandleFunc("/settings/login", s.settingsLoginHandler)
	mux.HandleFunc("/settings/logout", s.settingsLogoutHandler)
	mux.HandleFunc("/slideshow", s.slideshowHandler)
	mux.HandleFunc("/", s.homeHandler)

	// Wrap the mux with CORS middleware
	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Replace "*" with specific origins if needed
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Set to "true" if credentials are required

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{"message": "Hello World"}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := json.Marshal(s.db.Health())
	if err != nil {
		http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) faviconHandler(w http.ResponseWriter, r *http.Request) {
	icon, err := web.Files.ReadFile("assets/favicon.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if _, err := w.Write(icon); err != nil {
		log.Printf("Failed to write favicon: %v", err)
	}
}
