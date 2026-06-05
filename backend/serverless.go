package sqllens

import (
	"net/http"

	"sql-lens/internal/api"

	"github.com/gin-gonic/gin"
)

var router *gin.Engine

func init() {
	gin.SetMode(gin.ReleaseMode)
	router = api.NewRouter()
}

// ServeHTTP delegates the request to the Gin router.
// This is the entry point for Vercel serverless functions.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}
