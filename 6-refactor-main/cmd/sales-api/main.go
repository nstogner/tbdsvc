package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/nstogner/tbdsvc/6-refactor-main/cmd/sales-api/internal/handlers"
	"github.com/pkg/errors"
)

func main() {
	cfg := configure()

	svr, teardown := initialize(cfg)
	defer teardown()

	// Launch the server in a goroutine and provide a channel to capture any
	// startup errors.
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- svr.ListenAndServe()
	}()

	// Create a channel to listen for interrupt signals from the OS.
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	// Block waiting for an error from the server or a shutdown signal.
	select {
	case err := <-serverErrors:
		log.Fatal(errors.Wrap(err, "listening and serving"))
	case <-osSignals:
		log.Print("caught signal, shutting down")

		// Give outstanding requests 30 seconds to complete.
		const timeout = 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := svr.Shutdown(ctx); err != nil {
			log.Printf("error: %s", errors.Wrap(err, "shutting down server"))
			if err := svr.Close(); err != nil {
				log.Printf("error: %s", errors.Wrap(err, "forcing server to close"))
			}
		}
	}
}

// initialize the server and all dependencies and return a teardown function.
func initialize(cfg config) (*http.Server, func()) {
	// Initialize dependencies.
	db, err := sqlx.Connect("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&timezone=utc",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Name, cfg.dbSSLMode()))
	if err != nil {
		log.Fatal(errors.Wrap(err, "connecting to db"))
	}
	teardown := func() {
		db.Close()
	}

	productsHandler := handlers.Products{DB: db}

	svr := http.Server{
		Addr:    cfg.HTTP.Address,
		Handler: http.HandlerFunc(productsHandler.List),
		// TODO: Timeouts in later section?
	}

	log.Println("initialized")

	return &svr, teardown
}
