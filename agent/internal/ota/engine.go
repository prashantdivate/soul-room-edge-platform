package ota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type ProgressFunc func(string)

type transaction struct {
	Request         Request   `json:"request"`
	Phase           string    `json:"phase"`
	ArtifactPath    string    `json:"artifact_path,omitempty"`
	PreviousVersion string    `json:"previous_version,omitempty"`
	BootID          string    `json:"boot_id,omitempty"`
	Failure         string    `json:"failure,omitempty"`
	RebootAttempts  int       `json:"reboot_attempts,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Engine struct {
	cfg      Config
	runner   Runner
	download downloader
	now      func() time.Time
	bootID   func() string
}

func NewEngine(cfg Config, runner Runner) (*Engine, error) {
	if runner == nil {
		runner = OSRunner{}
	}
	if !cfg.Enabled {
		return nil, errors.New("OTA is disabled")
	}
	if cfg.StateDir == "" || cfg.StagingDir == "" || cfg.TrustedKeysDir == "" || cfg.PluginDir == "" {
		return nil, errors.New("OTA state, staging, trusted-key, and plugin directories are required")
	}
	for _, dir := range []string{cfg.StateDir, cfg.StagingDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	return &Engine{cfg: cfg, runner: runner, download: newDownloader(cfg), now: time.Now, bootID: currentBootID}, nil
}

func (e *Engine) Execute(ctx context.Context, request Request, progress ProgressFunc) (Status, error) {
	if progress == nil {
		progress = func(string) {}
	}
	if err := ValidateRequest(e.cfg, request); err != nil {
		return e.failed(request, "validation", err)
	}
	adapter, err := NewAdapter(request.Adapter, e.cfg.PluginDir, e.runner)
	if err != nil {
		return e.failed(request, "adapter", err)
	}
	if !adapter.Available() {
		return e.failed(request, "adapter", fmt.Errorf("adapter %s is unavailable", request.Adapter))
	}

	tx, err := e.loadOrCreate(request)
	if err != nil {
		return e.failed(request, "state", err)
	}
	if tx.Request.UpdateID != request.UpdateID || tx.Request.Adapter != request.Adapter || tx.Request.Digest != request.Digest {
		return e.failed(request, "state", errors.New("persisted OTA transaction does not match the delivered request"))
	}
	if !validPhase(tx.Phase) {
		return e.failed(request, "state", fmt.Errorf("persisted OTA transaction has unknown phase %q", tx.Phase))
	}
	if tx.Phase == "completed" {
		return status(request, "succeeded", ""), nil
	}
	if tx.Phase == "rolled_back" {
		return e.failed(request, "rolled_back", errors.New(tx.Failure))
	}
	if tx.Phase == "rollback_failed" {
		return e.failed(request, "rollback_failed", errors.New(tx.Failure))
	}

	if tx.Phase == "reboot_requested" || tx.Phase == "rollback_reboot_requested" {
		return e.resumeAfterReboot(ctx, adapter, &tx, progress)
	}

	if tx.Phase == "created" {
		progress("Running compatibility and storage preflight checks")
		previous, versionErr := adapter.CurrentVersion(ctx, request)
		if versionErr != nil {
			return e.failAndRollback(ctx, adapter, &tx, "current_version", versionErr)
		}
		tx.PreviousVersion = strings.TrimSpace(previous)
		if err := compatibleFrom(request.CompatibleFrom, tx.PreviousVersion); err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "compatibility", err)
		}
		if adapter.ArtifactRequired() {
			if err := ensureFreeSpace(e.cfg.StagingDir, e.cfg.MinFreeBytes, uint64(request.ArtifactSize)); err != nil {
				return e.failAndRollback(ctx, adapter, &tx, "preflight", err)
			}
		}
		tx.Phase = "prepared"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	}

	if tx.Phase == "prepared" && adapter.ArtifactRequired() {
		progress("Downloading the signed update artifact")
		artifact, err := e.download.fetch(ctx, request)
		if err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "download", err)
		}
		tx.ArtifactPath = artifact
		tx.Phase = "downloaded"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	} else if tx.Phase == "prepared" {
		tx.Phase = "downloaded"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	}

	if tx.Phase == "downloaded" {
		progress("Verifying digest, release signature, and native artifact metadata")
		if adapter.ArtifactRequired() {
			if err := verifyArtifact(tx.ArtifactPath, request, e.cfg.TrustedKeysDir); err != nil {
				return e.failAndRollback(ctx, adapter, &tx, "verification", err)
			}
		}
		if err := adapter.Check(ctx, request, tx.ArtifactPath); err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "native_verification", err)
		}
		tx.Phase = "verified"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	}

	if tx.Phase == "verified" {
		progress("Installing the update with the selected device adapter")
		if err := adapter.Install(ctx, request, tx.ArtifactPath); err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "install", err)
		}
		tx.Phase = "installed"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	}

	if tx.Phase == "installed" {
		progress("Activating the new device state")
		if err := adapter.Activate(ctx, request); err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "activate", err)
		}
		tx.Phase = "activated"
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
	}

	if tx.Phase == "activated" && adapter.RequiresReboot() {
		tx.Phase = "reboot_requested"
		tx.BootID = e.bootID()
		if err := e.save(tx); err != nil {
			return e.failed(request, "state", err)
		}
		progress("Requesting a reboot to boot the staged update")
		if err := e.requestReboot(ctx); err != nil {
			return e.failAndRollback(ctx, adapter, &tx, "reboot", err)
		}
		return status(request, "rebooting", ""), ErrRebootPending
	}

	return e.confirmAndCommit(ctx, adapter, &tx, progress)
}

func (e *Engine) resumeAfterReboot(ctx context.Context, adapter Adapter, tx *transaction, progress ProgressFunc) (Status, error) {
	if tx.BootID != "" && tx.BootID == e.bootID() {
		if e.now().Sub(tx.UpdatedAt) >= 2*time.Minute {
			if tx.RebootAttempts >= 3 {
				if tx.Phase == "rollback_reboot_requested" {
					tx.Phase = "rollback_failed"
					tx.Failure += "; device did not reboot after three requests"
					_ = e.save(*tx)
					return e.failed(tx.Request, "rollback_failed", errors.New(tx.Failure))
				}
				return e.failAndRollback(ctx, adapter, tx, "reboot", errors.New("device did not reboot after three requests"))
			}
			tx.RebootAttempts++
			if err := e.save(*tx); err != nil {
				return e.failed(tx.Request, "state", err)
			}
			if err := e.requestReboot(ctx); err != nil {
				return e.failed(tx.Request, "reboot", err)
			}
		}
		return status(tx.Request, "rebooting", ""), ErrRebootPending
	}
	if tx.Phase == "rollback_reboot_requested" {
		tx.Phase = "rolled_back"
		if err := e.save(*tx); err != nil {
			return e.failed(tx.Request, "state", err)
		}
		return e.failed(tx.Request, "rolled_back", errors.New(tx.Failure))
	}
	progress("The device rebooted; confirming update health")
	return e.confirmAndCommit(ctx, adapter, tx, progress)
}

func (e *Engine) confirmAndCommit(ctx context.Context, adapter Adapter, tx *transaction, progress ProgressFunc) (Status, error) {
	healthCtx, cancel := context.WithTimeout(ctx, e.cfg.HealthTimeout)
	defer cancel()
	if err := e.healthCheck(healthCtx); err != nil {
		return e.failAndRollback(ctx, adapter, tx, "health", err)
	}
	current, err := adapter.CurrentVersion(healthCtx, tx.Request)
	if err != nil {
		return e.failAndRollback(ctx, adapter, tx, "version_confirmation", err)
	}
	expected := tx.Request.Version
	if tx.Request.Adapter == "flatpak" {
		expected = tx.Request.FlatpakCommit
	}
	if expected != "" && strings.TrimSpace(current) != "" && !strings.Contains(current, expected) {
		return e.failAndRollback(ctx, adapter, tx, "version_confirmation", fmt.Errorf("reported version does not contain expected release %q", expected))
	}
	progress("Committing the healthy update")
	if err := adapter.Commit(ctx, tx.Request); err != nil {
		return e.failAndRollback(ctx, adapter, tx, "commit", err)
	}
	tx.Phase = "completed"
	if err := e.save(*tx); err != nil {
		return e.failed(tx.Request, "state", err)
	}
	if tx.ArtifactPath != "" {
		_ = os.Remove(tx.ArtifactPath)
	}
	return status(tx.Request, "succeeded", ""), nil
}

func (e *Engine) failAndRollback(ctx context.Context, adapter Adapter, tx *transaction, class string, cause error) (Status, error) {
	tx.Failure = class + ": " + cause.Error()
	if tx.Phase == "created" || tx.Phase == "prepared" || tx.Phase == "downloaded" || tx.Phase == "verified" {
		tx.Phase = "rolled_back"
		_ = e.save(*tx)
		return e.failed(tx.Request, class, cause)
	}
	if err := adapter.Rollback(ctx, tx.Request, tx.PreviousVersion); err != nil {
		tx.Phase = "rollback_failed"
		tx.Failure += "; rollback: " + err.Error()
		_ = e.save(*tx)
		return e.failed(tx.Request, "rollback_failed", errors.New(tx.Failure))
	}
	if adapter.RequiresReboot() {
		tx.Phase = "rollback_reboot_requested"
		tx.BootID = e.bootID()
		_ = e.save(*tx)
		if err := e.requestReboot(ctx); err != nil {
			tx.Phase = "rollback_failed"
			tx.Failure += "; rollback reboot: " + err.Error()
			_ = e.save(*tx)
			return e.failed(tx.Request, "rollback_failed", errors.New(tx.Failure))
		}
		return status(tx.Request, "rebooting", tx.Failure), ErrRebootPending
	}
	tx.Phase = "rolled_back"
	_ = e.save(*tx)
	return e.failed(tx.Request, class, cause)
}

func (e *Engine) requestReboot(ctx context.Context) error {
	if !e.cfg.AutoReboot {
		return errors.New("automatic reboot is disabled in the local agent configuration")
	}
	_, err := e.runner.Run(ctx, "systemctl", "reboot", "--no-block")
	return err
}

func (e *Engine) healthCheck(ctx context.Context) error {
	if strings.TrimSpace(e.cfg.HealthCheckCommand) == "" {
		return nil
	}
	parts := strings.Fields(e.cfg.HealthCheckCommand)
	if len(parts) == 0 {
		return nil
	}
	var last error
	for {
		if _, err := e.runner.Run(ctx, parts[0], parts[1:]...); err == nil {
			return nil
		} else {
			last = err
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("health check did not pass: %w", last)
		case <-timer.C:
		}
	}
}

func (e *Engine) loadOrCreate(request Request) (transaction, error) {
	path := e.transactionPath(request.UpdateID)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		tx := transaction{Request: request, Phase: "created"}
		return tx, e.save(tx)
	}
	if err != nil {
		return transaction{}, err
	}
	var tx transaction
	if err := json.Unmarshal(b, &tx); err != nil {
		return transaction{}, err
	}
	return tx, nil
}

func (e *Engine) save(tx transaction) error {
	tx.UpdatedAt = e.now().UTC()
	b, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	path := e.transactionPath(tx.Request.UpdateID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	f, err := os.OpenFile(tmp, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (e *Engine) transactionPath(updateID string) string {
	return filepath.Join(e.cfg.StateDir, safeID(updateID)+".json")
}

func safeID(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, value)
}

func (e *Engine) failed(request Request, class string, err error) (Status, error) {
	if err == nil {
		err = errors.New("unknown OTA error")
	}
	return status(request, "failed", class+": "+err.Error()), err
}

func status(request Request, state, message string) Status {
	return Status{UpdateID: request.UpdateID, Adapter: request.Adapter, State: state, Version: request.Version, Error: message}
}

func compatibleFrom(allowed []string, current string) error {
	if len(allowed) == 0 {
		return nil
	}
	for _, version := range allowed {
		if version == current {
			return nil
		}
	}
	return fmt.Errorf("current version is not in the release compatibility list")
}

func validPhase(phase string) bool {
	switch phase {
	case "created", "prepared", "downloaded", "verified", "installed", "activated", "reboot_requested", "rollback_reboot_requested", "completed", "rolled_back", "rollback_failed":
		return true
	default:
		return false
	}
}

func ensureFreeSpace(path string, minimum, artifactSize uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return err
	}
	available := stat.Bavail * uint64(stat.Bsize)
	required := minimum + artifactSize
	if available < required {
		return fmt.Errorf("insufficient staging space: need %d bytes, have %d", required, available)
	}
	return nil
}

func currentBootID() string {
	b, _ := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return strings.TrimSpace(string(b))
}
