package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soul-room/cloud-infra/internal/assistant"
	"github.com/soul-room/cloud-infra/internal/config"
	cloudjobs "github.com/soul-room/cloud-infra/internal/jobs"
	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/storage"
	"github.com/soul-room/cloud-infra/internal/tenancy"
)

func TestAIConfigurationEncryptsKeyAndRedactsResponse(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-config")
	server := New(config.Config{MaxPayloadBytes: 1024 * 1024, AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "owner", Roles: []string{"organization_owner"}}
	body := `{"provider":"openai","model":"small-model","api_key":"secret-provider-key","enabled":true}`
	request := httptest.NewRequest(http.MethodPut, "/api/v1/ai/config", strings.NewReader(body)).WithContext(tenancy.WithContext(httptest.NewRequest(http.MethodGet, "/", nil).Context(), tc))
	response := httptest.NewRecorder()
	server.aiConfiguration(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save returned %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "secret-provider-key") || strings.Contains(response.Body.String(), "api_key_ciphertext") {
		t.Fatal("AI configuration response exposed secret material")
	}
	stored, ok := store.AIConfiguration(tc)
	if !ok || stored.APIKeyCiphertext == "" || strings.Contains(stored.APIKeyCiphertext, "secret-provider-key") {
		t.Fatalf("provider key was not encrypted: %+v", stored)
	}
	plainText, err := server.Assistant.Open(stored.APIKeyCiphertext, aiConfigPurpose(tc))
	if err != nil || plainText != "secret-provider-key" {
		t.Fatalf("encrypted key did not round-trip: %q %v", plainText, err)
	}
}

func TestAIApprovedActionIsTypedAuditedAndReplaySafe(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-action")
	device, _ := store.CreateDevice(model.Device{TenantID: organization.ID, DisplayName: "edge-1", Presence: "connected", Capabilities: []string{"jobs"}})
	server := New(config.Config{MaxPayloadBytes: 1024 * 1024, DefaultJobTTLMinutes: 60, AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "operator", Roles: []string{"fleet_operator"}}
	approval := aiActionApproval{TenantID: organization.ID, ActorID: tc.ActorID, DeviceID: device.ID, JobType: "collect_diagnostics", Nonce: "once", ExpiresAt: time.Now().Add(time.Minute)}
	token, err := server.Assistant.SealJSON(approval, "ai-action")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"approval_token": token})

	for attempt := 0; attempt < 2; attempt++ {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/actions", strings.NewReader(string(payload)))
		request = request.WithContext(tenancy.WithContext(request.Context(), tc))
		response := httptest.NewRecorder()
		server.aiAction(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("attempt %d returned %d: %s", attempt, response.Code, response.Body.String())
		}
	}
	jobs := store.ListJobs(tc)
	if len(jobs) != 1 || jobs[0].Type != "collect_diagnostics" {
		t.Fatalf("approval created unexpected jobs: %+v", jobs)
	}
	if events := store.ListAudit(tc); len(events) == 0 || events[len(events)-1].Action != "ai.action.approved" {
		t.Fatalf("approved action was not audited: %+v", events)
	}
}

func TestAssistantSnapshotIsCompactAndExcludesSensitiveDeviceFields(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-snapshot")
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "viewer"}
	_, _ = store.CreateDevice(model.Device{TenantID: organization.ID, DisplayName: "edge-1", Serial: "sensitive-serial", RemoteAccessID: "sensitive-ssh-id", CertificateSubject: "sensitive-subject", Presence: "connected"})
	server := New(config.Config{AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	snapshot, err := server.aiFleetSnapshot(tc, "fleet status")
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"sensitive-serial", "sensitive-ssh-id", "sensitive-subject"} {
		if strings.Contains(snapshot, secret) {
			t.Fatalf("snapshot exposed %q: %s", secret, snapshot)
		}
	}
	if len(snapshot) > 24*1024 {
		t.Fatalf("snapshot exceeded token-efficient budget: %d bytes", len(snapshot))
	}
}

func TestUnsupportedAIJobCannotBeApproved(t *testing.T) {
	if _, allowed := aiJobPermission("ota_update"); allowed {
		t.Fatal("assistant was allowed to approve an OTA update")
	}
	if _, allowed := aiJobPermission("restart_service"); allowed {
		t.Fatal("assistant was allowed to approve an arbitrary service restart")
	}
}

func TestAIJobAllowlistUsesValidCloudJobTypes(t *testing.T) {
	for _, jobType := range []string{"collect_diagnostics", "collect_logs", "collect_inventory", "scan_vulnerabilities", "reboot"} {
		if _, allowed := aiJobPermission(jobType); !allowed || !cloudjobs.AllowedTypes[jobType] {
			t.Fatalf("assistant job %q is not a valid cloud job", jobType)
		}
	}
}

func TestAIActionRechecksCurrentDeviceCapability(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-capability-recheck")
	device, _ := store.CreateDevice(model.Device{TenantID: organization.ID, DisplayName: "edge-1", Capabilities: []string{"jobs"}})
	server := New(config.Config{MaxPayloadBytes: 1024 * 1024, DefaultJobTTLMinutes: 60, AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "owner", Roles: []string{"organization_owner"}}
	approval := aiActionApproval{TenantID: organization.ID, ActorID: tc.ActorID, DeviceID: device.ID, JobType: "reboot", Nonce: "once", ExpiresAt: time.Now().Add(time.Minute)}
	token, err := server.Assistant.SealJSON(approval, "ai-action")
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"approval_token": token})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/actions", strings.NewReader(string(payload)))
	request = request.WithContext(tenancy.WithContext(request.Context(), tc))
	response := httptest.NewRecorder()
	server.aiAction(response, request)
	if response.Code != http.StatusConflict || len(store.ListJobs(tc)) != 0 {
		t.Fatalf("unsupported device action was queued: status=%d body=%s jobs=%+v", response.Code, response.Body.String(), store.ListJobs(tc))
	}
}

func TestAIActionsRespectReportedDeviceCapabilities(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-capabilities")
	device, _ := store.CreateDevice(model.Device{TenantID: organization.ID, DisplayName: "edge-1", Presence: "connected", Capabilities: []string{"jobs"}})
	server := New(config.Config{AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "owner", Roles: []string{"organization_owner"}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/chat", nil)
	request = request.WithContext(tenancy.WithContext(request.Context(), tc))

	proposals := []assistant.ProposedAction{
		{Kind: "device_job", DeviceID: device.ID, JobType: "collect_logs"},
		{Kind: "device_job", DeviceID: device.ID, JobType: "scan_vulnerabilities"},
		{Kind: "device_job", DeviceID: device.ID, JobType: "reboot"},
	}
	actions := server.prepareAIActions(request, tc, proposals)
	if len(actions) != 1 || actions[0].JobType != "collect_logs" {
		t.Fatalf("unsupported actions were offered: %+v", actions)
	}

	device.Capabilities = append(device.Capabilities, "job:reboot", "security:trivy")
	if err := store.UpdateDevice(device); err != nil {
		t.Fatal(err)
	}
	actions = server.prepareAIActions(request, tc, proposals)
	if len(actions) != 3 {
		t.Fatalf("reported actions were not offered: %+v", actions)
	}
}

func TestAssistantSnapshotListsNewestJobsFirst(t *testing.T) {
	store, _ := storage.Open("")
	organization, _ := store.CreateOrganization("Tenant", "assistant-recency")
	tc := tenancy.Context{TenantID: organization.ID, ActorID: "viewer"}
	device, _ := store.CreateDevice(model.Device{TenantID: organization.ID, DisplayName: "edge-1"})
	oldJob, _ := store.CreateJob(tc, model.Job{DeviceID: device.ID, Type: "collect_inventory", ExpiresAt: time.Now().Add(time.Hour)})
	time.Sleep(10 * time.Millisecond)
	newJob, _ := store.CreateJob(tc, model.Job{DeviceID: device.ID, Type: "collect_logs", ExpiresAt: time.Now().Add(time.Hour)})
	server := New(config.Config{AIRequestTimeout: time.Second, AIRequestsPerMinute: 30}, store, nil)
	snapshotJSON, err := server.aiFleetSnapshot(tc, "recent jobs")
	if err != nil {
		t.Fatal(err)
	}
	var snapshot fleetSnapshot
	if err := json.Unmarshal([]byte(snapshotJSON), &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.RecentJobs) != 2 || snapshot.RecentJobs[0].ID != newJob.ID || snapshot.RecentJobs[1].ID != oldJob.ID {
		t.Fatalf("jobs were not ordered by recency: %+v", snapshot.RecentJobs)
	}
}
