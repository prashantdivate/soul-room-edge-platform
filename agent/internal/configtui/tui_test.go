package configtui

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soul-room/edge-agent/internal/config"
)

func TestRunEditsAndSavesServerSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := config.Default()
	input := strings.NewReader("1\nhttps://fleet.example.test:8443\n/etc/edge-agent/ca.pem\n20s\n0\n")
	var output bytes.Buffer
	if err := Run(path, cfg, input, &output); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.Endpoint != "https://fleet.example.test:8443" || loaded.Server.ConnectTimeout.String() != "20s" {
		t.Fatalf("unexpected saved server config: %+v", loaded.Server)
	}
}
