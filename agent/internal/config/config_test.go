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
}

func TestRejectHTTP(t *testing.T) {
	cfg := Default()
	cfg.Server.Endpoint = "http://example.test"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
