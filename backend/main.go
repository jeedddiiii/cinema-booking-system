package main

import (
	"context"
	"log"
	"net/http"

	"cinema-booking-system/config"
	"cinema-booking-system/handlers"
	"cinema-booking-system/services"
	"cinema-booking-system/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("🎬 Starting Cinema Booking System on port %s", cfg.Port)

	if _, err := config.InitMongoDB(cfg.MongoURI); err != nil {
		log.Printf("⚠️ MongoDB connection failed: %v", err)
		log.Println("Continuing without MongoDB...")
	}

	config.InitRedis(cfg.RedisHost, cfg.RedisPort)
	config.InitKafka(cfg.KafkaBroker, cfg.KafkaTopic)
	defer config.CloseConnections()

	wsHub := websocket.NewHub()
	go wsHub.Run()

	lockMonitor := services.NewLockExpiryMonitor(wsHub)
	go lockMonitor.Start(context.Background())

	kafkaConsumer := services.NewKafkaConsumerService(cfg.KafkaBroker, cfg.KafkaTopic, "audit-log-consumer")
	go kafkaConsumer.Start(context.Background())

	h := handlers.NewHandler(wsHub)
	adminHandler := handlers.NewAdminHandler()

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.GET("/health", h.HealthCheck)

	api := router.Group("/api")
	{
		api.GET("/sessions", h.GetSessions)
		api.GET("/sessions/:id", h.GetSession)
		api.POST("/sessions/demo", h.CreateDemoSession)

		api.POST("/seats/lock", h.LockSeats)
		api.POST("/seats/unlock", h.UnlockSeats)

		api.POST("/bookings", h.CreateBooking)

		authHandler := handlers.NewAuthHandler()
		api.POST("/auth/login", authHandler.Login)
		api.GET("/auth/role", authHandler.GetUserRole)
		api.POST("/auth/role", authHandler.SetUserRole)
	}

	admin := router.Group("/api/admin")
	{
		admin.GET("/bookings", adminHandler.GetBookings)
		admin.GET("/bookings/stats", adminHandler.GetBookingStats)
		admin.GET("/audit-logs", adminHandler.GetAuditLogs)
	}

	router.GET("/ws", func(c *gin.Context) {
		websocket.ServeWs(wsHub, c.Writer, c.Request)
	})

	log.Printf("Server ready at http://localhost:%s", cfg.Port)
	log.Printf("WebSocket endpoint: ws://localhost:%s/ws", cfg.Port)
	log.Printf("Admin dashboard: http://localhost:%s/api/admin/bookings", cfg.Port)

	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
