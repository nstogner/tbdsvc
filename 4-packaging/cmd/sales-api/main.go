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
	"github.com/nstogner/tbdsvc/4-packaging/cmd/sales-api/internal/handlers"
	"github.com/pkg/errors"
)

func main() {
	// Initialize dependencies.
	user, pass, host, name := "postgres", "postgres", "localhost", "postgres"
	db, err := sqlx.Connect("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&timezone=utc",
		user, pass, host, name))
	if err != nil {
		log.Fatal(errors.Wrap(err, "connecting to db"))
	}
	defer db.Close()

	productsHandler := handlers.Products{DB: db}

	server := http.Server{
		Addr:    ":7070",
		Handler: http.HandlerFunc(productsHandler.List),
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	log.Print("startup complete")

	select {
	case err := <-serverErrors:
		log.Fatal(errors.Wrap(err, "listening and serving"))
	case <-osSignals:
		log.Print("caught signal, shutting down")

		// Give outstanding requests 30 seconds to complete.
		const timeout = 30 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("error: %s", errors.Wrap(err, "shutting down server"))
			if err := server.Close(); err != nil {
				log.Printf("error: %s", errors.Wrap(err, "forcing server to close"))
			}
		}
	}

	log.Print("done")
}
