package applications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileDeployRequest struct {
	Destination string `json:"destination"`
	Content     []byte `json:"content"`
	SHA256      string `json:"sha256"`
	Mode        uint32 `json:"mode"`
	Backup      bool   `json:"backup"`
}

type FileDeployer struct {
	AllowedPrefixes []string
	MaxBytes        int64
	Audit           func(event string, fields map[string]string)
}

func (d FileDeployer) Deploy(ctx context.Context, raw json.RawMessage) error {
	var req FileDeployRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return err
	}
	if req.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	if int64(len(req.Content)) > d.MaxBytes {
		return fmt.Errorf("file exceeds maximum size")
	}
	clean := filepath.Clean(req.Destination)
	if !d.allowed(clean) {
		return fmt.Errorf("destination is not allowlisted")
	}
	sum := sha256.Sum256(req.Content)
	if req.SHA256 != "" && hex.EncodeToString(sum[:]) != strings.ToLower(req.SHA256) {
		return fmt.Errorf("checksum mismatch")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Dir(clean), 0755); err != nil {
		return err
	}
	if req.Backup {
		if existing, err := os.ReadFile(clean); err == nil {
			if err := os.WriteFile(clean+".bak", existing, 0600); err != nil {
				return err
			}
		}
	}
	mode := os.FileMode(0644)
	if req.Mode != 0 {
		mode = os.FileMode(req.Mode)
	}
	tmp, err := os.CreateTemp(filepath.Dir(clean), ".edge-agent-stage-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(req.Content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, clean); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if d.Audit != nil {
		d.Audit("file_deployed", map[string]string{"destination": clean})
	}
	return nil
}

func (d FileDeployer) allowed(path string) bool {
	for _, p := range d.AllowedPrefixes {
		p = filepath.Clean(p)
		if path == p || strings.HasPrefix(path, p+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
