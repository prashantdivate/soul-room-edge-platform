package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"time"

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
	host, _, err := net.SplitHostPort(cfg.DeviceAddr)
	if err != nil {
		log.Fatal(err)
	}
	cert, err := ca.IssueServerCertificate(host, 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:    cfg.DeviceAddr,
		Handler: server.New(cfg, store, ca).DeviceHandler(),
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		},
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("device-gateway listening on https://%s", cfg.DeviceAddr)
	log.Fatal(srv.ListenAndServeTLS("", ""))
}
