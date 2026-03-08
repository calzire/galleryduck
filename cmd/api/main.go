package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"galleryduck/internal/netinfo"
	"galleryduck/internal/qr"
	"galleryduck/internal/transport/http"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop() // Allow Ctrl+C to force shutdown

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	// Notify the main goroutine that the shutdown is complete
	done <- true
}

func logServerURLs(addr string) {
	localURL, lanURL, hasLAN := netinfo.URLs(addr)
	log.Printf("Local: %s", localURL)
	if !hasLAN {
		log.Println("LAN: IP not detected, using localhost only")
		return
	}

	log.Printf("LAN:   %s", lanURL)

	qrText, err := qr.ASCII(lanURL)
	if err != nil {
		log.Printf("LAN QR: unavailable (%v)", err)
		return
	}

	log.Println("LAN QR:")
	fmt.Print(qrText)
}

func main() {

	apiServer := server.New()
	logServerURLs(apiServer.Addr)

	// Create a done channel to signal when the shutdown is complete
	done := make(chan bool, 1)

	// Run graceful shutdown in a separate goroutine
	go gracefulShutdown(apiServer, done)

	err := apiServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	// Wait for the graceful shutdown to complete
	<-done
	log.Println("Graceful shutdown complete.")
}
