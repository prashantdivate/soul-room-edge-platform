package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/soul-room/edge-agent/internal/jobs"
	"github.com/soul-room/edge-agent/internal/storage"
)

func TestRegistrySupportsAssistantDeviceJobs(t *testing.T) {
	store, err := storage.Open(t.TempDir(), 1024*1024, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registry := newJobRegistry(&Agent{Store: store})
	for _, jobType := range []string{"collect_diagnostics", "collect_logs", "collect_inventory", "scan_vulnerabilities", "reboot"} {
		if !registry.Supports(jobType) {
			t.Fatalf("assistant job %q has no agent handler", jobType)
		}
	}
}

func TestRebootUsesDelayedSystemdUnit(t *testing.T) {
	lookups := map[string]string{"systemd-run": "/usr/bin/systemd-run", "systemctl": "/usr/bin/systemctl"}
	lookup := func(name string) (string, error) {
		if path := lookups[name]; path != "" {
			return path, nil
		}
		return "", errors.New("not found")
	}
	var command string
	var args []string
	result := runRebootWith(context.Background(), jobs.Job{ID: "job-1"}, jobs.NoopReporter{}, lookup, func(_ context.Context, name string, values ...string) ([]byte, error) {
		command, args = name, values
		return []byte("scheduled"), nil
	})
	if result.State != "succeeded" || command != "/usr/bin/systemd-run" {
		t.Fatalf("unexpected reboot result: %+v command=%q", result, command)
	}
	if len(args) != 4 || args[1] != "--on-active=10s" || !reflect.DeepEqual(args[2:], []string{"/usr/bin/systemctl", "reboot"}) {
		t.Fatalf("unexpected reboot arguments: %#v", args)
	}
}

func TestRebootFallsBackToShutdown(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "shutdown" {
			return "/sbin/shutdown", nil
		}
		return "", errors.New("not found")
	}
	command, args, _, err := rebootCommand(lookup, "job-2")
	if err != nil || command != "/sbin/shutdown" || !reflect.DeepEqual(args, []string{"-r", "+1", "Soul Room approved reboot"}) {
		t.Fatalf("unexpected fallback: command=%q args=%#v err=%v", command, args, err)
	}
}
