package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	// "github.com/Puneet-Vishnoi/order-matching-engine/cache/redis"
	// redisProvider "github.com/Puneet-Vishnoi/order-matching-engine/cache/redis/providers"
	"github.com/Puneet-Vishnoi/order-matching-engine/db/postgres"
	providers "github.com/Puneet-Vishnoi/order-matching-engine/db/postgres/providers"
	"github.com/Puneet-Vishnoi/order-matching-engine/repository"
	"github.com/Puneet-Vishnoi/order-matching-engine/routes"
	orderService "github.com/Puneet-Vishnoi/order-matching-engine/service"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Failed to load env file: ", err)
	}

	// // 1. Connect Redis
	// redisClient := redis.ConnectRedis()
	// redisHelper := redisProvider.NewRedisProvider(redisClient.RedisClient)
	// defer redisClient.Stop()

	// 2. Connect PostgreSQL
	postgresClient := postgres.ConnectDB()
	defer postgresClient.Stop()

	// 2.1 Init Schema (optional — only for development)
	if err := postgresClient.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	// 3. DB Helper
	dbHelper, err := providers.NewDbProvider(postgresClient.PostgresClient)
	if err != nil {
		log.Fatalf("Failed to initialize DB helper: %v", err)
	}

	// 4. Repo & Service
	orderRepo := repository.NewOrderRepository(dbHelper)
	tradeRepo := repository.NewTradeRepository(dbHelper)
	orderSrv := orderService.NewOrderService(orderRepo, tradeRepo)

	// 5. Gin Router & Handlers
	router := gin.Default()
	routes.RegisterRoutes(router, orderSrv)

	// 6. Run REST API
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	//7. run server in GO rutine so main thread become non blocking
	go func() {
		fmt.Printf("Order REST API running on %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start Gin server: %v", err)
		}
	}()

	// 8. wait for OS Signal to shutdown gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %s. Hence Gracefully Shutdown.", sig)

	//9.1 gracefully shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("gracefully shutdown")
}
