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
	"github.com/kelseyhightower/envconfig"
	"github.com/nstogner/tbdsvc/5-configuration/cmd/productsd/internal/handlers"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// This is the application name.
const name = "products"

type config struct {
	// NOTE: We don't pass in a connection string b/c our application may assume
	//       certain parameters are set.
	DB struct {
		User     string `default:"postgres"`
		Password string `default:"postgres"`
		Host     string `default:"localhost"`
		Name     string `default:"postgres"`

		// TLS
		DisableTLS bool `default:"false" envconfig:"disable_tls"`
		// TODO: TLS configuration.
	}

	HTTP struct {
		Address string `default:":7070"`
	}
}

func (c *config) validate() error {
	if !c.DB.DisableTLS {
		return errors.New("enabling tls for database connection is not yet supported")
	}

	return nil
}

// dbSSLMode is derived from setting DB.DisableTLS to true/false.
func (c *config) dbSSLMode() string {
	if c.DB.DisableTLS {
		return "disable"
	}
	return "required"
}

func main() {
	var cfg config
	if err := envconfig.Process(name, &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "parsing config"))
	}

	// Initialize dependencies.
	db, err := sqlx.Connect("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&timezone=utc",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Name, cfg.dbSSLMode()))
	if err != nil {
		log.Fatal(errors.Wrap(err, "connecting to db"))
	}
	defer db.Close()

	productsHandler := handlers.Products{DB: db}

	server := http.Server{
		Addr:    cfg.HTTP.Address,
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
