package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	app := server.New(cfg, store, ca)

	go serveControl(cfg, app)
	go serveDeviceGateway(cfg, app, ca)
	go workerLoop()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Printf("platform stopping")
}

func serveControl(cfg config.Config, app *server.Server) {
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           app.ControlHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("control-api listening on http://%s", cfg.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func serveDeviceGateway(cfg config.Config, app *server.Server, ca *certificates.Authority) {
	host, _, err := net.SplitHostPort(cfg.DeviceAddr)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.DevicePublicHost != "" {
		host = cfg.DevicePublicHost
	}
	cert, err := ca.IssueServerCertificate(host, 24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}
	srv := &http.Server{
		Addr:    cfg.DeviceAddr,
		Handler: app.DeviceHandler(),
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		},
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("device-gateway listening on https://%s (certificate host %s)", cfg.DeviceAddr, host)
	if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func workerLoop() {
	for {
		time.Sleep(30 * time.Second)
		log.Printf("worker heartbeat")
	}
}
