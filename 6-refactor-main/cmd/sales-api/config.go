package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

// This is the application name.
const name = "products"

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

func (c config) validate() error {
	if !c.DB.DisableTLS {
		return errors.New("enabling tls for database connection is not yet supported")
	}

	return nil
}

// dbSSLMode is derived from setting DB.DisableTLS to true/false.
func (c config) dbSSLMode() string {
	if c.DB.DisableTLS {
		return "disable"
	}
	return "require"
}

// configure the server by parsing environment variables and flags.
func configure() config {
	var flags struct {
		configOnly bool
	}
	flag.Usage = func() {
		fmt.Print("This daemon is a service which manages products.\n\nUsage of sales-api:\n\nsales-api [flags]\n\n")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		fmt.Print("\nConfiguration:\n\n")
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

	return cfg
}
