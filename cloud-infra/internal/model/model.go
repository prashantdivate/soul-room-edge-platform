package model

import (
	"encoding/json"
	"time"
)

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type PlatformSettings struct {
	OrganizationName          string    `json:"organization_name"`
	CompanyDomain             string    `json:"company_domain,omitempty"`
	ShellHubURL               string    `json:"shellhub_url,omitempty"`
	ShellHubSSHPort           int       `json:"shellhub_ssh_port"`
	DeviceOfflineMinutes      int       `json:"device_offline_minutes"`
	DefaultTelemetryWindow    string    `json:"default_telemetry_window"`
	DefaultOTAPilotPercent    int       `json:"default_ota_pilot_percent"`
	DefaultEnrollmentTTLHours int       `json:"default_enrollment_ttl_hours"`
	DefaultJobTTLMinutes      int       `json:"default_job_ttl_minutes"`
	UpdatedBy                 string    `json:"updated_by,omitempty"`
	UpdatedAt                 time.Time `json:"updated_at,omitempty"`
}

type User struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	Role             string    `json:"role,omitempty"`
	PasswordHash     string    `json:"password_hash,omitempty"`
	ActivatedAt      time.Time `json:"activated_at,omitempty"`
	MFAEnabled       bool      `json:"mfa_enabled"`
	LockedUntil      time.Time `json:"locked_until,omitempty"`
	FailedLoginCount int       `json:"failed_login_count"`
	CreatedAt        time.Time `json:"created_at"`
}

type Membership struct {
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
}

type EnrollmentToken struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	ProfileID       string    `json:"profile_id,omitempty"`
	TokenHash       string    `json:"token_hash,omitempty"`
	PlainToken      string    `json:"token,omitempty"`
	ExpectedSerial  string    `json:"expected_serial,omitempty"`
	ExpectedProduct string    `json:"expected_product,omitempty"`
	MaxDevices      int       `json:"max_devices"`
	UsedCount       int       `json:"used_count"`
	ExpiresAt       time.Time `json:"expires_at"`
	RevokedAt       time.Time `json:"revoked_at,omitempty"`
	CreatedBy       string    `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type Device struct {
	ID                   string          `json:"id"`
	TenantID             string          `json:"tenant_id"`
	ProfileID            string          `json:"profile_id,omitempty"`
	ParentGatewayID      string          `json:"parent_gateway_id,omitempty"`
	DisplayName          string          `json:"display_name"`
	EnrollmentStatus     string          `json:"enrollment_status"`
	CertificateSubject   string          `json:"certificate_subject,omitempty"`
	CertificateSerial    string          `json:"certificate_serial,omitempty"`
	CertificateExpiresAt time.Time       `json:"certificate_expires_at,omitempty"`
	LastSeenAt           time.Time       `json:"last_seen_at,omitempty"`
	Presence             string          `json:"presence"`
	AgentVersion         string          `json:"agent_version,omitempty"`
	Architecture         string          `json:"architecture,omitempty"`
	OS                   string          `json:"os,omitempty"`
	Kernel               string          `json:"kernel,omitempty"`
	HardwareModel        string          `json:"hardware_model,omitempty"`
	Serial               string          `json:"serial,omitempty"`
	Capabilities         []string        `json:"capabilities,omitempty"`
	Tags                 []string        `json:"tags,omitempty"`
	Location             *DeviceLocation `json:"location,omitempty"`
	RemoteAccessID       string          `json:"remote_access_id,omitempty"`
	SoftwareState        json.RawMessage `json:"software_state,omitempty"`
	UpdateState          string          `json:"update_state"`
	HealthState          string          `json:"health_state"`
	CreatedAt            time.Time       `json:"created_at"`
}

type DeviceLocation struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	AccuracyM float64   `json:"accuracy_m,omitempty"`
	Source    string    `json:"source"`
	Label     string    `json:"label,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DownstreamDevice struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	ParentGatewayID  string    `json:"parent_gateway_id"`
	ConnectorType    string    `json:"connector_type"`
	ProtocolAddress  string    `json:"protocol_address"`
	Model            string    `json:"model,omitempty"`
	Serial           string    `json:"serial,omitempty"`
	FirmwareVersion  string    `json:"firmware_version,omitempty"`
	LastSeenAt       time.Time `json:"last_seen_at,omitempty"`
	Health           string    `json:"health"`
	Capabilities     []string  `json:"capabilities,omitempty"`
	ApprovedCommands []string  `json:"approved_commands,omitempty"`
}

type Metric struct {
	DeviceID   string            `json:"device_id"`
	Name       string            `json:"name"`
	Value      float64           `json:"value"`
	Unit       string            `json:"unit"`
	Labels     map[string]string `json:"labels,omitempty"`
	DeviceTime time.Time         `json:"device_time"`
	ReceivedAt time.Time         `json:"received_at"`
	MessageID  string            `json:"message_id"`
}

type Job struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"tenant_id"`
	DeviceID       string          `json:"device_id"`
	DeploymentID   string          `json:"deployment_id,omitempty"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	PayloadDigest  string          `json:"payload_digest"`
	CreatedBy      string          `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	ApprovalStatus string          `json:"approval_status"`
	Attempt        int             `json:"attempt"`
	State          string          `json:"state"`
	Result         json.RawMessage `json:"result,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
}

type Artifact struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	ObjectKey   string    `json:"object_key"`
	Digest      string    `json:"digest"`
	Signature   string    `json:"signature,omitempty"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	CreatedAt   time.Time `json:"created_at"`
}

type AuditEvent struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id,omitempty"`
	ActorID       string          `json:"actor_id,omitempty"`
	Action        string          `json:"action"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    string          `json:"resource_id,omitempty"`
	Result        string          `json:"result"`
	SourceIP      string          `json:"source_ip,omitempty"`
	UserAgent     string          `json:"user_agent,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Summary       json.RawMessage `json:"summary,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}
