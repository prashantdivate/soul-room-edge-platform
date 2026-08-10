package diagnostics

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/soul-room/edge-agent/internal/config"
	"github.com/soul-room/edge-agent/internal/storage"
)

type BundleInfo struct {
	AgentVersion string             `json:"agent_version"`
	Config       config.Config      `json:"config"`
	Queue        storage.QueueStats `json:"queue"`
	CreatedAt    time.Time          `json:"created_at"`
}

func Create(path, version string, cfg config.Config, stats storage.QueueStats) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	w, err := zw.Create("summary.json")
	if err != nil {
		return err
	}
	cfg.Server.CAFile = redact(cfg.Server.CAFile)
	info := BundleInfo{AgentVersion: version, Config: cfg, Queue: stats, CreatedAt: time.Now()}
	return json.NewEncoder(w).Encode(info)
}

func redact(s string) string {
	if s == "" {
		return ""
	}
	return "<redacted-path>"
}
