package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr         string
	DeviceAddr       string
	DevicePublicHost string
	StorePath        string
	DevCADir         string
	SessionTTL       time.Duration
	CertificateTTL   time.Duration
	SignupEnabled    bool
	MaxPayloadBytes  int64
	ShellHubURL      string
}

func Load() Config {
	return Config{
		HTTPAddr:         getenv("UFM_HTTP_ADDR", "127.0.0.1:8080"),
		DeviceAddr:       getenv("UFM_DEVICE_ADDR", "127.0.0.1:8443"),
		DevicePublicHost: getenv("UFM_DEVICE_PUBLIC_HOST", "localhost"),
		StorePath:        getenv("UFM_STORE_PATH", ".ufm/control-plane.json"),
		DevCADir:         getenv("UFM_DEV_CA_DIR", ".ufm/ca"),
		SessionTTL:       duration("UFM_SESSION_TTL", 12*time.Hour),
		CertificateTTL:   duration("UFM_CERTIFICATE_TTL", 90*24*time.Hour),
		SignupEnabled:    getenv("UFM_SIGNUP_ENABLED", "false") == "true",
		MaxPayloadBytes:  int64(integer("UFM_MAX_PAYLOAD_BYTES", 2*1024*1024)),
		ShellHubURL:      getenv("UFM_SHELLHUB_URL", ""),
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
