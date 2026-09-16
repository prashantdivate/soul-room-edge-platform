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

func TestParseUbuntuOSRelease(t *testing.T) {
	release := parseOSRelease(`PRETTY_NAME="Ubuntu 25.04"
NAME="Ubuntu"
VERSION_ID="25.04"
ID=ubuntu`)
	if got := osDisplayName(release); got != "Ubuntu 25.04" {
		t.Fatalf("unexpected display name %q", got)
	}
	if release["ID"] != "ubuntu" || release["VERSION_ID"] != "25.04" {
		t.Fatalf("unexpected Ubuntu release: %#v", release)
	}
}

func TestParseYoctoOSRelease(t *testing.T) {
	release := parseOSRelease(`ID=poky
NAME="Poky (Yocto Project Reference Distro)"
VERSION="5.0.6 (scarthgap)"
VERSION_ID="5.0.6"
BUILD_ID="factory-2026.09"`)
	if got := osDisplayName(release); got != "Poky (Yocto Project Reference Distro) 5.0.6 (scarthgap)" {
		t.Fatalf("unexpected display name %q", got)
	}
	if release["BUILD_ID"] != "factory-2026.09" {
		t.Fatalf("unexpected Yocto build ID: %#v", release)
	}
}

func TestParseARMCPUModel(t *testing.T) {
	if got := parseLSCPUModel("Architecture: aarch64\nModel name: Cortex-A76\n"); got != "Cortex-A76" {
		t.Fatalf("unexpected lscpu model %q", got)
	}
	if got := parseCPUModel("processor: 0\nmodel name: Intel(R) Test CPU\n"); got != "Intel(R) Test CPU" {
		t.Fatalf("unexpected cpuinfo model %q", got)
	}
}
