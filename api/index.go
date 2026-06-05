package handler

import (
	"net/http"

	sqllens "sql-lens"
)

// Handler is the Vercel serverless function entry point.
// All /api/* requests are rewritten to this handler via vercel.json.
// The original URL path is preserved, so the Gin router can route correctly.
func Handler(w http.ResponseWriter, r *http.Request) {
	sqllens.ServeHTTP(w, r)
}
