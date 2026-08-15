package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"time"

	"github.com/soul-room/edge-agent/internal/applications"
	"github.com/soul-room/edge-agent/internal/inventory"
	"github.com/soul-room/edge-agent/internal/jobs"
	"github.com/soul-room/edge-agent/internal/ota"
	"github.com/soul-room/edge-agent/internal/securityscan"
)

func newJobRegistry(a *Agent) *jobs.Registry {
	registry := jobs.NewRegistry(a.Store)
	registry.Register("collect_logs", jobs.NewFunctionHandler(nil, collectLogs))
	registry.Register("collect_inventory", jobs.NewFunctionHandler(nil, func(_ context.Context, job jobs.Job, _ jobs.ProgressReporter) jobs.Result {
		return successfulJob(job.ID, inventory.CollectWithPluginDir(a.Identity, Version, a.Config.OTA.PluginDir))
	}))
	registry.Register("collect_diagnostics", jobs.NewFunctionHandler(nil, func(_ context.Context, job jobs.Job, _ jobs.ProgressReporter) jobs.Result {
		queue, _ := a.Store.Stats()
		return successfulJob(job.ID, map[string]any{
			"agent_version": Version,
			"inventory":     inventory.CollectWithPluginDir(a.Identity, Version, a.Config.OTA.PluginDir),
			"offline_queue": queue,
		})
	}))
	registry.Register("scan_vulnerabilities", jobs.NewFunctionHandler(nil, func(ctx context.Context, job jobs.Job, reporter jobs.ProgressReporter) jobs.Result {
		started := time.Now()
		_ = reporter.Progress(ctx, job.ID, "Scanning operating-system packages with Trivy")
		report, err := securityscan.Scan(ctx)
		if err != nil {
			return jobs.Result{JobID: job.ID, State: "failed", ExitCode: 1, Stderr: err.Error(), ErrorClass: "vulnerability_scan", StartedAt: started, FinishedAt: time.Now()}
		}
		return successfulJob(job.ID, report)
	}))
	registry.Register("flatpak_update", jobs.NewFunctionHandler(func(_ context.Context, job jobs.Job) error {
		_, err := applications.ParseFlatpakUpdate(job.Payload)
		return err
	}, runFlatpakUpdate))
	registry.Register("ota_update", jobs.NewFunctionHandler(func(_ context.Context, job jobs.Job) error {
		if a.OTA == nil {
			return errors.New("OTA is disabled on this device")
		}
		var request ota.Request
		if err := json.Unmarshal(job.Payload, &request); err != nil {
			return errors.New("invalid OTA update payload")
		}
		return ota.ValidateRequest(ota.Config{
			Enabled: a.Config.OTA.Enabled, Product: a.Config.OTA.Product,
			MaxArtifactBytes: a.Config.OTA.MaxArtifactBytes,
		}, request)
	}, func(ctx context.Context, job jobs.Job, reporter jobs.ProgressReporter) jobs.Result {
		started := time.Now()
		var request ota.Request
		if err := json.Unmarshal(job.Payload, &request); err != nil {
			return jobs.Result{JobID: job.ID, State: "failed", ExitCode: 1, Stderr: err.Error(), ErrorClass: "validation", StartedAt: started, FinishedAt: time.Now()}
		}
		updateStatus, err := a.OTA.Execute(ctx, request, func(message string) { _ = reporter.Progress(ctx, job.ID, message) })
		result := jobs.Result{JobID: job.ID, State: updateStatus.State, StartedAt: started, FinishedAt: time.Now()}
		encoded, _ := json.Marshal(updateStatus)
		result.Stdout = string(encoded)
		if err != nil && !errors.Is(err, ota.ErrRebootPending) {
			result.State = "failed"
			result.ExitCode = 1
			result.Stderr = err.Error()
			result.ErrorClass = "ota_update"
		}
		return result
	}))
	return registry
}

func runFlatpakUpdate(ctx context.Context, job jobs.Job, reporter jobs.ProgressReporter) jobs.Result {
	started := time.Now()
	update, err := applications.ParseFlatpakUpdate(job.Payload)
	if err != nil {
		return jobs.Result{JobID: job.ID, State: "failed", ExitCode: 1, Stderr: err.Error(), ErrorClass: "validation", StartedAt: started, FinishedAt: time.Now()}
	}
	_ = reporter.Progress(ctx, job.ID, "Validating Flatpak remote and application reference")
	output, err := applications.RunFlatpakUpdate(ctx, update)
	result := jobs.Result{JobID: job.ID, State: "succeeded", ExitCode: 0, Stdout: output, StartedAt: started, FinishedAt: time.Now()}
	if err != nil {
		result.State = "failed"
		result.ExitCode = 1
		result.Stderr = err.Error()
		result.ErrorClass = "flatpak_update"
	}
	return result
}

func collectLogs(ctx context.Context, job jobs.Job, _ jobs.ProgressReporter) jobs.Result {
	started := time.Now()
	output, err := exec.CommandContext(ctx, "journalctl", "-n", "200", "--no-pager", "-o", "short-iso").CombinedOutput()
	result := jobs.Result{JobID: job.ID, State: "succeeded", ExitCode: 0, Stdout: string(output), StartedAt: started, FinishedAt: time.Now()}
	if err != nil {
		result.State = "failed"
		result.ExitCode = 1
		result.Stderr = err.Error()
		result.ErrorClass = "log_collection"
	}
	return result
}

func successfulJob(jobID string, value any) jobs.Result {
	started := time.Now()
	output, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return jobs.Result{JobID: jobID, State: "failed", ExitCode: 1, Stderr: err.Error(), ErrorClass: "encoding", StartedAt: started, FinishedAt: time.Now()}
	}
	return jobs.Result{JobID: jobID, State: "succeeded", ExitCode: 0, Stdout: string(output), StartedAt: started, FinishedAt: time.Now()}
}
