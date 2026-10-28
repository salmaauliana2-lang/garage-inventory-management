package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"motorcycle-garage-inventory/internal/config"
	"motorcycle-garage-inventory/internal/database"
	"motorcycle-garage-inventory/internal/handlers"
	"motorcycle-garage-inventory/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Hasil build React (frontend/dist) di-embed ke binary, jadi satu file binary
// sudah menyajikan API + halaman web. Jalankan `npm run build` di frontend
// sebelum `go build`/`go run` agar isi terbaru ikut ter-embed.
//
//go:embed all:frontend/dist
var distFS embed.FS

// serveFrontend menyajikan file statis frontend/dist; path yang bukan file
// (rute React seperti /parts) dijawab dengan index.html (SPA fallback),
// sedangkan path di bawah /api/ yang tidak dikenal tetap balas JSON 404.
func serveFrontend(r *gin.Engine) {
	sub, err := fs.Sub(distFS, "frontend/dist")
	if err != nil {
		log.Fatalf("frontend dist: %v", err)
	}
	files := http.FileServer(http.FS(sub))
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "endpoint tidak ditemukan"})
			return
		}
		clean := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
		if clean != "" {
			if _, err := fs.Stat(sub, clean); err == nil {
				if strings.HasPrefix(clean, "assets/") {
					// Nama file hasil Vite sudah ber-hash — aman di-cache lama.
					c.Header("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusNotFound, "frontend belum dibuild (npm run build)")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg.DBPath)
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
	// Jangan percayai proxy/X-Forwarded-For apa pun (default Gin mempercayai semua → tidak aman).
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("set trusted proxies: %v", err)
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
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

	serveFrontend(r)

	// Prioritas port: PORT dari platform hosting, lalu cfg.Port, lalu 8080.
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}
	if port == "" {
		port = "8080"
	}
	log.Printf("API listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
