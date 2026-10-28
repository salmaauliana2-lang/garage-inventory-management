package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"motorcycle-garage-inventory/internal/config"
	"motorcycle-garage-inventory/internal/database"
	"motorcycle-garage-inventory/internal/handlers"
	"motorcycle-garage-inventory/internal/middleware"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.DatabaseURL, cfg.DBPath)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	if cfg.SeedDemo {
		if err := database.SeedDemo(db); err != nil {
			log.Printf("demo seed skipped: %v", err)
		}
	}

	if cfg.GinMode != "" {
		gin.SetMode(cfg.GinMode)
	}
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false, MaxAge: 12 * time.Hour,
	}))
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	h := handlers.New(db, cfg)
	api := r.Group("/api/v1")
	api.POST("/auth/register", h.Register)
	api.POST("/auth/login", h.Login)
	protected := api.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	protected.POST("/auth/logout", h.Logout) // JWT is stateless; client discards token.
	protected.GET("/auth/me", h.Me)
	protected.GET("/parts", h.ListParts)
	protected.POST("/parts", middleware.RequireRole("manager"), h.CreatePart)
	protected.GET("/parts/:id", h.GetPart)
	protected.PUT("/parts/:id", middleware.RequireRole("manager"), h.UpdatePart)
	protected.DELETE("/parts/:id", middleware.RequireRole("manager"), h.DeletePart)
	protected.POST("/parts/:id/restock", middleware.RequireRole("manager"), h.RestockPart)
	protected.GET("/stock-movements", h.ListStockMovements)
	protected.GET("/sales", h.ListSales)
	protected.POST("/sales", h.CreateSale)
	protected.GET("/services", h.ListServices)
	protected.POST("/services", h.CreateService)
	protected.PATCH("/services/:id/status", h.UpdateServiceStatus)
	protected.GET("/dashboard", h.Dashboard)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("API listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
