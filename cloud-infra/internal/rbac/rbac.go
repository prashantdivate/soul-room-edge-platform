package rbac

import (
	"context"
	"errors"

	"github.com/soul-room/cloud-infra/internal/tenancy"
)

type Permission string

const (
	DeviceRead        Permission = "device.read"
	DeviceEnroll      Permission = "device.enroll"
	DeviceUpdate      Permission = "device.update"
	DeviceDelete      Permission = "device.delete"
	DeviceCommand     Permission = "device.command"
	JobDelete         Permission = "job.delete"
	DeviceReboot      Permission = "device.reboot"
	TelemetryRead     Permission = "telemetry.read"
	DeploymentCreate  Permission = "deployment.create"
	DeploymentApprove Permission = "deployment.approve"
	DeploymentCancel  Permission = "deployment.cancel"
	ArtifactUpload    Permission = "artifact.upload"
	ProfileManage     Permission = "profile.manage"
	UserManage        Permission = "user.manage"
	AuditRead         Permission = "audit.read"
	RemoteCreate      Permission = "remote_access.create"
	TenantSettings    Permission = "tenant.settings.manage"
	AIUse             Permission = "ai.use"
)

var RolePermissions = map[string][]Permission{
	"platform_administrator":     {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceDelete, DeviceCommand, JobDelete, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate, TenantSettings, AIUse},
	"organization_owner":         {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceDelete, DeviceCommand, JobDelete, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate, TenantSettings, AIUse},
	"organization_administrator": {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceDelete, DeviceCommand, JobDelete, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate, AIUse},
	"fleet_operator":             {DeviceRead, DeviceCommand, TelemetryRead, AIUse},
	"deployment_manager":         {DeviceRead, TelemetryRead, DeploymentCreate, DeploymentCancel, ArtifactUpload, AIUse},
	"support_engineer":           {DeviceRead, TelemetryRead, DeviceCommand, RemoteCreate, AIUse},
	"security_auditor":           {DeviceRead, TelemetryRead, AuditRead, AIUse},
	"read_only_viewer":           {DeviceRead, TelemetryRead, AIUse},
}

func Authorize(ctx context.Context, permission Permission) error {
	tc, err := tenancy.Require(ctx)
	if err != nil {
		return err
	}
	for _, scope := range tc.Scopes {
		if scope == string(permission) {
			return nil
		}
	}
	for _, role := range tc.Roles {
		for _, p := range RolePermissions[role] {
			if p == permission {
				return nil
			}
		}
	}
	return ErrForbidden
}

var ErrForbidden = errors.New("forbidden")
