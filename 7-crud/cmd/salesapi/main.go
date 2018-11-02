package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	"github.com/nstogner/tbdsvc/7-crud/cmd/salesapi/internal/handlers"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// This is the application name.
const name = "salesapi"

type config struct {
	// NOTE: We don't pass in a connection string b/c our application may assume
	//       certain parameters are set.
	DB struct {
		User     string `default:"postgres"`
		Password string `default:"postgres" json:"-"` // Prevent the marshalling of secrets.
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
	return "require"
}

func main() {
	cfg := configure()

	svr, teardown := initialize(cfg)
	defer teardown()

	waitAndShutdown := startup(svr)
	waitAndShutdown()
}

// configure the server by parsing environment variables and flags.
func configure() *config {
	var flags struct {
		configOnly bool
	}
	flag.Usage = func() {
		fmt.Print("This daemon is a service which manages products.\n\nUsage of salesapi:\n\nsalesapi [flags]\n\n")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		fmt.Println("\nConfiguration:\n")
		envconfig.Usage(name, &config{})
	}
	flag.BoolVar(&flags.configOnly, "config-only", false, "only show parsed configuration and exit")
	flag.Parse()

	var cfg config
	if err := envconfig.Process(name, &cfg); err != nil {
		log.Fatal(errors.Wrap(err, "parsing config"))
	}

	if flags.configOnly {
		if err := json.NewEncoder(os.Stdout).Encode(cfg); err != nil {
			log.Fatal(errors.Wrap(err, "encoding config as json"))
		}
		os.Exit(2)
	}

	log.Println("configured")

	return &cfg
}

// initialize the server and all dependencies and return a teardown function.
func initialize(cfg *config) (*http.Server, func()) {
	// Initialize dependencies.
	db, err := sqlx.Connect("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=%s&timezone=utc",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Name, cfg.dbSSLMode()))
	if err != nil {
		log.Fatal(errors.Wrap(err, "connecting to db"))
	}
	teardown := func() {
		db.Close()
	}

	productsHandler := handlers.NewProducts(db)

	svr := http.Server{
		Addr:    cfg.HTTP.Address,
		Handler: productsHandler,
		// TODO: Timeouts in later section?
	}

	log.Println("initialized")

	return &svr, teardown
}

// startup the server and return a blocking shutdown function.
func startup(svr *http.Server) func() {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- svr.ListenAndServe()
	}()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	shutdown := func() {
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

	return shutdown
}
