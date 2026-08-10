package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/unified-fleet/edge-agent/internal/agent"
	"github.com/unified-fleet/edge-agent/internal/config"
)

func main() {
	var cfgPath string
	var once bool
	flag.StringVar(&cfgPath, "config", "/etc/edge-agent/config.yaml", "configuration path")
	flag.BoolVar(&once, "once", false, "run one collection cycle and exit")
	flag.Parse()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	a, err := agent.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := a.Run(ctx, once); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
