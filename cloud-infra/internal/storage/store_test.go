package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/unified-fleet/cloud-infra/internal/model"
	"github.com/unified-fleet/cloud-infra/internal/tenancy"
)

func TestTenantIsolationDevicesTelemetryJobsAudit(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	orgA, _ := store.CreateOrganization("Tenant A", "a")
	orgB, _ := store.CreateOrganization("Tenant B", "b")
	tcA := tenancy.Context{TenantID: orgA.ID, ActorID: "u1"}
	tcB := tenancy.Context{TenantID: orgB.ID, ActorID: "u2"}
	devA, err := store.CreateDevice(model.Device{TenantID: orgA.ID, DisplayName: "a", EnrollmentStatus: "enrolled"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetDevice(tcB, devA.ID); err == nil {
		t.Fatal("tenant B accessed tenant A device")
	}
	if err := store.AddTelemetry(tcA, []model.Metric{{DeviceID: devA.ID, Name: "cpu", Value: 1, DeviceTime: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	if got := store.ListTelemetry(tcB, devA.ID); len(got) != 0 {
		t.Fatalf("tenant B saw telemetry: %+v", got)
	}
	job, err := store.CreateJob(tcA, model.Job{DeviceID: devA.ID, Type: "collect_inventory", Payload: json.RawMessage(`{}`), ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteJob(tcB, job.ID, json.RawMessage(`{}`)); err == nil {
		t.Fatal("tenant B completed tenant A job")
	}
	if err := store.Audit(model.AuditEvent{TenantID: orgA.ID, Action: "device.read", ResourceType: "device", Result: "ok"}); err != nil {
		t.Fatal(err)
	}
	if got := store.ListAudit(tcB); len(got) != 0 {
		t.Fatalf("tenant B saw audit: %+v", got)
	}
}

func TestJobIdempotency(t *testing.T) {
	store, _ := Open("")
	org, _ := store.CreateOrganization("Tenant", "tenant")
	tc := tenancy.Context{TenantID: org.ID}
	dev, _ := store.CreateDevice(model.Device{TenantID: org.ID, DisplayName: "d"})
	first, err := store.CreateJob(tc, model.Job{DeviceID: dev.ID, Type: "collect_inventory", IdempotencyKey: "k1", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateJob(tc, model.Job{DeviceID: dev.ID, Type: "collect_inventory", IdempotencyKey: "k1", ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatal("idempotency key created duplicate job")
	}
}
