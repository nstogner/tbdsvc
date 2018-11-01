package main

import (
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
)

func TestConfig(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string

		expectedConfig func() *config

		// Derived fields.
		expectedDBTLSMode string

		expectedValidateErr error
	}{
		{
			name: "happy",
			env: map[string]string{
				"PRODUCTS_DB_HOST":        "my-db-host",
				"PRODUCTS_DB_USER":        "my-db-user",
				"PRODUCTS_DB_NAME":        "my-db-name",
				"PRODUCTS_DB_PASSWORD":    "my-db-password",
				"PRODUCTS_DB_DISABLE_TLS": "true",
				"PRODUCTS_HTTP_ADDRESS":   ":9090",
			},
			expectedConfig: func() *config {
				var c config
				c.DB.Host = "my-db-host"
				c.DB.User = "my-db-user"
				c.DB.Name = "my-db-name"
				c.DB.Password = "my-db-password"
				c.DB.DisableTLS = true
				c.HTTP.Address = ":9090"
				return &c
			},
			expectedDBTLSMode:   "disable",
			expectedValidateErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for k, v := range c.env {
				os.Setenv(k, v)
			}

			var parsed config
			if err := envconfig.Process(name, &parsed); err != nil {
				t.Errorf("parsing: %s", err)
			}
			if exp, got := *c.expectedConfig(), parsed; exp != got {
				t.Errorf("expected config %+v, got %+v", exp, got)
			}

			if exp, got := c.expectedValidateErr, c.expectedConfig().validate(); exp != got {
				t.Errorf("expected validation error %q, got %q", exp, got)
			}
		})
	}
}
