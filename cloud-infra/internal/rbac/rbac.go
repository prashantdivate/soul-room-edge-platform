package rbac

import (
	"context"
	"errors"

	"github.com/unified-fleet/cloud-infra/internal/tenancy"
)

type Permission string

const (
	DeviceRead        Permission = "device.read"
	DeviceEnroll      Permission = "device.enroll"
	DeviceUpdate      Permission = "device.update"
	DeviceCommand     Permission = "device.command"
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
)

var RolePermissions = map[string][]Permission{
	"platform_administrator":     {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceCommand, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate, TenantSettings},
	"organization_owner":         {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceCommand, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate, TenantSettings},
	"organization_administrator": {DeviceRead, DeviceEnroll, DeviceUpdate, DeviceCommand, DeviceReboot, TelemetryRead, DeploymentCreate, DeploymentApprove, DeploymentCancel, ArtifactUpload, ProfileManage, UserManage, AuditRead, RemoteCreate},
	"fleet_operator":             {DeviceRead, DeviceCommand, TelemetryRead},
	"deployment_manager":         {DeviceRead, TelemetryRead, DeploymentCreate, DeploymentCancel, ArtifactUpload},
	"support_engineer":           {DeviceRead, TelemetryRead, DeviceCommand, RemoteCreate},
	"security_auditor":           {DeviceRead, TelemetryRead, AuditRead},
	"read_only_viewer":           {DeviceRead, TelemetryRead},
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
