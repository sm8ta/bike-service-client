package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/app"
	"github.com/sm8ta/webike_bike_microservice_nikita/internal/config"

	_ "github.com/lib/pq"
)

// @title Bike Microservice API
// @version 1.1
// @description API для управления байками

// @host localhost:8081
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Loading environment
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Create app
	ctx := context.Background()
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	application.Run()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	// Создаём контекст с таймаутом для shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := application.Stop(shutdownCtx); err != nil {
		log.Fatalf("Failed to stop app: %v", err)
	}
}
