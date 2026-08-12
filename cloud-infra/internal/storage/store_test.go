package storage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/tenancy"
)

func TestBootstrapOrganizationOwnerRunsOnlyOnce(t *testing.T) {
	store, err := Open("")
	if err != nil {
		t.Fatal(err)
	}
	created, err := store.BootstrapOrganizationOwner("Acme Devices", "acme-devices", "admin@acme.com", "first-hash")
	if err != nil || !created {
		t.Fatalf("first bootstrap failed: created=%v err=%v", created, err)
	}
	created, err = store.BootstrapOrganizationOwner("Other Company", "other", "owner@other.com", "replacement-hash")
	if err != nil || created {
		t.Fatalf("second bootstrap changed installation: created=%v err=%v", created, err)
	}
	organizations := store.ListOrganizations()
	if len(organizations) != 1 || organizations[0].Name != "Acme Devices" {
		t.Fatalf("bootstrap organization changed: %+v", organizations)
	}
	owner, err := store.FindUserByEmail("admin@acme.com")
	if err != nil || owner.PasswordHash != "first-hash" {
		t.Fatalf("bootstrap owner changed: %+v err=%v", owner, err)
	}
	if _, err := store.FindUserByEmail("owner@other.com"); err == nil {
		t.Fatal("second bootstrap created another owner")
	}
}

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

func TestDeleteDeviceRemovesOperationalData(t *testing.T) {
	store, _ := Open("")
	org, _ := store.CreateOrganization("Tenant", "tenant-delete")
	tc := tenancy.Context{TenantID: org.ID}
	device, _ := store.CreateDevice(model.Device{TenantID: org.ID, DisplayName: "edge"})
	_ = store.AddTelemetry(tc, []model.Metric{{DeviceID: device.ID, Name: "cpu", Value: 1, DeviceTime: time.Now()}})
	_ = store.SaveInventory(tc, device.ID, json.RawMessage(`{"os":"linux"}`))
	_, _ = store.CreateJob(tc, model.Job{DeviceID: device.ID, Type: "collect_inventory", ExpiresAt: time.Now().Add(time.Hour)})
	if err := store.DeleteDevice(tc, device.ID); err != nil {
		t.Fatal(err)
	}
	if len(store.ListDevices(tc)) != 0 || len(store.ListTelemetry(tc, device.ID)) != 0 || len(store.ListInventory(tc)) != 0 || len(store.ListJobs(tc)) != 0 {
		t.Fatal("device operational data was not removed")
	}
}

func TestDeleteJobRequiresTerminalState(t *testing.T) {
	store, _ := Open("")
	org, _ := store.CreateOrganization("Tenant", "tenant-job-delete")
	tc := tenancy.Context{TenantID: org.ID}
	device, _ := store.CreateDevice(model.Device{TenantID: org.ID, DisplayName: "edge"})
	job, _ := store.CreateJob(tc, model.Job{DeviceID: device.ID, Type: "collect_inventory", State: "queued", ExpiresAt: time.Now().Add(time.Hour)})
	if err := store.DeleteJob(tc, job.ID); err == nil {
		t.Fatal("queued job was deleted")
	}
	job.State = "succeeded"
	store.data.Jobs[job.ID] = job
	if err := store.DeleteJob(tc, job.ID); err != nil {
		t.Fatal(err)
	}
}
