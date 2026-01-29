package main

import (
	"log"

	"cxtv-alerts/internal/database"
	"cxtv-alerts/internal/handler"
	"cxtv-alerts/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database
	db, err := database.New("data.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize service
	svc, err := service.New(db, "config/streamers.json", "config/settings.json")
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	// Start background scanner
	svc.StartScanner()

	// Start avatar updater (downloads avatars daily)
	svc.StartAvatarUpdater()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Serve static files
	r.Static("/static", "./web")
	r.StaticFile("/", "./web/index.html")

	// Register API routes
	h := handler.New(svc)
	h.RegisterRoutes(r)

	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
