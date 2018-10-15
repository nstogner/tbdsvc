package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	if err := http.ListenAndServe(":7070", http.HandlerFunc(ListProducts)); err != nil {
		log.Fatal(err)
	}
}

type Product struct {
	Name     string
	Cost     int
	Quantity int
}

func ListProducts(w http.ResponseWriter, r *http.Request) {
	products := []Product{
		{Name: "Comic Books", Cost: 50, Quantity: 42},
		{Name: "McDonalds Toys", Cost: 75, Quantity: 120},
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		log.Println("encoding response:", err)
	}
}
