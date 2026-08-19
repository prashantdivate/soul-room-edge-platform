package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                  string
	DeviceAddr                string
	DevicePublicHost          string
	StorePath                 string
	DevCADir                  string
	OrganizationName          string
	OrganizationSlug          string
	AdminEmail                string
	AdminPassword             string
	SessionTTL                time.Duration
	CertificateTTL            time.Duration
	SignupEnabled             bool
	MaxPayloadBytes           int64
	ShellHubURL               string
	ShellHubSSHPort           int
	ShellHubManaged           bool
	CompanyDomain             string
	DeviceOfflineMinutes      int
	DefaultTelemetryWindow    string
	DefaultOTAPilotPercent    int
	DefaultEnrollmentTTLHours int
	DefaultJobTTLMinutes      int
}

func Load() Config {
	return Config{
		HTTPAddr:                  getenv("SOULROOM_HTTP_ADDR", "127.0.0.1:8080"),
		DeviceAddr:                getenv("SOULROOM_DEVICE_ADDR", "127.0.0.1:8443"),
		DevicePublicHost:          getenv("SOULROOM_DEVICE_PUBLIC_HOST", "localhost"),
		StorePath:                 getenv("SOULROOM_STORE_PATH", ".soul-room/control-plane.json"),
		DevCADir:                  getenv("SOULROOM_DEV_CA_DIR", ".soul-room/ca"),
		OrganizationName:          getenv("SOULROOM_ORGANIZATION_NAME", "Soul Room Local"),
		OrganizationSlug:          getenv("SOULROOM_ORGANIZATION_SLUG", "local"),
		AdminEmail:                getenv("SOULROOM_ADMIN_EMAIL", "admin@soulroom.local"),
		AdminPassword:             getenv("SOULROOM_ADMIN_PASSWORD", "change-me-local"),
		SessionTTL:                duration("SOULROOM_SESSION_TTL", 12*time.Hour),
		CertificateTTL:            duration("SOULROOM_CERTIFICATE_TTL", 90*24*time.Hour),
		SignupEnabled:             getenv("SOULROOM_SIGNUP_ENABLED", "false") == "true",
		MaxPayloadBytes:           int64(integer("SOULROOM_MAX_PAYLOAD_BYTES", 2*1024*1024)),
		ShellHubURL:               getenv("SOULROOM_SHELLHUB_URL", ""),
		ShellHubSSHPort:           integer("SOULROOM_SHELLHUB_SSH_PORT", 22),
		ShellHubManaged:           getenv("SOULROOM_SHELLHUB_MANAGED", "false") == "true",
		CompanyDomain:             getenv("SOULROOM_COMPANY_DOMAIN", ""),
		DeviceOfflineMinutes:      integerRange("SOULROOM_DEVICE_OFFLINE_MINUTES", 5, 2, 1440),
		DefaultTelemetryWindow:    choice("SOULROOM_DEFAULT_TELEMETRY_WINDOW", "1h", "15m", "1h", "24h"),
		DefaultOTAPilotPercent:    integerRange("SOULROOM_DEFAULT_OTA_PILOT_PERCENT", 10, 1, 100),
		DefaultEnrollmentTTLHours: integerRange("SOULROOM_DEFAULT_ENROLLMENT_TTL_HOURS", 24, 1, 720),
		DefaultJobTTLMinutes:      integerRange("SOULROOM_DEFAULT_JOB_TTL_MINUTES", 60, 5, 10080),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func integer(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func integerRange(key string, fallback, minimum, maximum int) int {
	value := integer(key, fallback)
	if value < minimum || value > maximum {
		return fallback
	}
	return value
}

func choice(key, fallback string, allowed ...string) string {
	value := getenv(key, fallback)
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	return fallback
}
