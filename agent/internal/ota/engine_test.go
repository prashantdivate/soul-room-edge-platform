package ota

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu        sync.Mutex
	calls     []string
	failNames map[string]error
}

func (r *fakeRunner) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }
func (r *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if err := r.failNames[name]; err != nil {
		return "", err
	}
	if name == "flatpak" && len(args) > 0 && args[0] == "info" {
		return "old-flatpak-commit", nil
	}
	if strings.Contains(strings.Join(args, " "), "show-provides") {
		installed := "1.0.0"
		for _, call := range r.calls {
			if strings.Contains(call, " install ") {
				installed = "2.0.0"
			}
		}
		return "rootfs-image.version=" + installed, nil
	}
	return "ok", nil
}

func TestEngineRollsBackFlatpakWhenHealthCheckFails(t *testing.T) {
	root := t.TempDir()
	runner := &fakeRunner{failNames: map[string]error{"device-health": errors.New("service is unhealthy")}}
	cfg := Config{Enabled: true, StateDir: filepath.Join(root, "state"), StagingDir: filepath.Join(root, "stage"), TrustedKeysDir: filepath.Join(root, "keys"), PluginDir: filepath.Join(root, "plugins"), MaxArtifactBytes: 1024 * 1024, DownloadTimeout: time.Minute, HealthTimeout: 10 * time.Millisecond, HealthCheckCommand: "device-health"}
	engine, err := NewEngine(cfg, runner)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{UpdateID: "flatpak-1", Adapter: "flatpak", Version: "2.0.0", FlatpakRef: "com.example.Kiosk", FlatpakRemote: "factory"}
	result, err := engine.Execute(context.Background(), request, nil)
	if err == nil || result.State != "failed" {
		t.Fatalf("health failure was not reported: %#v %v", result, err)
	}
	joined := strings.Join(runner.calls, "\n")
	if !strings.Contains(joined, "--commit=old-flatpak-commit") {
		t.Fatalf("previous Flatpak commit was not restored:\n%s", joined)
	}
}

func TestEnginePersistsAcrossRebootAndCommits(t *testing.T) {
	content := []byte("signed operating system artifact")
	server := httptest.NewTLSServer(httpHandler(content))
	defer server.Close()

	root := t.TempDir()
	trusted := filepath.Join(root, "keys")
	if err := os.MkdirAll(trusted, 0700); err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trusted, "production.pem"), pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encodedKey}), 0600); err != nil {
		t.Fatal(err)
	}
	digestBytes := sha256.Sum256(content)
	digest := hex.EncodeToString(digestBytes[:])
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(digest)))

	cfg := Config{Enabled: true, Product: "board-a", StateDir: filepath.Join(root, "state"), StagingDir: filepath.Join(root, "stage"), TrustedKeysDir: trusted, PluginDir: filepath.Join(root, "plugins"), MaxArtifactBytes: 1024 * 1024, DownloadTimeout: time.Minute, HealthTimeout: time.Second, AutoReboot: true}
	runner := &fakeRunner{}
	engine, err := NewEngine(cfg, runner)
	if err != nil {
		t.Fatal(err)
	}
	engine.download.client = server.Client()
	boot := "boot-a"
	engine.bootID = func() string { return boot }
	request := Request{UpdateID: "update-1", Adapter: "mender", Version: "2.0.0", Product: "board-a", Architecture: runtime.GOARCH, ArtifactURL: server.URL + "/release.mender", ArtifactSize: int64(len(content)), Digest: digest, Signature: signature, SigningKeyID: "production", CompatibleFrom: []string{"1.0.0"}}

	first, err := engine.Execute(context.Background(), request, nil)
	if !errors.Is(err, ErrRebootPending) || first.State != "rebooting" {
		t.Fatalf("first execution = %#v, %v", first, err)
	}
	boot = "boot-b"
	second, err := engine.Execute(context.Background(), request, nil)
	if err != nil || second.State != "succeeded" {
		t.Fatalf("second execution = %#v, %v", second, err)
	}
	joined := strings.Join(runner.calls, "\n")
	for _, expected := range []string{"mender-update install", "systemctl reboot --no-block", "mender-update commit"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing command %q in:\n%s", expected, joined)
		}
	}
}

func TestValidateRequestRequiresConfiguredProduct(t *testing.T) {
	request := Request{UpdateID: "u-1", Adapter: "rauc", Version: "2", Product: "board", Architecture: runtime.GOARCH, ArtifactURL: "https://example.test/u.raucb", Digest: strings.Repeat("a", 64), Signature: base64.StdEncoding.EncodeToString(make([]byte, 64)), SigningKeyID: "key"}
	if err := ValidateRequest(Config{MaxArtifactBytes: 1024 * 1024}, request); err == nil {
		t.Fatal("OS OTA accepted without a configured device product")
	}
}

func httpHandler(content []byte) *testHandler { return &testHandler{content: content} }

type testHandler struct{ content []byte }

func (h *testHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(h.content) }
