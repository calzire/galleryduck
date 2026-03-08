package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port <= 0 {
		port = 8080
	}
	srv := &Server{
		port: port,

		db: db.New(),
	}
	gallerySvc, err := gallery.NewService()
	if err != nil {
		log.Printf("gallery init failed: %v", err)
	} else {
		srv.gallery = gallerySvc
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
