package main

import (
	"log"

	"sql-lens/internal/api"
)

func main() {
	r := api.NewRouter()

	log.Println("SQL Lens server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
