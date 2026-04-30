package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	v1 := r.Group("/api")
	{
		v1.POST("/sql/parse", ParseSQLHandler)
		v1.POST("/sql/format", FormatSQLHandler)
		v1.POST("/log/extract-sql", ExtractSQLHandler)
		v1.POST("/report/markdown", BuildMarkdownReportHandler)
	}

	return r
}
