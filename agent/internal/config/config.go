package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Version    int
	Server     ServerConfig
	Identity   IdentityConfig
	Storage    StorageConfig
	Telemetry  TelemetryConfig
	Jobs       JobsConfig
	Containers ContainerConfig
	Gateway    GatewayConfig
	Location   LocationConfig
	OTA        OTAConfig
}

type ServerConfig struct {
	Endpoint       string
	CAFile         string
	ConnectTimeout time.Duration
}

type IdentityConfig struct {
	StateDir string
	UseTPM   bool
}

type StorageConfig struct {
	StateDir      string
	MaxQueueBytes int64
	MaxQueueAge   time.Duration
}

type TelemetryConfig struct {
	Interval   time.Duration
	Jitter     time.Duration
	Collectors map[string]bool
}

type JobsConfig struct {
	MaxConcurrent  int
	DefaultTimeout time.Duration
	AllowShell     bool
}

type ContainerConfig struct {
	Provider string
	Socket   string
}

type GatewayConfig struct {
	Enabled      bool
	ConnectorDir string
}

type LocationConfig struct {
	Source      string
	Latitude    float64
	Longitude   float64
	Label       string
	GPSDAddress string
	IPURL       string
}

type OTAConfig struct {
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

func Default() Config {
	return Config{
		Version: 1,
		Server: ServerConfig{
			Endpoint:       "https://127.0.0.1:9443",
			ConnectTimeout: 15 * time.Second,
		},
		Identity: IdentityConfig{
			StateDir: "/var/lib/edge-agent/identity",
		},
		Storage: StorageConfig{
			StateDir:      "/var/lib/edge-agent",
			MaxQueueBytes: 100 * 1024 * 1024,
			MaxQueueAge:   168 * time.Hour,
		},
		Telemetry: TelemetryConfig{
			Interval: 60 * time.Second,
			Jitter:   10 * time.Second,
			Collectors: map[string]bool{
				"cpu": true, "memory": true, "disk": true, "network": true,
				"uptime": true, "os": true, "processes": false,
			},
		},
		Jobs: JobsConfig{
			MaxConcurrent:  2,
			DefaultTimeout: 10 * time.Minute,
			AllowShell:     false,
		},
		Containers: ContainerConfig{
			Provider: "docker",
			Socket:   "unix:///var/run/docker.sock",
		},
		Gateway: GatewayConfig{
			Enabled:      true,
			ConnectorDir: "/etc/edge-agent/connectors.d",
		},
		Location: LocationConfig{Source: "disabled", GPSDAddress: "127.0.0.1:2947", IPURL: "https://ipwho.is/"},
		OTA: OTAConfig{
			Enabled:          true,
			StateDir:         "/var/lib/edge-agent/ota",
			StagingDir:       "/var/lib/edge-agent/ota/staging",
			TrustedKeysDir:   "/etc/edge-agent/trusted-update-keys",
			PluginDir:        "/usr/libexec/edge-agent/ota",
			MaxArtifactBytes: 4 * 1024 * 1024 * 1024,
			MinFreeBytes:     256 * 1024 * 1024,
			DownloadTimeout:  2 * time.Hour,
			HealthTimeout:    2 * time.Minute,
			AutoReboot:       false,
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	var section string
	var subsection string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		if strings.HasSuffix(line, ":") {
			if indent == 0 {
				section = strings.TrimSuffix(line, ":")
				subsection = ""
			} else {
				subsection = strings.TrimSuffix(line, ":")
			}
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return cfg, fmt.Errorf("invalid config line %q", raw)
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if subsection != "" {
			key = subsection + "." + key
		}
		if err := setValue(&cfg, section, key, value); err != nil {
			return cfg, err
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, err
	}
	return cfg, cfg.Validate()
}

func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	mode := os.FileMode(0640)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".edge-agent-config-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(render(cfg)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func render(cfg Config) string {
	var out strings.Builder
	fmt.Fprintf(&out, "version: %d\n", cfg.Version)
	fmt.Fprintf(&out, "server:\n  endpoint: %s\n  ca_file: %s\n  connect_timeout: %s\n", cfg.Server.Endpoint, cfg.Server.CAFile, cfg.Server.ConnectTimeout)
	fmt.Fprintf(&out, "identity:\n  state_dir: %s\n  use_tpm: %t\n", cfg.Identity.StateDir, cfg.Identity.UseTPM)
	fmt.Fprintf(&out, "storage:\n  state_dir: %s\n  max_queue_bytes: %d\n  max_queue_age: %s\n", cfg.Storage.StateDir, cfg.Storage.MaxQueueBytes, cfg.Storage.MaxQueueAge)
	fmt.Fprintf(&out, "telemetry:\n  interval: %s\n  jitter: %s\n  collectors:\n", cfg.Telemetry.Interval, cfg.Telemetry.Jitter)
	collectorNames := make([]string, 0, len(cfg.Telemetry.Collectors))
	for name := range cfg.Telemetry.Collectors {
		collectorNames = append(collectorNames, name)
	}
	sort.Strings(collectorNames)
	for _, name := range collectorNames {
		fmt.Fprintf(&out, "    %s: %t\n", name, cfg.Telemetry.Collectors[name])
	}
	fmt.Fprintf(&out, "jobs:\n  max_concurrent: %d\n  default_timeout: %s\n  allow_shell: %t\n", cfg.Jobs.MaxConcurrent, cfg.Jobs.DefaultTimeout, cfg.Jobs.AllowShell)
	fmt.Fprintf(&out, "containers:\n  provider: %s\n  socket: %s\n", cfg.Containers.Provider, cfg.Containers.Socket)
	fmt.Fprintf(&out, "gateway:\n  enabled: %t\n  connector_dir: %s\n", cfg.Gateway.Enabled, cfg.Gateway.ConnectorDir)
	fmt.Fprintf(&out, "location:\n  source: %s\n  latitude: %g\n  longitude: %g\n  label: %s\n  gpsd_address: %s\n  ip_url: %s\n", cfg.Location.Source, cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Label, cfg.Location.GPSDAddress, cfg.Location.IPURL)
	fmt.Fprintf(&out, "ota:\n  enabled: %t\n  product: %s\n  state_dir: %s\n  staging_dir: %s\n", cfg.OTA.Enabled, cfg.OTA.Product, cfg.OTA.StateDir, cfg.OTA.StagingDir)
	fmt.Fprintf(&out, "  trusted_keys_dir: %s\n  plugin_dir: %s\n  max_artifact_bytes: %d\n", cfg.OTA.TrustedKeysDir, cfg.OTA.PluginDir, cfg.OTA.MaxArtifactBytes)
	fmt.Fprintf(&out, "  min_free_bytes: %d\n  download_timeout: %s\n  health_timeout: %s\n", cfg.OTA.MinFreeBytes, cfg.OTA.DownloadTimeout, cfg.OTA.HealthTimeout)
	fmt.Fprintf(&out, "  health_check_command: %s\n  auto_reboot: %t\n", cfg.OTA.HealthCheckCommand, cfg.OTA.AutoReboot)
	return out.String()
}

func setValue(cfg *Config, section, key, value string) error {
	switch section {
	case "":
		if key == "version" {
			v, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			cfg.Version = v
		}
	case "server":
		switch key {
		case "endpoint":
			cfg.Server.Endpoint = value
		case "ca_file":
			cfg.Server.CAFile = value
		case "connect_timeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Server.ConnectTimeout = d
		}
	case "identity":
		switch key {
		case "state_dir":
			cfg.Identity.StateDir = value
		case "use_tpm":
			cfg.Identity.UseTPM = parseBool(value)
		}
	case "storage":
		switch key {
		case "state_dir":
			cfg.Storage.StateDir = value
		case "max_queue_bytes":
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			cfg.Storage.MaxQueueBytes = v
		case "max_queue_age":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Storage.MaxQueueAge = d
		}
	case "telemetry":
		switch {
		case key == "interval":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Telemetry.Interval = d
		case key == "jitter":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Telemetry.Jitter = d
		case strings.HasPrefix(key, "collectors."):
			if cfg.Telemetry.Collectors == nil {
				cfg.Telemetry.Collectors = map[string]bool{}
			}
			cfg.Telemetry.Collectors[strings.TrimPrefix(key, "collectors.")] = parseBool(value)
		}
	case "jobs":
		switch key {
		case "max_concurrent":
			v, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			cfg.Jobs.MaxConcurrent = v
		case "default_timeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.Jobs.DefaultTimeout = d
		case "allow_shell":
			cfg.Jobs.AllowShell = parseBool(value)
		}
	case "containers":
		if key == "provider" {
			cfg.Containers.Provider = value
		} else if key == "socket" {
			cfg.Containers.Socket = value
		}
	case "gateway":
		if key == "enabled" {
			cfg.Gateway.Enabled = parseBool(value)
		} else if key == "connector_dir" {
			cfg.Gateway.ConnectorDir = value
		}
	case "location":
		switch key {
		case "source":
			cfg.Location.Source = strings.ToLower(value)
		case "latitude":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			cfg.Location.Latitude = v
		case "longitude":
			v, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			cfg.Location.Longitude = v
		case "label":
			cfg.Location.Label = value
		case "gpsd_address":
			cfg.Location.GPSDAddress = value
		case "ip_url":
			cfg.Location.IPURL = value
		}
	case "ota":
		switch key {
		case "enabled":
			cfg.OTA.Enabled = parseBool(value)
		case "product":
			cfg.OTA.Product = value
		case "state_dir":
			cfg.OTA.StateDir = value
		case "staging_dir":
			cfg.OTA.StagingDir = value
		case "trusted_keys_dir":
			cfg.OTA.TrustedKeysDir = value
		case "plugin_dir":
			cfg.OTA.PluginDir = value
		case "max_artifact_bytes":
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			cfg.OTA.MaxArtifactBytes = v
		case "min_free_bytes":
			v, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			cfg.OTA.MinFreeBytes = v
		case "download_timeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.OTA.DownloadTimeout = d
		case "health_timeout":
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			cfg.OTA.HealthTimeout = d
		case "health_check_command":
			cfg.OTA.HealthCheckCommand = value
		case "auto_reboot":
			cfg.OTA.AutoReboot = parseBool(value)
		}
	}
	return nil
}

func parseBool(s string) bool {
	return strings.EqualFold(s, "true") || s == "1" || strings.EqualFold(s, "yes")
}

func (c Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version %d", c.Version)
	}
	if !strings.HasPrefix(c.Server.Endpoint, "https://") {
		return fmt.Errorf("server.endpoint must use https")
	}
	if c.Identity.StateDir == "" {
		return fmt.Errorf("identity.state_dir is required")
	}
	if c.Storage.StateDir == "" {
		return fmt.Errorf("storage.state_dir is required")
	}
	if c.Storage.MaxQueueBytes < 1024*1024 {
		return fmt.Errorf("storage.max_queue_bytes must be at least 1 MiB")
	}
	if c.Telemetry.Interval < time.Second {
		return fmt.Errorf("telemetry.interval must be at least 1s")
	}
	if c.Jobs.MaxConcurrent < 1 {
		return fmt.Errorf("jobs.max_concurrent must be positive")
	}
	if c.Jobs.AllowShell {
		return fmt.Errorf("jobs.allow_shell must remain false for production handler set")
	}
	if c.Location.Source != "disabled" && c.Location.Source != "static" && c.Location.Source != "gpsd" && c.Location.Source != "ip" {
		return fmt.Errorf("location.source must be disabled, static, gpsd, or ip")
	}
	if c.Location.Source == "static" && (c.Location.Latitude < -90 || c.Location.Latitude > 90 || c.Location.Longitude < -180 || c.Location.Longitude > 180) {
		return fmt.Errorf("static location coordinates are invalid")
	}
	if c.Location.Source == "ip" && !strings.HasPrefix(c.Location.IPURL, "https://") {
		return fmt.Errorf("location.ip_url must use https")
	}
	if c.OTA.Enabled {
		if c.OTA.StateDir == "" || c.OTA.StagingDir == "" || c.OTA.TrustedKeysDir == "" || c.OTA.PluginDir == "" {
			return fmt.Errorf("ota directories are required when OTA is enabled")
		}
		if c.OTA.MaxArtifactBytes < 1024*1024 {
			return fmt.Errorf("ota.max_artifact_bytes must be at least 1 MiB")
		}
		if c.OTA.DownloadTimeout < time.Minute || c.OTA.HealthTimeout < time.Second {
			return fmt.Errorf("OTA download and health timeouts are too short")
		}
	}
	return nil
}
