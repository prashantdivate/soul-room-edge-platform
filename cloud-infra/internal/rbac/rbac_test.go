package rbac

import (
	"context"
	"testing"

	"github.com/soul-room/cloud-infra/internal/tenancy"
)

func TestAuthorizeRole(t *testing.T) {
	ctx := tenancy.WithContext(context.Background(), tenancy.Context{TenantID: "t1", ActorID: "u1", Roles: []string{"read_only_viewer"}})
	if err := Authorize(ctx, DeviceRead); err != nil {
		t.Fatal(err)
	}
	if err := Authorize(ctx, DeploymentCreate); err == nil {
		t.Fatal("expected deployment create to be forbidden")
	}
}
