package main

import (
	"log"
	"net/http"

	"github.com/unified-fleet/cloud-infra/internal/certificates"
	"github.com/unified-fleet/cloud-infra/internal/config"
	"github.com/unified-fleet/cloud-infra/internal/server"
	"github.com/unified-fleet/cloud-infra/internal/storage"
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
