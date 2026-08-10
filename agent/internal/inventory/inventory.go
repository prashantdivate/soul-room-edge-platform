package inventory

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/soul-room/edge-agent/internal/identity"
)

type InstalledPackage struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Architecture string `json:"architecture,omitempty"`
	Source       string `json:"source"`
}

type Inventory struct {
	DeviceID          string             `json:"device_id"`
	Hostname          string             `json:"hostname"`
	Architecture      string             `json:"architecture"`
	OS                string             `json:"os"`
	Kernel            string             `json:"kernel"`
	OSRelease         map[string]string  `json:"os_release"`
	HardwareModel     string             `json:"hardware_model"`
	Serial            string             `json:"serial"`
	CPUModel          string             `json:"cpu_model"`
	CPUCores          int                `json:"cpu_cores"`
	MemoryTotalBytes  uint64             `json:"memory_total_bytes"`
	StorageTotalBytes uint64             `json:"storage_total_bytes"`
	AgentVersion      string             `json:"agent_version"`
	Capabilities      []string           `json:"capabilities"`
	DownstreamIDs     []string           `json:"downstream_ids,omitempty"`
	InstalledPackages []InstalledPackage `json:"installed_packages,omitempty"`
}

func Collect(id identity.Identity, version string) Inventory {
	hostname, _ := os.Hostname()
	release := osRelease()
	return Inventory{
		DeviceID:          id.DeviceID,
		Hostname:          hostname,
		Architecture:      runtime.GOARCH,
		OS:                first(release["PRETTY_NAME"], release["NAME"], runtime.GOOS),
		Kernel:            readTrim("/proc/sys/kernel/osrelease"),
		OSRelease:         release,
		HardwareModel:     hardwareModel(),
		Serial:            serialNumber(),
		CPUModel:          cpuModel(),
		CPUCores:          runtime.NumCPU(),
		MemoryTotalBytes:  memoryTotal(),
		StorageTotalBytes: storageTotal("/"),
		AgentVersion:      version,
		Capabilities:      capabilities(),
		InstalledPackages: installedPackages(),
	}
}

func capabilities() []string {
	values := []string{"telemetry", "inventory", "jobs"}
	if _, err := exec.LookPath("flatpak"); err == nil {
		values = append(values, "ota:flatpak")
	}
	if _, err := exec.LookPath("trivy"); err == nil {
		values = append(values, "security:trivy")
	}
	return values
}

func installedPackages() []InstalledPackage {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	commands := []struct {
		name   string
		args   []string
		source string
	}{
		{"dpkg-query", []string{"-W", "-f=${Package}\t${Version}\t${Architecture}\n"}, "dpkg"},
		{"rpm", []string{"-qa", "--qf", "%{NAME}\t%{VERSION}-%{RELEASE}\t%{ARCH}\n"}, "rpm"},
	}
	for _, command := range commands {
		if _, err := exec.LookPath(command.name); err != nil {
			continue
		}
		output, err := exec.CommandContext(ctx, command.name, command.args...).Output()
		if err == nil {
			return parsePackages(string(output), command.source)
		}
	}
	return nil
}

func parsePackages(output, source string) []InstalledPackage {
	packages := make([]InstalledPackage, 0)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || strings.TrimSpace(fields[0]) == "" {
			continue
		}
		pkg := InstalledPackage{Name: bounded(fields[0], 256), Version: bounded(fields[1], 256), Source: source}
		if len(fields) > 2 {
			pkg.Architecture = bounded(fields[2], 64)
		}
		packages = append(packages, pkg)
		if len(packages) == 4000 {
			break
		}
	}
	return packages
}

func bounded(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) > limit {
		return value[:limit]
	}
	return value
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	return strings.Trim(strings.TrimSpace(string(b)), "\x00")
}

func osRelease() map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			out[key] = strings.Trim(value, `"`)
		}
	}
	return out
}

func hardwareModel() string {
	for _, path := range []string{"/proc/device-tree/model", "/sys/firmware/devicetree/base/model", "/sys/devices/virtual/dmi/id/product_name"} {
		if value := readTrim(path); value != "unknown" && value != "" {
			return value
		}
	}
	return "unknown"
}

func serialNumber() string {
	for _, path := range []string{"/sys/firmware/devicetree/base/serial-number", "/sys/devices/virtual/dmi/id/product_serial"} {
		if value := readTrim(path); value != "unknown" && value != "" {
			return value
		}
	}
	b, _ := os.ReadFile("/proc/cpuinfo")
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && strings.EqualFold(strings.TrimSpace(key), "serial") {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}

func cpuModel() string {
	b, _ := os.ReadFile("/proc/cpuinfo")
	for _, line := range strings.Split(string(b), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && (strings.EqualFold(strings.TrimSpace(key), "model name") || strings.EqualFold(strings.TrimSpace(key), "hardware")) {
			return strings.TrimSpace(value)
		}
	}
	return "unknown"
}

func memoryTotal() uint64 {
	b, _ := os.ReadFile("/proc/meminfo")
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value * 1024
		}
	}
	return 0
}

func storageTotal(path string) uint64 {
	var stat syscall.Statfs_t
	if syscall.Statfs(path, &stat) != nil {
		return 0
	}
	return stat.Blocks * uint64(stat.Bsize)
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unknown"
}
