package main

import (
	"log"
	"net/http"

	"github.com/soul-room/cloud-infra/internal/certificates"
	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/server"
	"github.com/soul-room/cloud-infra/internal/storage"
)

func main() {
	cfg := config.Load()
	store, err := storage.Open(cfg.StorePath)
	if err != nil {
		log.Fatal(err)
	}
	ca, err := certificates.LoadOrCreateDevCA(cfg.DevCADir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("control-api listening on %s", cfg.HTTPAddr)
	log.Fatal(http.ListenAndServe(cfg.HTTPAddr, server.New(cfg, store, ca).ControlHandler()))
}
