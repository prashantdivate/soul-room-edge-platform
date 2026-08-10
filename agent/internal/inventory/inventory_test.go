package inventory

import (
	"runtime"
	"testing"

	"github.com/soul-room/edge-agent/internal/identity"
)

func TestCollectReadsLinuxInventory(t *testing.T) {
	got := Collect(identity.Identity{DeviceID: "device-test"}, "test-version")
	if got.DeviceID != "device-test" || got.AgentVersion != "test-version" {
		t.Fatalf("identity fields not preserved: %+v", got)
	}
	if got.Hostname == "" || got.Architecture != runtime.GOARCH || got.OS == "" || got.Kernel == "" {
		t.Fatalf("missing host inventory: %+v", got)
	}
	if got.CPUCores < 1 || got.MemoryTotalBytes == 0 || got.StorageTotalBytes == 0 {
		t.Fatalf("missing hardware capacity: %+v", got)
	}
}

func TestParsePackages(t *testing.T) {
	packages := parsePackages("openssl\t3.0.2\tarm64\ncurl\t8.0\tarm64\n", "dpkg")
	if len(packages) != 2 || packages[0].Name != "openssl" || packages[1].Architecture != "arm64" {
		t.Fatalf("unexpected packages: %#v", packages)
	}
}
