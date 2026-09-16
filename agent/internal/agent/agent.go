package agent

import (
	"context"
	"encoding/json"
	"time"

	"github.com/soul-room/edge-agent/internal/config"
	"github.com/soul-room/edge-agent/internal/heartbeat"
	"github.com/soul-room/edge-agent/internal/identity"
	"github.com/soul-room/edge-agent/internal/inventory"
	"github.com/soul-room/edge-agent/internal/jobs"
	devicelocation "github.com/soul-room/edge-agent/internal/location"
	"github.com/soul-room/edge-agent/internal/observability"
	"github.com/soul-room/edge-agent/internal/ota"
	"github.com/soul-room/edge-agent/internal/protocol"
	"github.com/soul-room/edge-agent/internal/storage"
	"github.com/soul-room/edge-agent/internal/telemetry"
	"github.com/soul-room/edge-agent/internal/transport"
)

const Version = "0.1.0"

type Agent struct {
	Config    config.Config
	Identity  identity.Identity
	Store     *storage.Store
	Transport *transport.Client
	Jobs      *jobs.Registry
	OTA       *ota.Engine
	location  *protocol.Location
}

func New(cfg config.Config) (*Agent, error) {
	id, err := identity.LoadIdentity(cfg.Identity.StateDir)
	if err != nil {
		return nil, err
	}
	store, err := storage.Open(cfg.Storage.StateDir, cfg.Storage.MaxQueueBytes, cfg.Storage.MaxQueueAge)
	if err != nil {
		return nil, err
	}
	tr, err := transport.New(cfg, id)
	if err != nil {
		return nil, err
	}
	a := &Agent{Config: cfg, Identity: id, Store: store, Transport: tr}
	if cfg.OTA.Enabled {
		a.OTA, err = ota.NewEngine(ota.Config{
			Enabled: cfg.OTA.Enabled, Product: cfg.OTA.Product, StateDir: cfg.OTA.StateDir,
			StagingDir: cfg.OTA.StagingDir, TrustedKeysDir: cfg.OTA.TrustedKeysDir,
			PluginDir: cfg.OTA.PluginDir, MaxArtifactBytes: cfg.OTA.MaxArtifactBytes,
			MinFreeBytes: cfg.OTA.MinFreeBytes, DownloadTimeout: cfg.OTA.DownloadTimeout,
			HealthTimeout: cfg.OTA.HealthTimeout, HealthCheckCommand: cfg.OTA.HealthCheckCommand,
			AutoReboot: cfg.OTA.AutoReboot,
		}, nil)
		if err != nil {
			return nil, err
		}
	}
	a.Jobs = newJobRegistry(a)
	return a, nil
}

func (a *Agent) Run(ctx context.Context, once bool) error {
	if err := a.tick(ctx); err != nil {
		return err
	}
	if once {
		return nil
	}
	timer := time.NewTicker(a.Config.Telemetry.Interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if err := a.tick(ctx); err != nil {
				observability.Error("agent tick failed", "error", err)
			}
		}
	}
}

func (a *Agent) tick(ctx context.Context) error {
	if err := a.flush(ctx); err != nil {
		observability.Error("queue flush failed", "error", err)
	}
	location := a.location
	if a.Config.Location.Source != "ip" || location == nil {
		location = devicelocation.Collect(ctx, devicelocation.Config{Source: a.Config.Location.Source, Latitude: a.Config.Location.Latitude, Longitude: a.Config.Location.Longitude, Label: a.Config.Location.Label, GPSDAddress: a.Config.Location.GPSDAddress, IPURL: a.Config.Location.IPURL})
		if a.Config.Location.Source == "ip" && location != nil {
			a.location = location
		}
	}
	hb := heartbeat.Collect(Version, location)
	if _, err := a.Transport.PostEnvelope(ctx, "/v1/heartbeat", "heartbeat.v1", hb); err != nil {
		observability.Error("heartbeat delivery failed; buffered for retry", "endpoint", a.Config.Server.Endpoint, "error", err)
		return a.enqueue("heartbeat", storage.High, hb)
	}
	batch := telemetry.CollectAll(ctx, []telemetry.Collector{telemetry.BasicCollector{}})
	if _, err := a.Transport.PostEnvelope(ctx, "/v1/telemetry", "telemetry.v1", batch); err != nil {
		_ = a.enqueue("telemetry", storage.Low, batch)
	}
	inv := inventory.CollectWithPluginDir(a.Identity, Version, a.Config.OTA.PluginDir)
	if _, err := a.Transport.PostEnvelope(ctx, "/v1/inventory", "inventory.v1", inv); err != nil {
		_ = a.enqueue("inventory", storage.Normal, inv)
	}
	if err := a.processNextJob(ctx); err != nil {
		observability.Error("job processing failed", "error", err)
	}
	return nil
}

func (a *Agent) processNextJob(ctx context.Context) error {
	var response struct {
		Job *jobs.Job `json:"job"`
	}
	if err := a.Transport.GetDeviceJSON(ctx, "/v1/jobs/next", &response); err != nil {
		return err
	}
	if response.Job == nil {
		return nil
	}
	timeout := a.Config.Jobs.DefaultTimeout
	if response.Job.TimeoutSeconds > 0 {
		timeout = time.Duration(response.Job.TimeoutSeconds) * time.Second
	}
	jobContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result := a.Jobs.Execute(jobContext, *response.Job, jobs.NoopReporter{})
	return a.Transport.PostDeviceJSON(ctx, "/v1/jobs/result", map[string]any{
		"job_id": response.Job.ID,
		"result": result,
	}, nil)
}

func (a *Agent) enqueue(kind string, priority storage.Priority, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return a.Store.Enqueue(storage.QueueItem{ID: kind + "-" + time.Now().Format("20060102150405.000000000"), Kind: kind, Priority: priority, Payload: raw})
}

func (a *Agent) flush(ctx context.Context) error {
	items, err := a.Store.Dequeue(16)
	if err != nil {
		return err
	}
	for i, it := range items {
		endpoint := map[string]string{
			"heartbeat": "/v1/heartbeat",
			"telemetry": "/v1/telemetry",
			"inventory": "/v1/inventory",
		}[it.Kind]
		if endpoint == "" {
			endpoint = "/v1/offline"
		}
		if _, err := a.Transport.PostEnvelope(ctx, endpoint, it.Kind+".v1", json.RawMessage(it.Payload)); err != nil {
			for _, unsent := range items[i:] {
				_ = a.Store.Enqueue(unsent)
			}
			return err
		}
	}
	return nil
}
