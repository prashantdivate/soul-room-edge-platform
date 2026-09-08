package server

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/soul-room/cloud-infra/internal/audit"
	"github.com/soul-room/cloud-infra/internal/auth"
	"github.com/soul-room/cloud-infra/internal/certificates"
	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/enrollment"
	"github.com/soul-room/cloud-infra/internal/jobs"
	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/observability"
	"github.com/soul-room/cloud-infra/internal/ota"
	"github.com/soul-room/cloud-infra/internal/protocol"
	"github.com/soul-room/cloud-infra/internal/rbac"
	platformsettings "github.com/soul-room/cloud-infra/internal/settings"
	"github.com/soul-room/cloud-infra/internal/storage"
	"github.com/soul-room/cloud-infra/internal/telemetry"
	"github.com/soul-room/cloud-infra/internal/tenancy"
)

type Server struct {
	Config config.Config
	Store  *storage.Store
	CA     *certificates.Authority
	Hash   auth.PasswordHasher
}

func New(cfg config.Config, store *storage.Store, ca *certificates.Authority) *Server {
	return &Server{Config: cfg, Store: store, CA: ca, Hash: auth.PBKDF2Hasher{}}
}

func (s *Server) ControlHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", observability.Healthz)
	mux.HandleFunc("/readyz", observability.Healthz)
	mux.HandleFunc("/metrics", observability.Metrics)
	mux.HandleFunc("/api/v1/auth/login", s.login)
	mux.HandleFunc("/api/v1/auth/logout", s.logout)
	mux.Handle("/api/v1/auth/session", s.withAuth(http.HandlerFunc(s.currentSession)))
	mux.Handle("/api/v1/organizations", s.withAuth(http.HandlerFunc(s.organizations)))
	mux.Handle("/api/v1/enrollment-tokens", s.withAuth(http.HandlerFunc(s.enrollmentTokens)))
	mux.Handle("/api/v1/devices", s.withAuth(http.HandlerFunc(s.devices)))
	mux.Handle("/api/v1/downstream", s.withAuth(http.HandlerFunc(s.downstream)))
	mux.Handle("/api/v1/inventory", s.withAuth(http.HandlerFunc(s.inventory)))
	mux.Handle("/api/v1/telemetry", s.withAuth(http.HandlerFunc(s.telemetryQuery)))
	mux.Handle("/api/v1/jobs", s.withAuth(http.HandlerFunc(s.jobs)))
	mux.Handle("/api/v1/audit", s.withAuth(http.HandlerFunc(s.audit)))
	mux.Handle("/api/v1/roles", s.withAuth(http.HandlerFunc(s.roles)))
	mux.Handle("/api/v1/users", s.withAuth(http.HandlerFunc(s.users)))
	mux.Handle("/api/v1/dev-ca", s.withAuth(http.HandlerFunc(s.devCA)))
	mux.Handle("/api/v1/platform-info", s.withAuth(http.HandlerFunc(s.platformInfo)))
	mux.Handle("/api/v1/platform-settings", s.withAuth(http.HandlerFunc(s.platformSettings)))
	mux.Handle("/api/v1/deployments", s.withAuth(staticList("deployments")))
	mux.Handle("/api/v1/applications", s.withAuth(staticList("applications")))
	mux.Handle("/api/v1/artifacts", s.withAuth(http.HandlerFunc(s.artifacts)))
	mux.Handle("/api/v1/ota", s.withAuth(http.HandlerFunc(s.otaCampaigns)))
	mux.Handle("/api/v1/remote-access", s.withAuth(http.HandlerFunc(s.remoteAccess)))
	mux.Handle("/api/v1/alerts", s.withAuth(staticList("alerts")))
	return secureHeaders(mux)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "soul_room_session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	writeJSON(w, http.StatusOK, map[string]string{"status": "signed_out"})
}

func (s *Server) currentSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"membership": model.Membership{TenantID: tc.TenantID, UserID: tc.ActorID, Role: tc.Roles[0]}})
}

func (s *Server) platformInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	settings, _ := s.effectivePlatformSettings(tc)
	writeJSON(w, http.StatusOK, map[string]any{
		"device_gateway_endpoint":  "https://" + s.Config.DevicePublicHost + ":8443",
		"remote_access_provider":   "shellhub",
		"remote_access_url":        settings.ShellHubURL,
		"remote_access_configured": settings.ShellHubURL != "",
		"remote_access_managed":    s.Config.ShellHubManaged,
		"remote_access_ssh_port":   settings.ShellHubSSHPort,
	})
}

func (s *Server) platformSettings(w http.ResponseWriter, r *http.Request) {
	tc, _ := tenancy.Require(r.Context())
	if r.Method == http.MethodGet {
		value, source := s.effectivePlatformSettings(tc)
		writeJSON(w, http.StatusOK, map[string]any{
			"settings":   value,
			"source":     source,
			"can_manage": rbac.Authorize(r.Context(), rbac.TenantSettings) == nil,
			"infrastructure": []map[string]any{
				{"key": "device_gateway", "label": "Device gateway", "value": "https://" + s.Config.DevicePublicHost + ":8443", "restart_required": true},
				{"key": "remote_access_mode", "label": "ShellHub service", "value": map[bool]string{true: "Bundled", false: "External"}[s.Config.ShellHubManaged], "restart_required": true},
				{"key": "session_lifetime", "label": "Session lifetime", "value": s.Config.SessionTTL.String(), "restart_required": true},
				{"key": "request_limit", "label": "Request size limit", "value": strconv.FormatInt(s.Config.MaxPayloadBytes, 10) + " bytes", "restart_required": true},
			},
		})
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.TenantSettings); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	if r.Method == http.MethodDelete {
		if err := s.Store.ResetPlatformSettings(tc); err != nil {
			errorJSON(w, http.StatusInternalServerError, "settings_reset_failed", err.Error())
			return
		}
		audit.Record(s.Store, tc, "platform_settings.reset", "organization", tc.TenantID, "success", nil)
		value, _ := s.effectivePlatformSettings(tc)
		writeJSON(w, http.StatusOK, map[string]any{"settings": value, "source": "deployment", "can_manage": true})
		return
	}
	if r.Method != http.MethodPut {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var value model.PlatformSettings
	if err := decode(w, r, s.Config.MaxPayloadBytes, &value); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	value.OrganizationName = strings.TrimSpace(value.OrganizationName)
	value.CompanyDomain = strings.ToLower(strings.TrimSpace(value.CompanyDomain))
	value.ShellHubURL = strings.TrimRight(strings.TrimSpace(value.ShellHubURL), "/")
	if err := platformsettings.Validate(value); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_settings", err.Error())
		return
	}
	if err := s.Store.SavePlatformSettings(tc, value); err != nil {
		errorJSON(w, http.StatusInternalServerError, "settings_save_failed", err.Error())
		return
	}
	value, _ = s.effectivePlatformSettings(tc)
	audit.Record(s.Store, tc, "platform_settings.updated", "organization", tc.TenantID, "success", map[string]string{"company_domain": value.CompanyDomain})
	writeJSON(w, http.StatusOK, map[string]any{"settings": value, "source": "platform", "can_manage": true})
}

func (s *Server) effectivePlatformSettings(tc tenancy.Context) (model.PlatformSettings, string) {
	if value, ok := s.Store.PlatformSettings(tc); ok {
		if value.ShellHubURL == "" {
			value.ShellHubURL = s.Config.ShellHubURL
		}
		if value.ShellHubSSHPort == 0 {
			value.ShellHubSSHPort = s.Config.ShellHubSSHPort
		}
		return value, "platform"
	}
	organizationName := s.Config.OrganizationName
	if organization, err := s.Store.Organization(tc); err == nil && organization.Name != "" {
		organizationName = organization.Name
	}
	return model.PlatformSettings{
		OrganizationName:          organizationName,
		CompanyDomain:             s.Config.CompanyDomain,
		ShellHubURL:               s.Config.ShellHubURL,
		ShellHubSSHPort:           s.Config.ShellHubSSHPort,
		DeviceOfflineMinutes:      s.Config.DeviceOfflineMinutes,
		DefaultTelemetryWindow:    s.Config.DefaultTelemetryWindow,
		DefaultOTAPilotPercent:    s.Config.DefaultOTAPilotPercent,
		DefaultEnrollmentTTLHours: s.Config.DefaultEnrollmentTTLHours,
		DefaultJobTTLMinutes:      s.Config.DefaultJobTTLMinutes,
	}, "deployment"
}

func (s *Server) DeviceHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", observability.Healthz)
	mux.HandleFunc("/device/v1/enroll", s.deviceEnroll)
	mux.HandleFunc("/v1/enroll", s.deviceEnroll)
	mux.HandleFunc("/device/v1/heartbeat", s.deviceHeartbeat)
	mux.HandleFunc("/v1/heartbeat", s.deviceHeartbeat)
	mux.HandleFunc("/device/v1/telemetry", s.deviceTelemetry)
	mux.HandleFunc("/v1/telemetry", s.deviceTelemetry)
	mux.HandleFunc("/device/v1/inventory", s.deviceInventory)
	mux.HandleFunc("/v1/inventory", s.deviceInventory)
	mux.HandleFunc("/device/v1/jobs/next", s.deviceNextJob)
	mux.HandleFunc("/device/v1/jobs/result", s.deviceJobResult)
	mux.HandleFunc("/v1/jobs/next", s.deviceNextJob)
	mux.HandleFunc("/v1/jobs/result", s.deviceJobResult)
	mux.HandleFunc("/v1/offline", s.deviceOffline)
	return secureHeaders(mux)
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	user, err := s.Store.FindUserByEmail(strings.ToLower(req.Email))
	if err != nil || !s.Hash.Verify(req.Password, user.PasswordHash) || time.Now().Before(user.LockedUntil) {
		errorJSON(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
		return
	}
	session, token, err := auth.NewSession(user.ID, s.Config.SessionTTL)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not create session")
		return
	}
	if err := s.Store.SaveSession(session); err != nil {
		errorJSON(w, http.StatusInternalServerError, "session_error", "could not save session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "soul_room_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt})
	writeJSON(w, http.StatusOK, map[string]any{"session_expires_at": session.ExpiresAt, "memberships": s.Store.Memberships(user.ID)})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r.Header.Get("Authorization"))
		if token == "" {
			if c, err := r.Cookie("soul_room_session"); err == nil {
				token = c.Value
			}
		}
		session, err := s.Store.SessionByToken(token)
		if err != nil {
			errorJSON(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		tenantID := r.Header.Get("X-Tenant-ID")
		var roles []string
		for _, m := range s.Store.Memberships(session.UserID) {
			if tenantID == "" {
				tenantID = m.TenantID
			}
			if m.TenantID == tenantID {
				roles = append(roles, m.Role)
			}
		}
		if tenantID == "" || len(roles) == 0 {
			errorJSON(w, http.StatusForbidden, "forbidden", "tenant membership required")
			return
		}
		tc := tenancy.Context{TenantID: tenantID, ActorID: session.UserID, Roles: roles}
		next.ServeHTTP(w, r.WithContext(tenancy.WithContext(r.Context(), tc)))
	})
}

func (s *Server) organizations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": s.Store.ListOrganizations()})
}

func (s *Server) enrollmentTokens(w http.ResponseWriter, r *http.Request) {
	tc, _ := tenancy.Require(r.Context())
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"enrollment_tokens": s.Store.ListEnrollmentTokens(tc)})
		return
	}
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.DeviceEnroll); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	var req struct {
		ProfileID string `json:"profile_id"`
		TTL       string `json:"ttl"`
	}
	_ = decode(w, r, s.Config.MaxPayloadBytes, &req)
	settings, _ := s.effectivePlatformSettings(tc)
	ttl := time.Duration(settings.DefaultEnrollmentTTLHours) * time.Hour
	if req.TTL != "" {
		if parsed, err := time.ParseDuration(req.TTL); err == nil {
			ttl = parsed
		}
	}
	token, err := enrollment.NewToken(s.Store, tc, req.ProfileID, ttl)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "token_error", err.Error())
		return
	}
	token.TokenHash = ""
	audit.Record(s.Store, tc, "enrollment_token.created", "enrollment_token", token.ID, "success", map[string]string{"profile_id": req.ProfileID})
	writeJSON(w, http.StatusCreated, token)
}

func (s *Server) downstream(w http.ResponseWriter, r *http.Request) {
	if err := rbac.Authorize(r.Context(), rbac.DeviceRead); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"downstream": s.Store.ListDownstream(tc)})
}

func (s *Server) inventory(w http.ResponseWriter, r *http.Request) {
	if err := rbac.Authorize(r.Context(), rbac.DeviceRead); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"inventory": s.Store.ListInventory(tc)})
}

func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	if err := rbac.Authorize(r.Context(), rbac.DeviceRead); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	if r.Method == http.MethodGet {
		devices := s.Store.ListDevices(tc)
		settings, _ := s.effectivePlatformSettings(tc)
		offlineAfter := time.Duration(settings.DeviceOfflineMinutes) * time.Minute
		for index := range devices {
			if !devices[index].LastSeenAt.IsZero() && time.Since(devices[index].LastSeenAt) > offlineAfter {
				devices[index].Presence = "offline"
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"devices": devices})
		return
	}
	if r.Method == http.MethodDelete {
		if err := rbac.Authorize(r.Context(), rbac.DeviceDelete); err != nil {
			errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		deviceID := strings.TrimSpace(r.URL.Query().Get("device_id"))
		if deviceID == "" {
			errorJSON(w, http.StatusBadRequest, "bad_request", "device_id is required")
			return
		}
		if err := s.Store.DeleteDevice(tc, deviceID); err != nil {
			errorJSON(w, http.StatusBadRequest, "device_delete_failed", err.Error())
			return
		}
		audit.Record(s.Store, tc, "device.removed", "device", deviceID, "success", nil)
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed", "device_id": deviceID})
		return
	}
	if r.Method != http.MethodPatch {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.DeviceUpdate); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	var req struct {
		DeviceID       string                `json:"device_id"`
		Location       *model.DeviceLocation `json:"location,omitempty"`
		RemoteAccessID *string               `json:"remote_access_id,omitempty"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	device, err := s.Store.GetDevice(tc, req.DeviceID)
	if err != nil {
		errorJSON(w, http.StatusNotFound, "not_found", "device not found")
		return
	}
	if req.Location != nil {
		if !validCoordinates(req.Location.Latitude, req.Location.Longitude) {
			errorJSON(w, http.StatusBadRequest, "invalid_location", "latitude or longitude is invalid")
			return
		}
		req.Location.Source = "manual"
		req.Location.UpdatedAt = time.Now()
		device.Location = req.Location
	}
	if req.RemoteAccessID != nil {
		device.RemoteAccessID = strings.TrimSpace(*req.RemoteAccessID)
	}
	if err := s.Store.UpdateDevice(device); err != nil {
		errorJSON(w, http.StatusInternalServerError, "device_update_failed", err.Error())
		return
	}
	audit.Record(s.Store, tc, "device.settings_updated", "device", device.ID, "success", map[string]string{"location": boolString(req.Location != nil), "remote_access": boolString(req.RemoteAccessID != nil)})
	writeJSON(w, http.StatusOK, device)
}

func (s *Server) otaCampaigns(w http.ResponseWriter, r *http.Request) {
	tc, _ := tenancy.Require(r.Context())
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"ota_campaigns": s.Store.ListOTACampaigns(tc)})
		return
	}
	if r.Method == http.MethodPatch {
		s.promoteOTACampaign(w, r, tc)
		return
	}
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.DeploymentCreate); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	var campaign ota.Campaign
	if err := decode(w, r, s.Config.MaxPayloadBytes, &campaign); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := ota.ValidateMetadata(campaign); err != nil {
		errorJSON(w, http.StatusBadRequest, "invalid_campaign", err.Error())
		return
	}
	for _, deviceID := range campaign.TargetIDs {
		device, err := s.Store.GetDevice(tc, deviceID)
		if err != nil {
			errorJSON(w, http.StatusBadRequest, "invalid_target", "campaign contains an unknown device")
			return
		}
		if campaign.Architecture != "" && device.Architecture != "" && canonicalArchitecture(campaign.Architecture) != canonicalArchitecture(device.Architecture) {
			errorJSON(w, http.StatusBadRequest, "incompatible_target", "campaign architecture does not match every target")
			return
		}
		if !containsString(device.Capabilities, "ota:"+campaign.Adapter) {
			errorJSON(w, http.StatusBadRequest, "incompatible_target", "every target must report support for the selected OTA adapter")
			return
		}
	}
	campaign.CreatedBy = tc.ActorID
	campaign.State = "ready"
	created, err := s.Store.CreateOTACampaign(tc, campaign)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "campaign_error", err.Error())
		return
	}
	canaryCount := int(math.Ceil(float64(len(created.TargetIDs)) * float64(created.CanaryPercent) / 100))
	if canaryCount < 1 {
		canaryCount = 1
	}
	if _, err := s.queueOTAJobs(tc, created, created.TargetIDs[:canaryCount]); err != nil {
		created.State = "failed"
		_ = s.Store.UpdateOTACampaign(tc, created)
		errorJSON(w, http.StatusInternalServerError, "ota_queue_failed", err.Error())
		return
	}
	if canaryCount < len(created.TargetIDs) {
		created.State = "canary_queued"
	} else {
		created.State = "queued"
	}
	if err := s.Store.UpdateOTACampaign(tc, created); err != nil {
		errorJSON(w, http.StatusInternalServerError, "campaign_update_failed", err.Error())
		return
	}
	audit.Record(s.Store, tc, "ota_campaign.created", "ota_campaign", created.ID, "success", map[string]string{"adapter": created.Adapter, "targets": boolString(len(created.TargetIDs) > 0)})
	writeJSON(w, http.StatusCreated, created)
}

func canonicalArchitecture(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "arm64", "aarch64":
		return "arm64"
	case "arm", "armv7", "armv7l", "armhf":
		return "armv7"
	case "amd64", "x86_64":
		return "amd64"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func (s *Server) promoteOTACampaign(w http.ResponseWriter, r *http.Request, tc tenancy.Context) {
	if err := rbac.Authorize(r.Context(), rbac.DeploymentApprove); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	var req struct {
		CampaignID string `json:"campaign_id"`
		Action     string `json:"action"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	campaign, err := s.Store.GetOTACampaign(tc, req.CampaignID)
	if err != nil {
		errorJSON(w, http.StatusNotFound, "campaign_not_found", "campaign was not found")
		return
	}
	if req.Action != "promote" || campaign.State != "canary_complete" {
		errorJSON(w, http.StatusConflict, "campaign_not_promotable", "only a completed pilot group can be promoted")
		return
	}
	existing := map[string]bool{}
	for _, job := range s.Store.JobsForDeployment(tc, campaign.ID) {
		existing[job.DeviceID] = true
	}
	remaining := []string{}
	for _, deviceID := range campaign.TargetIDs {
		if !existing[deviceID] {
			remaining = append(remaining, deviceID)
		}
	}
	if len(remaining) == 0 {
		errorJSON(w, http.StatusConflict, "campaign_already_promoted", "all campaign targets already have an update job")
		return
	}
	if _, err := s.queueOTAJobs(tc, campaign, remaining); err != nil {
		errorJSON(w, http.StatusInternalServerError, "ota_queue_failed", err.Error())
		return
	}
	campaign.State = "queued"
	if err := s.Store.UpdateOTACampaign(tc, campaign); err != nil {
		errorJSON(w, http.StatusInternalServerError, "campaign_update_failed", err.Error())
		return
	}
	audit.Record(s.Store, tc, "ota_campaign.promoted", "ota_campaign", campaign.ID, "success", map[string]string{"targets": boolString(len(remaining) > 0)})
	writeJSON(w, http.StatusOK, campaign)
}

func (s *Server) queueOTAJobs(tc tenancy.Context, campaign ota.Campaign, deviceIDs []string) (int, error) {
	payload, err := json.Marshal(map[string]any{
		"update_id": campaign.ID, "adapter": campaign.Adapter, "version": campaign.Version,
		"product": campaign.Product, "architecture": campaign.Architecture,
		"artifact_url": campaign.ArtifactURL, "artifact_size": campaign.ArtifactSize,
		"digest": campaign.Digest, "signature": campaign.Signature, "signing_key_id": campaign.SigningKeyID,
		"compatible_from": campaign.CompatibleFrom, "flatpak_ref": campaign.FlatpakRef,
		"flatpak_remote": campaign.FlatpakRemote, "flatpak_commit": campaign.FlatpakCommit,
		"repository_url": campaign.FlatpakRepositoryURL,
		"ostree_remote":  campaign.OSTreeRemote, "ostree_ref": campaign.OSTreeRef,
		"ostree_commit": campaign.OSTreeCommit, "ostree_os": campaign.OSTreeOS,
	})
	if err != nil {
		return 0, err
	}
	created := 0
	for _, deviceID := range deviceIDs {
		job, err := jobs.New(deviceID, "ota_update", payload, tc.ActorID, 24*time.Hour)
		if err != nil {
			return created, err
		}
		job.DeploymentID = campaign.ID
		job.TimeoutSeconds = 7200
		job.IdempotencyKey = campaign.ID + ":" + deviceID
		if _, err := s.Store.CreateJob(tc, job); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func (s *Server) remoteAccess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.RemoteCreate); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	settings, _ := s.effectivePlatformSettings(tc)
	if settings.ShellHubURL == "" {
		errorJSON(w, http.StatusServiceUnavailable, "remote_access_unconfigured", "ShellHub URL is not configured")
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
		User     string `json:"user"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	device, err := s.Store.GetDevice(tc, req.DeviceID)
	if err != nil || device.RemoteAccessID == "" {
		errorJSON(w, http.StatusBadRequest, "remote_access_unmapped", "device does not have a ShellHub SSHID")
		return
	}
	if !safeSSHPart(req.User) || !safeSSHPart(device.RemoteAccessID) {
		errorJSON(w, http.StatusBadRequest, "invalid_ssh_identity", "SSH user or ShellHub SSHID is invalid")
		return
	}
	sshCommand := "ssh "
	if settings.ShellHubSSHPort != 22 {
		sshCommand += "-p " + strconv.Itoa(settings.ShellHubSSHPort) + " "
	}
	sshCommand += req.User + "@" + device.RemoteAccessID
	audit.Record(s.Store, tc, "remote_access.requested", "device", device.ID, "success", map[string]string{"provider": "shellhub", "user": req.User})
	writeJSON(w, http.StatusCreated, map[string]string{"provider": "shellhub", "launch_url": settings.ShellHubURL, "ssh_command": sshCommand})
}

func (s *Server) telemetryQuery(w http.ResponseWriter, r *http.Request) {
	if err := rbac.Authorize(r.Context(), rbac.TelemetryRead); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	deviceID := r.URL.Query().Get("device_id")
	writeJSON(w, http.StatusOK, map[string]any{"metrics": s.Store.ListTelemetry(tc, deviceID)})
}

func (s *Server) jobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tc, _ := tenancy.Require(r.Context())
		writeJSON(w, http.StatusOK, map[string]any{"jobs": s.Store.ListJobs(tc)})
		return
	}
	if r.Method == http.MethodDelete {
		if err := rbac.Authorize(r.Context(), rbac.JobDelete); err != nil {
			errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		tc, _ := tenancy.Require(r.Context())
		jobID := strings.TrimSpace(r.URL.Query().Get("job_id"))
		if jobID == "" {
			errorJSON(w, http.StatusBadRequest, "bad_request", "job_id is required")
			return
		}
		if err := s.Store.DeleteJob(tc, jobID); err != nil {
			errorJSON(w, http.StatusConflict, "job_delete_failed", err.Error())
			return
		}
		audit.Record(s.Store, tc, "job.history_removed", "job", jobID, "success", nil)
		writeJSON(w, http.StatusOK, map[string]string{"status": "removed", "job_id": jobID})
		return
	}
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.DeviceCommand); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	var req struct {
		DeviceID string          `json:"device_id"`
		Type     string          `json:"type"`
		Payload  json.RawMessage `json:"payload"`
		TTL      string          `json:"ttl"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	settings, _ := s.effectivePlatformSettings(tc)
	ttl := time.Duration(settings.DefaultJobTTLMinutes) * time.Minute
	if req.TTL != "" {
		if d, err := time.ParseDuration(req.TTL); err == nil {
			ttl = d
		}
	}
	job, err := jobs.New(req.DeviceID, req.Type, req.Payload, tc.ActorID, ttl)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_job", err.Error())
		return
	}
	job.IdempotencyKey = r.Header.Get("Idempotency-Key")
	job, err = s.Store.CreateJob(tc, job)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "job_error", err.Error())
		return
	}
	audit.Record(s.Store, tc, "device_job.queued", "job", job.ID, "success", map[string]string{"device_id": job.DeviceID, "type": job.Type})
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) users(w http.ResponseWriter, r *http.Request) {
	tc, _ := tenancy.Require(r.Context())
	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"users": s.Store.ListUsers(tc)})
		return
	}
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if err := rbac.Authorize(r.Context(), rbac.UserManage); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email {
		errorJSON(w, http.StatusBadRequest, "invalid_email", "enter a valid email address")
		return
	}
	if len(req.Password) < 12 || len(req.Password) > 256 {
		errorJSON(w, http.StatusBadRequest, "invalid_password", "temporary password must contain 12 to 256 characters")
		return
	}
	if !canAssignRole(tc.Roles, req.Role) {
		errorJSON(w, http.StatusForbidden, "invalid_role", "you cannot assign this role")
		return
	}
	passwordHash, err := s.Hash.Hash(req.Password)
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "password_hash_failed", "could not secure the temporary password")
		return
	}
	user, err := s.Store.CreateTenantUser(tc, req.Email, passwordHash, req.Role)
	if errors.Is(err, storage.ErrDuplicate) {
		errorJSON(w, http.StatusConflict, "user_exists", "a user with this email already exists")
		return
	}
	if err != nil {
		errorJSON(w, http.StatusInternalServerError, "user_create_failed", err.Error())
		return
	}
	audit.Record(s.Store, tc, "user.created", "user", user.ID, "success", map[string]string{"role": req.Role})
	writeJSON(w, http.StatusCreated, user)
}

func canAssignRole(actorRoles []string, role string) bool {
	if _, exists := rbac.RolePermissions[role]; !exists {
		return false
	}
	for _, actorRole := range actorRoles {
		if actorRole == "platform_administrator" {
			return true
		}
		if actorRole == "organization_owner" && role != "platform_administrator" {
			return true
		}
		if actorRole == "organization_administrator" && role != "platform_administrator" && role != "organization_owner" {
			return true
		}
	}
	return false
}

func (s *Server) artifacts(w http.ResponseWriter, r *http.Request) {
	tc, _ := tenancy.Require(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": s.Store.ListArtifacts(tc)})
}

func (s *Server) devCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || s.CA == nil {
		errorJSON(w, http.StatusNotFound, "not_found", "development CA is unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", `attachment; filename="soul-room-dev-ca.pem"`)
	_, _ = w.Write(s.CA.CertPEM)
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if err := rbac.Authorize(r.Context(), rbac.AuditRead); err != nil {
		errorJSON(w, http.StatusForbidden, "forbidden", "permission denied")
		return
	}
	tc, _ := tenancy.Require(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"audit": s.Store.ListAudit(tc)})
}

func (s *Server) roles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, rbac.RolePermissions)
}

func staticList(name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{name: []any{}})
	})
}

func (s *Server) deviceEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		errorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	var req protocol.EnrollmentRequest
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	svc := enrollment.Service{Store: s.Store, CA: s.CA, CertificateTTL: s.Config.CertificateTTL, ServerEndpoint: "https://" + r.Host}
	res, err := svc.Enroll(r.Context(), req)
	if err != nil {
		errorJSON(w, http.StatusForbidden, "enrollment_rejected", err.Error())
		return
	}
	audit.Record(s.Store, tenancy.Context{TenantID: res.TenantID}, "device.enrolled", "device", res.DeviceID, "success", map[string]string{"name": req.RequestedName})
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	env, tc, err := s.deviceEnvelope(w, r)
	if err != nil {
		return
	}
	var hb protocol.Heartbeat
	_ = json.Unmarshal(env.Payload, &hb)
	device, err := s.Store.GetDevice(tc, env.DeviceID)
	if err != nil {
		errorJSON(w, http.StatusForbidden, "unknown_device", "device not registered")
		return
	}
	device.LastSeenAt = time.Now()
	device.Presence = "connected"
	device.AgentVersion = hb.AgentVersion
	device.HealthState = hb.Health
	if hb.Location != nil && validCoordinates(hb.Location.Latitude, hb.Location.Longitude) {
		device.Location = &model.DeviceLocation{Latitude: hb.Location.Latitude, Longitude: hb.Location.Longitude, AccuracyM: hb.Location.AccuracyM, Source: hb.Location.Source, Label: hb.Location.Label, UpdatedAt: time.Now()}
	}
	_ = s.Store.UpdateDevice(device)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "message_id": env.MessageID})
}

func validCoordinates(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}
func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
func safeSSHPart(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("._-@", r) {
			return false
		}
	}
	return true
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (s *Server) deviceTelemetry(w http.ResponseWriter, r *http.Request) {
	env, tc, err := s.deviceEnvelope(w, r)
	if err != nil {
		return
	}
	var batch protocol.TelemetryBatch
	if err := json.Unmarshal(env.Payload, &batch); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_telemetry", "invalid telemetry payload")
		return
	}
	metrics, err := telemetry.Normalize(env.DeviceID, env.MessageID, batch)
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_telemetry", err.Error())
		return
	}
	if err := s.Store.AddTelemetry(tc, metrics); err != nil {
		errorJSON(w, http.StatusBadRequest, "telemetry_rejected", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accepted": len(metrics), "message_id": env.MessageID})
}

func (s *Server) deviceInventory(w http.ResponseWriter, r *http.Request) {
	env, tc, err := s.deviceEnvelope(w, r)
	if err != nil {
		return
	}
	if err := s.Store.SaveInventory(tc, env.DeviceID, env.Payload); err != nil {
		errorJSON(w, http.StatusBadRequest, "inventory_rejected", err.Error())
		return
	}
	var inventory struct {
		Hostname      string   `json:"hostname"`
		Architecture  string   `json:"architecture"`
		OS            string   `json:"os"`
		Kernel        string   `json:"kernel"`
		HardwareModel string   `json:"hardware_model"`
		Serial        string   `json:"serial"`
		AgentVersion  string   `json:"agent_version"`
		Capabilities  []string `json:"capabilities"`
	}
	if json.Unmarshal(env.Payload, &inventory) == nil {
		device, getErr := s.Store.GetDevice(tc, env.DeviceID)
		if getErr == nil {
			device.Architecture = firstNonEmpty(inventory.Architecture, device.Architecture)
			device.OS = firstNonEmpty(inventory.OS, device.OS)
			device.Kernel = firstNonEmpty(inventory.Kernel, device.Kernel)
			device.HardwareModel = firstUseful(inventory.HardwareModel, device.HardwareModel)
			device.Serial = firstUseful(inventory.Serial, device.Serial)
			device.AgentVersion = firstNonEmpty(inventory.AgentVersion, device.AgentVersion)
			if len(inventory.Capabilities) > 0 {
				device.Capabilities = inventory.Capabilities
			}
			if device.DisplayName == "edge-device" && inventory.Hostname != "" {
				device.DisplayName = inventory.Hostname
			}
			_ = s.Store.UpdateDevice(device)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "message_id": env.MessageID})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstUseful(values ...string) string {
	for _, value := range values {
		if value != "" && value != "unknown" {
			return value
		}
	}
	return "unknown"
}

func (s *Server) deviceOffline(w http.ResponseWriter, r *http.Request) {
	env, _, err := s.deviceEnvelope(w, r)
	if err != nil {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted", "message_id": env.MessageID})
}

func (s *Server) deviceNextJob(w http.ResponseWriter, r *http.Request) {
	tc, err := s.deviceContextFromHeaders(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "unauthorized", "device identity required")
		return
	}
	deviceID := r.Header.Get("X-Device-ID")
	job, err := s.Store.NextJobForDevice(tc, deviceID)
	if errors.Is(err, storage.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"job": nil})
		return
	}
	if err != nil {
		errorJSON(w, http.StatusBadRequest, "job_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (s *Server) deviceJobResult(w http.ResponseWriter, r *http.Request) {
	tc, err := s.deviceContextFromHeaders(r)
	if err != nil {
		errorJSON(w, http.StatusUnauthorized, "unauthorized", "device identity required")
		return
	}
	var req struct {
		JobID  string          `json:"job_id"`
		Result json.RawMessage `json:"result"`
	}
	if err := decode(w, r, s.Config.MaxPayloadBytes, &req); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := s.Store.CompleteJob(tc, req.JobID, req.Result); err != nil {
		errorJSON(w, http.StatusBadRequest, "job_result_rejected", err.Error())
		return
	}
	audit.Record(s.Store, tc, "device_job.completed", "job", req.JobID, "success", nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Server) deviceEnvelope(w http.ResponseWriter, r *http.Request) (protocol.Envelope, tenancy.Context, error) {
	var env protocol.Envelope
	if err := decode(w, r, s.Config.MaxPayloadBytes, &env); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_envelope", err.Error())
		return env, tenancy.Context{}, err
	}
	if err := env.Validate(time.Now(), 10*time.Minute); err != nil {
		errorJSON(w, http.StatusBadRequest, "bad_envelope", err.Error())
		return env, tenancy.Context{}, err
	}
	tc := tenancy.Context{TenantID: env.TenantID}
	if _, err := s.Store.GetDevice(tc, env.DeviceID); err != nil {
		errorJSON(w, http.StatusUnauthorized, "unknown_device", "device identity rejected")
		return env, tenancy.Context{}, err
	}
	return env, tc, nil
}

func (s *Server) deviceContextFromHeaders(r *http.Request) (tenancy.Context, error) {
	tenantID := r.Header.Get("X-Tenant-ID")
	deviceID := r.Header.Get("X-Device-ID")
	if tenantID == "" || deviceID == "" {
		return tenancy.Context{}, errors.New("missing device headers")
	}
	tc := tenancy.Context{TenantID: tenantID}
	_, err := s.Store.GetDevice(tc, deviceID)
	return tc, err
}

func decode(w http.ResponseWriter, r *http.Request, max int64, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(http.MaxBytesReader(w, r.Body, max)).Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func errorJSON(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}

func bearer(header string) string {
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func ContextWithTenant(ctx context.Context, tenantID string) context.Context {
	return tenancy.WithContext(ctx, tenancy.Context{TenantID: tenantID})
}
