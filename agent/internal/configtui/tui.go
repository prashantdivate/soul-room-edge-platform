package configtui

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/soul-room/edge-agent/internal/config"
)

type editor struct {
	in  *bufio.Scanner
	out io.Writer
}

func Run(path string, cfg config.Config, in io.Reader, out io.Writer) error {
	e := editor{in: bufio.NewScanner(in), out: out}
	for {
		e.header(path, cfg)
		choice, ok := e.read("Select a section, or 0 to save and exit")
		if !ok || choice == "0" || strings.EqualFold(choice, "q") {
			if err := config.Save(path, cfg); err != nil {
				fmt.Fprintf(out, "\nCannot save configuration: %v\n", err)
				if !ok {
					return err
				}
				_, _ = e.read("Press Enter to continue")
				continue
			}
			fmt.Fprintf(out, "\nConfiguration saved to %s\nRestart edge-agent to apply it.\n", path)
			return nil
		}
		switch choice {
		case "1":
			e.server(&cfg)
		case "2":
			e.identityStorage(&cfg)
		case "3":
			e.telemetry(&cfg)
		case "4":
			e.jobsContainers(&cfg)
		case "5":
			e.gatewayLocation(&cfg)
		case "6":
			e.ota(&cfg)
		default:
			fmt.Fprintln(out, "Choose a number from 0 to 6.")
		}
	}
}

func (e editor) header(path string, cfg config.Config) {
	fmt.Fprint(e.out, "\033[2J\033[H")
	fmt.Fprintln(e.out, "Soul Room Edge Agent Configuration")
	fmt.Fprintf(e.out, "File: %s\nEndpoint: %s\n\n", path, cfg.Server.Endpoint)
	fmt.Fprintln(e.out, "  1  Server and trust")
	fmt.Fprintln(e.out, "  2  Identity and storage")
	fmt.Fprintln(e.out, "  3  Telemetry")
	fmt.Fprintln(e.out, "  4  Jobs and containers")
	fmt.Fprintln(e.out, "  5  Gateway and location")
	fmt.Fprintln(e.out, "  6  OTA updates")
	fmt.Fprintln(e.out, "  0  Save and exit")
}

func (e editor) server(cfg *config.Config) {
	cfg.Server.Endpoint = e.text("Server endpoint", cfg.Server.Endpoint)
	cfg.Server.CAFile = e.text("CA certificate file", cfg.Server.CAFile)
	cfg.Server.ConnectTimeout = e.duration("Connection timeout", cfg.Server.ConnectTimeout)
}

func (e editor) identityStorage(cfg *config.Config) {
	cfg.Identity.StateDir = e.text("Identity directory", cfg.Identity.StateDir)
	cfg.Identity.UseTPM = e.boolean("Use TPM", cfg.Identity.UseTPM)
	cfg.Storage.StateDir = e.text("State directory", cfg.Storage.StateDir)
	cfg.Storage.MaxQueueBytes = e.int64("Maximum offline queue bytes", cfg.Storage.MaxQueueBytes)
	cfg.Storage.MaxQueueAge = e.duration("Maximum offline queue age", cfg.Storage.MaxQueueAge)
}

func (e editor) telemetry(cfg *config.Config) {
	cfg.Telemetry.Interval = e.duration("Collection interval", cfg.Telemetry.Interval)
	cfg.Telemetry.Jitter = e.duration("Collection jitter", cfg.Telemetry.Jitter)
	for _, name := range []string{"cpu", "memory", "disk", "network", "uptime", "os", "processes"} {
		cfg.Telemetry.Collectors[name] = e.boolean("Collect "+name, cfg.Telemetry.Collectors[name])
	}
}

func (e editor) jobsContainers(cfg *config.Config) {
	cfg.Jobs.MaxConcurrent = e.integer("Maximum concurrent jobs", cfg.Jobs.MaxConcurrent)
	cfg.Jobs.DefaultTimeout = e.duration("Default job timeout", cfg.Jobs.DefaultTimeout)
	cfg.Jobs.AllowShell = e.boolean("Allow shell jobs (must remain false)", cfg.Jobs.AllowShell)
	cfg.Containers.Provider = e.text("Container provider", cfg.Containers.Provider)
	cfg.Containers.Socket = e.text("Container socket", cfg.Containers.Socket)
}

func (e editor) gatewayLocation(cfg *config.Config) {
	cfg.Gateway.Enabled = e.boolean("Enable downstream gateway", cfg.Gateway.Enabled)
	cfg.Gateway.ConnectorDir = e.text("Connector directory", cfg.Gateway.ConnectorDir)
	cfg.Location.Source = strings.ToLower(e.text("Location source (disabled, static, gpsd, ip)", cfg.Location.Source))
	cfg.Location.Latitude = e.float("Static latitude", cfg.Location.Latitude)
	cfg.Location.Longitude = e.float("Static longitude", cfg.Location.Longitude)
	cfg.Location.Label = e.text("Location label", cfg.Location.Label)
	cfg.Location.GPSDAddress = e.text("gpsd address", cfg.Location.GPSDAddress)
	cfg.Location.IPURL = e.text("IP geolocation URL (approximate)", cfg.Location.IPURL)
}

func (e editor) ota(cfg *config.Config) {
	cfg.OTA.Enabled = e.boolean("Enable OTA", cfg.OTA.Enabled)
	cfg.OTA.Product = e.text("Product compatibility name", cfg.OTA.Product)
	cfg.OTA.StateDir = e.text("OTA state directory", cfg.OTA.StateDir)
	cfg.OTA.StagingDir = e.text("OTA staging directory", cfg.OTA.StagingDir)
	cfg.OTA.TrustedKeysDir = e.text("Trusted update keys directory", cfg.OTA.TrustedKeysDir)
	cfg.OTA.PluginDir = e.text("OTA plugin directory", cfg.OTA.PluginDir)
	cfg.OTA.MaxArtifactBytes = e.int64("Maximum artifact bytes", cfg.OTA.MaxArtifactBytes)
	cfg.OTA.MinFreeBytes = e.uint64("Minimum free bytes", cfg.OTA.MinFreeBytes)
	cfg.OTA.DownloadTimeout = e.duration("Download timeout", cfg.OTA.DownloadTimeout)
	cfg.OTA.HealthTimeout = e.duration("Health check timeout", cfg.OTA.HealthTimeout)
	cfg.OTA.HealthCheckCommand = e.text("Health check command", cfg.OTA.HealthCheckCommand)
	cfg.OTA.AutoReboot = e.boolean("Automatically reboot when required", cfg.OTA.AutoReboot)
}

func (e editor) read(label string) (string, bool) {
	fmt.Fprintf(e.out, "%s: ", label)
	if !e.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(e.in.Text()), true
}

func (e editor) text(label, current string) string {
	value, ok := e.read(fmt.Sprintf("%s [%s]", label, current))
	if !ok || value == "" {
		return current
	}
	if value == "-" {
		return ""
	}
	return value
}

func (e editor) boolean(label string, current bool) bool {
	for {
		value := e.text(label+" (true/false)", strconv.FormatBool(current))
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Enter true or false.")
	}
}

func (e editor) duration(label string, current time.Duration) time.Duration {
	for {
		value := e.text(label, current.String())
		parsed, err := time.ParseDuration(value)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Use a duration such as 30s, 10m, or 2h.")
	}
}

func (e editor) integer(label string, current int) int {
	for {
		value := e.text(label, strconv.Itoa(current))
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Enter a whole number.")
	}
}

func (e editor) int64(label string, current int64) int64 {
	for {
		value := e.text(label, strconv.FormatInt(current, 10))
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Enter a whole number.")
	}
}

func (e editor) uint64(label string, current uint64) uint64 {
	for {
		value := e.text(label, strconv.FormatUint(current, 10))
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Enter a positive whole number.")
	}
}

func (e editor) float(label string, current float64) float64 {
	for {
		value := e.text(label, strconv.FormatFloat(current, 'f', -1, 64))
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
		fmt.Fprintln(e.out, "Enter a number.")
	}
}
