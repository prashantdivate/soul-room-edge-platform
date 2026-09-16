package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`version: 1
server:
  endpoint: https://example.test
identity:
  state_dir: /tmp/edge/id
storage:
  state_dir: /tmp/edge
  max_queue_bytes: 1048576
jobs:
  allow_shell: false
ota:
  product: imx8mp-kiosk
  auto_reboot: true
`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Endpoint != "https://example.test" {
		t.Fatalf("endpoint = %s", cfg.Server.Endpoint)
	}
	if cfg.OTA.Product != "imx8mp-kiosk" || !cfg.OTA.AutoReboot {
		t.Fatalf("OTA configuration was not loaded: %+v", cfg.OTA)
	}
}

func TestRejectHTTP(t *testing.T) {
	cfg := Default()
	cfg.Server.Endpoint = "http://example.test"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := Default()
	cfg.Server.Endpoint = "https://fleet.example.test:8443"
	cfg.Server.CAFile = "/etc/edge-agent/ca.pem"
	cfg.Location.Source = "static"
	cfg.Location.Latitude = 18.5204
	cfg.Location.Longitude = 73.8567
	cfg.Location.Label = "Pune lab"
	cfg.Location.IPURL = "https://location.example.test/"
	cfg.OTA.Product = "imx8mp-kiosk"
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.Endpoint != cfg.Server.Endpoint || loaded.Location.Label != cfg.Location.Label || loaded.Location.IPURL != cfg.Location.IPURL || loaded.OTA.Product != cfg.OTA.Product {
		t.Fatalf("saved config did not round trip: %+v", loaded)
	}
}

func TestValidateIPLocation(t *testing.T) {
	cfg := Default()
	cfg.Location.Source = "ip"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.Location.IPURL = "http://location.example.test"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure IP location URL to be rejected")
	}
}
