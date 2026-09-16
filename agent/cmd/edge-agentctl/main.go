package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/soul-room/edge-agent/internal/agent"
	"github.com/soul-room/edge-agent/internal/config"
	"github.com/soul-room/edge-agent/internal/configtui"
	"github.com/soul-room/edge-agent/internal/diagnostics"
	"github.com/soul-room/edge-agent/internal/enrollment"
	"github.com/soul-room/edge-agent/internal/identity"
	"github.com/soul-room/edge-agent/internal/storage"
	"github.com/soul-room/edge-agent/internal/telemetry"
	"github.com/soul-room/edge-agent/internal/transport"
)

func main() {
	var cfgPath string
	var jsonOut bool
	flag.StringVar(&cfgPath, "config", "/etc/edge-agent/config.yaml", "configuration path")
	flag.BoolVar(&jsonOut, "json", false, "write JSON output")
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}
	cmd := flag.Arg(0)
	cfg, err := config.Load(cfgPath)
	if err != nil && cmd != "version" {
		log.Fatal(err)
	}
	switch cmd {
	case "version":
		printOut(jsonOut, map[string]string{"version": agent.Version})
	case "config":
		if flag.Arg(1) != "validate" {
			usage()
		}
		printOut(jsonOut, map[string]string{"status": "valid"})
	case "tui":
		must(configtui.Run(cfgPath, cfg, os.Stdin, os.Stdout))
	case "enroll":
		enrollFlags := flag.NewFlagSet("enroll", flag.ExitOnError)
		token := enrollFlags.String("token", "", "one-time enrollment token")
		name := enrollFlags.String("name", "", "device display name")
		must(enrollFlags.Parse(flag.Args()[1:]))
		if *token == "" {
			log.Fatal("-token is required")
		}
		c, err := enrollment.NewClient(cfg)
		must(err)
		ctx, cancel := enrollment.ContextWithDefaultTimeout(context.Background())
		defer cancel()
		id, err := c.Enroll(ctx, *token, *name)
		must(err)
		printOut(jsonOut, id)
	case "identity":
		if flag.Arg(1) != "show" {
			usage()
		}
		id, err := identity.LoadIdentity(cfg.Identity.StateDir)
		must(err)
		id.CertificatePEM = "<redacted>"
		id.CAPEM = "<redacted>"
		printOut(jsonOut, id)
	case "status":
		id, identityErr := identity.LoadIdentity(cfg.Identity.StateDir)
		st, err := storage.Open(cfg.Storage.StateDir, cfg.Storage.MaxQueueBytes, cfg.Storage.MaxQueueAge)
		must(err)
		stats, err := st.Stats()
		must(err)
		connection := map[string]any{"status": "not_enrolled"}
		if identityErr == nil {
			connection["status"] = "unreachable"
			client, err := transport.New(cfg, id)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ConnectTimeout)
				err = client.Status(ctx)
				cancel()
			}
			if err == nil {
				connection["status"] = "connected"
			} else {
				connection["error"] = err.Error()
			}
		} else {
			connection["error"] = identityErr.Error()
		}
		printOut(jsonOut, map[string]any{"version": agent.Version, "device_id": id.DeviceID, "tenant_id": id.TenantID, "endpoint": cfg.Server.Endpoint, "connection": connection, "queue": stats})
	case "telemetry":
		if flag.Arg(1) != "collect" {
			usage()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		printOut(jsonOut, telemetry.CollectAll(ctx, []telemetry.Collector{telemetry.BasicCollector{}}))
	case "queue":
		if flag.Arg(1) != "status" && flag.Arg(1) != "archive" {
			usage()
		}
		st, err := storage.Open(cfg.Storage.StateDir, cfg.Storage.MaxQueueBytes, cfg.Storage.MaxQueueAge)
		must(err)
		if flag.Arg(1) == "archive" {
			path, err := st.ArchiveQueue()
			must(err)
			printOut(jsonOut, map[string]string{"archived_queue": path})
		} else {
			stats, err := st.Stats()
			must(err)
			printOut(jsonOut, stats)
		}
	case "jobs":
		if flag.Arg(1) != "list" {
			usage()
		}
		printOut(jsonOut, map[string]any{"jobs": []any{}})
	case "connectors":
		if flag.Arg(1) != "list" {
			usage()
		}
		printOut(jsonOut, map[string]any{"connectors": []string{"simulator", "modbus-tcp"}})
	case "diagnostics":
		if flag.Arg(1) != "create" {
			usage()
		}
		st, err := storage.Open(cfg.Storage.StateDir, cfg.Storage.MaxQueueBytes, cfg.Storage.MaxQueueAge)
		must(err)
		stats, _ := st.Stats()
		path := filepath.Join(cfg.Storage.StateDir, "diagnostics", "bundle-"+time.Now().Format("20060102150405")+".zip")
		must(diagnostics.Create(path, agent.Version, cfg, stats))
		printOut(jsonOut, map[string]string{"path": path})
	default:
		usage()
	}
}

func printOut(jsonOut bool, v any) {
	if jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(v)
		return
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println("edge-agentctl [-config PATH] tui | enroll -token TOKEN [-name NAME] | status | identity show | config validate | telemetry collect | queue status|archive | diagnostics create | version")
	os.Exit(2)
}
