package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"rio-notify/internal/crypto"
	"rio-notify/internal/database"
	"rio-notify/internal/logger"
)

func main() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	log := logger.New(env)
	log.Info("🚀 Iniciando Rio Notify Service",
		"env", env,
		"version", "1.0.0",
	)

	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Error("DATABASE_URL não definida")
		os.Exit(1)
	}

	db, err := database.NewPostgresDB(ctx, databaseURL, log)
	if err != nil {
		log.Error("Falha ao conectar no PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Error("REDIS_URL não definida")
		os.Exit(1)
	}

	redis, err := database.NewRedisClient(ctx, redisURL, log)
	if err != nil {
		log.Error("Falha ao conectar no Redis", "error", err)
		os.Exit(1)
	}
	defer redis.Close()

	if err := database.RunMigrations(ctx, db); err != nil {
		log.Error("Falha ao executar migrations", "error", err)
		os.Exit(1)
	}

	pepper := os.Getenv("PEPPER")
	if pepper == "" {
		log.Error("PEPPER não definido")
		os.Exit(1)
	}

	hasher, err := crypto.NewHasher(pepper, log)
	if err != nil {
		log.Error("Falha ao inicializar hasher", "error", err)
		os.Exit(1)
	}

	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestIDMiddleware())
	router.Use(loggingMiddleware(log))
	router.Use(corsMiddleware())

	router.GET("/health", healthHandler(db, redis, log))

	api := router.Group("/api/v1")
	{
		api.GET("/notifications", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "A ser implementado",
			})
		})

		api.PATCH("/notifications/:id/read", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "A ser implementado",
			})
		})

		api.GET("/notifications/unread-count", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "A ser implementado",
			})
		})
	}

	router.POST("/webhook", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "A ser implementado",
		})
	})

	router.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "A ser implementado",
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("🌐 Servidor HTTP iniciado",
			"port", port,
			"env", env,
		)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Falha ao iniciar servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	log.Info("🛑 Sinal de desligamento recebido, iniciando graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Erro durante shutdown do servidor HTTP", "error", err)
	}

	db.Close()
	redis.Close()

	log.Info("👋 Serviço desligado com sucesso")

	_ = hasher
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func loggingMiddleware(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get("request_id")

		logFunc := log.Info
		if status >= 500 {
			logFunc = log.Error
		} else if status >= 400 {
			logFunc = log.Warn
		}

		logFunc("Requisição HTTP",
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"request_id", requestID,
			"client_ip", c.ClientIP(),
		)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func healthHandler(db *database.PostgresDB, redis *database.RedisClient, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		status := gin.H{
			"service":   "rio-notify",
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		httpStatus := http.StatusOK

		if err := db.Health(ctx); err != nil {
			status["postgres"] = "unhealthy"
			status["status"] = "degraded"
			httpStatus = http.StatusServiceUnavailable
			log.Warn("Health check: PostgreSQL indisponível", "error", err)
		} else {
			status["postgres"] = "healthy"
		}

		if err := redis.Health(ctx); err != nil {
			status["redis"] = "unhealthy"
			status["status"] = "degraded"
			httpStatus = http.StatusServiceUnavailable
			log.Warn("Health check: Redis indisponível", "error", err)
		} else {
			status["redis"] = "healthy"
		}

		c.JSON(httpStatus, status)
	}
}
