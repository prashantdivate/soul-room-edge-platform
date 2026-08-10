package applications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileDeployAllowlistAndChecksum(t *testing.T) {
	dir := t.TempDir()
	content := []byte("hello")
	sum := sha256.Sum256(content)
	req := FileDeployRequest{Destination: filepath.Join(dir, "app.conf"), Content: content, SHA256: hex.EncodeToString(sum[:]), Backup: true}
	raw, _ := json.Marshal(req)
	deployer := FileDeployer{AllowedPrefixes: []string{dir}, MaxBytes: 1024}
	if err := deployer.Deploy(context.Background(), raw); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(req.Destination)
	if string(got) != "hello" {
		t.Fatalf("content = %q", got)
	}
	req.Destination = filepath.Join(t.TempDir(), "nope")
	raw, _ = json.Marshal(req)
	if err := deployer.Deploy(context.Background(), raw); err == nil {
		t.Fatal("expected allowlist failure")
	}
}
