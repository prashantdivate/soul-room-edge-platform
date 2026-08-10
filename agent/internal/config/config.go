package config

import (
	"bufio"
	"fmt"
	"os"
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
		Location: LocationConfig{Source: "disabled", GPSDAddress: "127.0.0.1:2947"},
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
	if c.Location.Source != "disabled" && c.Location.Source != "static" && c.Location.Source != "gpsd" {
		return fmt.Errorf("location.source must be disabled, static, or gpsd")
	}
	if c.Location.Source == "static" && (c.Location.Latitude < -90 || c.Location.Latitude > 90 || c.Location.Longitude < -180 || c.Location.Longitude > 180) {
		return fmt.Errorf("static location coordinates are invalid")
	}
	return nil
}
