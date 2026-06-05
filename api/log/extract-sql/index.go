package handler

import (
	"net/http"

	sqllens "sql-lens"
)

// Handler is the Vercel serverless function for POST /api/log/extract-sql
func Handler(w http.ResponseWriter, r *http.Request) {
	sqllens.ServeHTTP(w, r)
}
