package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sagittarius/auction/services/api-gateway/internal/middleware"
	"golang.org/x/time/rate"
)

func main() {
	httpPort := getEnv("HTTP_PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "your-secret-key")

	router := gin.Default()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Rate limiting
	limiter := middleware.NewRateLimiter(100, time.Second)
	router.Use(middleware.RateLimitMiddleware(limiter))

	// Auth middleware (optional for development)
	router.Use(middleware.JWTAuthMiddleware(jwtSecret))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes - proxy to services
	api := router.Group("/api/v1")
	{
		// Proxy to Bidder Service
		api.Any("/bids/*path", proxyToService(getEnv("BIDDER_SERVICE_URL", "http://localhost:8081")))
		api.Any("/auctions/*path", proxyToService(getEnv("AUCTION_SERVICE_URL", "http://localhost:8082")))
	}

	// Metrics
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	go func() {
		log.Printf("API Gateway listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API Gateway...")
	// Graceful shutdown would go here
	log.Println("API Gateway stopped")
}

func proxyToService(baseURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Simple proxy implementation
		// In production, use a proper reverse proxy library
		c.JSON(http.StatusNotImplemented, gin.H{"error": "proxy not implemented yet"})
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

