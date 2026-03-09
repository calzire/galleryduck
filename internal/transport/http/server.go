package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"galleryduck/internal/app/gallery"
	_ "github.com/joho/godotenv/autoload"

	"galleryduck/internal/store/db"
)

type Server struct {
	port int

	db      db.Service
	gallery *gallery.Service
}

func New() *http.Server {
	srv := &Server{
		port: 8787,
		db:   db.New(),
	}
	gallerySvc, err := gallery.NewService()
	if err != nil {
		log.Printf("gallery init failed: %v", err)
	} else {
		srv.gallery = gallerySvc
		cfg := gallerySvc.Config()
		if cfg.Port >= 1 && cfg.Port <= 65535 {
			srv.port = cfg.Port
		}
	}

	if envPort := strings.TrimSpace(os.Getenv("PORT")); envPort != "" {
		port, err := strconv.Atoi(envPort)
		if err != nil || port < 1 || port > 65535 {
			log.Printf("invalid PORT value %q, using %d", envPort, srv.port)
		} else {
			srv.port = port
		}
	}

	// Declare Server config
	apiServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", srv.port),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return apiServer
}
