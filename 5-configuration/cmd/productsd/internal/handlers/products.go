package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/nstogner/tbdsvc/5-configuration/internal/products"
	"github.com/pkg/errors"
)

type Products struct {
	DB *sqlx.DB
}

func (s *Products) List(w http.ResponseWriter, r *http.Request) {
	list, err := products.List(s.DB)
	if err != nil {
		log.Println(errors.Wrap(err, "listing products"))
		w.WriteHeader(500)
		return
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Println(errors.Wrap(err, "encoding response"))
		return
	}
}
