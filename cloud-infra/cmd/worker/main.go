package main

import (
	"log"
	"time"

	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/storage"
)

func main() {
	cfg := config.Load()
	if _, err := storage.Open(cfg.StorePath); err != nil {
		log.Fatal(err)
	}
	log.Printf("worker started; durable PostgreSQL-backed workers are represented by repository interfaces in this MVP")
	for {
		time.Sleep(30 * time.Second)
		log.Printf("worker heartbeat")
	}
}
