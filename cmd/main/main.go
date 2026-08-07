package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ferrarodomenico/gimli/internal/config"
	"github.com/ferrarodomenico/gimli/internal/queue"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using system environment variables: %v", err)
	}

	cfg := config.LoadConfig()
	log.Printf("Loaded configuration: %+v", cfg)

	consumer, err := queue.NewConsumer(cfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	if err := consumer.Setup(); err != nil {
		log.Fatalf("Failed to setup queues: %v", err)
	}

	if err := consumer.Start(cfg.WorkerCount); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	log.Printf("Gimli worker started, listening on queue: %s", cfg.MainQueue.Name)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}
