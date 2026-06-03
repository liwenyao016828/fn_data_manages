package main

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/example/db-manager/core/app/model"
	"github.com/example/db-manager/core/app/repo"
	"github.com/example/db-manager/core/router"
)

var (
	logger *logrus.Logger
)

func init() {
	logger = logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	dataDir := os.Getenv("TRIM_PKGVAR")
	if dataDir == "" {
		dataDir = "./data"
	}

	err := os.MkdirAll(dataDir, 0755)
	if err != nil {
		logger.Fatalf("Failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "database.sqlite")
	repo.DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.Fatalf("Failed to connect database: %v", err)
	}

	err = repo.DB.AutoMigrate(
		&model.Database{},
		&model.DatabaseMysql{},
		&model.DatabasePostgresql{},
	)
	if err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	r := gin.Default()
	r.Use(CORSMiddleware())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.RegisterRoutes(r)

	distPath := filepath.Join(".", "frontend", "dist")
	r.NoRoute(func(c *gin.Context) {
		path := filepath.Join(distPath, filepath.Clean(c.Request.URL.Path))
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			c.File(path)
			return
		}
		c.File(filepath.Join(distPath, "index.html"))
	})

	port := os.Getenv("TRIM_SERVICE_PORT")
	if port == "" {
		port = "8080"
	}

	logger.Infof("Server starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}