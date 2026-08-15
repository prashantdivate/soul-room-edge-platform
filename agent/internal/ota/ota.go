package ota

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

var adapterNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`)

var ErrRebootPending = errors.New("OTA reboot is pending")

type Request struct {
	UpdateID       string   `json:"update_id"`
	Adapter        string   `json:"adapter"`
	Version        string   `json:"version"`
	Product        string   `json:"product"`
	Architecture   string   `json:"architecture"`
	ArtifactURL    string   `json:"artifact_url,omitempty"`
	ArtifactSize   int64    `json:"artifact_size,omitempty"`
	Digest         string   `json:"digest,omitempty"`
	Signature      string   `json:"signature,omitempty"`
	SigningKeyID   string   `json:"signing_key_id,omitempty"`
	CompatibleFrom []string `json:"compatible_from,omitempty"`
	FlatpakRef     string   `json:"flatpak_ref,omitempty"`
	FlatpakRemote  string   `json:"flatpak_remote,omitempty"`
	FlatpakCommit  string   `json:"flatpak_commit,omitempty"`
	RepositoryURL  string   `json:"repository_url,omitempty"`
}

type Status struct {
	UpdateID string `json:"update_id"`
	Adapter  string `json:"adapter"`
	State    string `json:"state"`
	Version  string `json:"version"`
	Error    string `json:"error,omitempty"`
}

type Config struct {
	Enabled            bool
	Product            string
	StateDir           string
	StagingDir         string
	TrustedKeysDir     string
	PluginDir          string
	MaxArtifactBytes   int64
	MinFreeBytes       uint64
	DownloadTimeout    time.Duration
	HealthTimeout      time.Duration
	HealthCheckCommand string
	AutoReboot         bool
}

type Adapter interface {
	Name() string
	ArtifactRequired() bool
	Available() bool
	Check(context.Context, Request, string) error
	Install(context.Context, Request, string) error
	Activate(context.Context, Request) error
	Commit(context.Context, Request) error
	Rollback(context.Context, Request, string) error
	CurrentVersion(context.Context, Request) (string, error)
	RequiresReboot() bool
}

type Runner interface {
	LookPath(string) (string, error)
	Run(context.Context, string, ...string) (string, error)
}

type OSRunner struct{}

func (OSRunner) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (OSRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("%s failed: %w", name, err)
	}
	return strings.TrimSpace(string(output)), nil
}

func CanonicalArchitecture(value string) string {
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

func ValidateRequest(cfg Config, request Request) error {
	if request.UpdateID == "" || request.Version == "" || !adapterNamePattern.MatchString(request.Adapter) {
		return errors.New("OTA request requires an update ID, version, and valid adapter")
	}
	if request.Adapter != "flatpak" {
		if request.ArtifactURL == "" || request.Digest == "" || request.Signature == "" || request.SigningKeyID == "" {
			return errors.New("OS OTA requires an artifact URL, SHA-256 digest, detached signature, and signing key ID")
		}
		if CanonicalArchitecture(request.Architecture) != CanonicalArchitecture(runtime.GOARCH) {
			return fmt.Errorf("artifact architecture %q does not match device architecture %q", request.Architecture, runtime.GOARCH)
		}
		if cfg.Product == "" {
			return errors.New("ota.product must be configured before OS updates are allowed")
		}
		if request.Product != cfg.Product {
			return fmt.Errorf("artifact product %q does not match configured product %q", request.Product, cfg.Product)
		}
	}
	if request.ArtifactSize < 0 || (cfg.MaxArtifactBytes > 0 && request.ArtifactSize > cfg.MaxArtifactBytes) {
		return errors.New("artifact exceeds the configured size limit")
	}
	return nil
}

func DiscoverCapabilities(pluginDir string, runner Runner) []string {
	names := []string{}
	for name, command := range map[string]string{
		"flatpak":  "flatpak",
		"mender":   "mender-update",
		"ostree":   "ostree",
		"rauc":     "rauc",
		"swupdate": "swupdate",
	} {
		if _, err := runner.LookPath(command); err == nil {
			names = append(names, "ota:"+name)
		} else if name == "mender" {
			if _, fallbackErr := runner.LookPath("mender"); fallbackErr == nil {
				names = append(names, "ota:mender")
			}
		}
	}
	entries, _ := os.ReadDir(pluginDir)
	for _, entry := range entries {
		if entry.IsDir() || !adapterNamePattern.MatchString(entry.Name()) {
			continue
		}
		if info, err := entry.Info(); err == nil && info.Mode().Perm()&0111 != 0 {
			names = append(names, "ota:"+entry.Name())
		}
	}
	sort.Strings(names)
	return compact(names)
}

func compact(values []string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
