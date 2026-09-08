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

func TestValidateOSTreeRepositoryRequest(t *testing.T) {
	commit := strings.Repeat("a", 64)
	request := Request{UpdateID: "ostree-1", Adapter: "ostree", Version: "2", Product: "board-a", Architecture: runtime.GOARCH, Digest: commit, Signature: base64.StdEncoding.EncodeToString(make([]byte, 64)), SigningKeyID: "key", OSTreeRemote: "factory", OSTreeRef: "device/stable", OSTreeCommit: commit}
	if err := ValidateRequest(Config{Product: "board-a"}, request); err != nil {
		t.Fatalf("valid OSTree repository request rejected: %v", err)
	}
	request.Digest = strings.Repeat("b", 64)
	if err := ValidateRequest(Config{Product: "board-a"}, request); err == nil {
		t.Fatal("OSTree repository request accepted a digest that does not pin the target commit")
	}
}

type ostreeRunner struct {
	calls         []string
	target        string
	current       string
	staged        string
	remoteURL     string
	contentURL    string
	gpgVerify     string
	tlsPermissive string
}

func (r *ostreeRunner) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }
func (r *ostreeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	r.calls = append(r.calls, call)
	joined := strings.Join(args, " ")
	switch {
	case joined == "--repo=/ostree/repo remote show-url factory":
		return r.remoteURL, nil
	case strings.HasSuffix(joined, "gpg-verify"):
		return r.gpgVerify, nil
	case strings.HasSuffix(joined, "tls-permissive"):
		return r.tlsPermissive, nil
	case strings.HasSuffix(joined, "contenturl"):
		return r.contentURL, nil
	case strings.Contains(joined, " rev-parse "):
		return r.target, nil
	case strings.HasPrefix(joined, "admin deploy ") || strings.Contains(joined, " --os="):
		r.staged = args[len(args)-1]
		return "", nil
	case joined == "admin status":
		return "* device " + r.current, nil
	default:
		return "", nil
	}
}

func TestOSTreeRepositoryDeploysAndConfirmsPinnedCommit(t *testing.T) {
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
	oldCommit := strings.Repeat("1", 64)
	target := strings.Repeat("2", 64)
	runner := &ostreeRunner{target: target, current: oldCommit, remoteURL: "https://updates.example.test/repo", gpgVerify: "true", tlsPermissive: "false"}
	cfg := Config{Enabled: true, Product: "board-a", StateDir: filepath.Join(root, "state"), StagingDir: filepath.Join(root, "stage"), TrustedKeysDir: trusted, PluginDir: filepath.Join(root, "plugins"), HealthTimeout: time.Second, AutoReboot: true}
	engine, err := NewEngine(cfg, runner)
	if err != nil {
		t.Fatal(err)
	}
	boot := "boot-a"
	engine.bootID = func() string { return boot }
	request := Request{UpdateID: "ostree-online-1", Adapter: "ostree", Version: "2.0.0", Product: "board-a", Architecture: runtime.GOARCH, Digest: target, Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, []byte(target))), SigningKeyID: "production", OSTreeRemote: "factory", OSTreeRef: "device/stable", OSTreeCommit: target}

	first, err := engine.Execute(context.Background(), request, nil)
	if !errors.Is(err, ErrRebootPending) || first.State != "rebooting" {
		t.Fatalf("first execution = %#v, %v", first, err)
	}
	if runner.staged != target {
		t.Fatalf("deployed %q instead of pinned commit %q", runner.staged, target)
	}
	joined := strings.Join(runner.calls, "\n")
	if !strings.Contains(joined, "ostree --repo=/ostree/repo pull factory device/stable@"+target) {
		t.Fatalf("pinned repository pull was not used:\n%s", joined)
	}

	runner.current = target
	boot = "boot-b"
	second, err := engine.Execute(context.Background(), request, nil)
	if err != nil || second.State != "succeeded" {
		t.Fatalf("second execution = %#v, %v", second, err)
	}
}

func TestOSTreeRepositoryRejectsUnsafeRemote(t *testing.T) {
	request := Request{Adapter: "ostree", OSTreeRemote: "factory", OSTreeRef: "device/stable", OSTreeCommit: strings.Repeat("a", 64)}
	for name, runner := range map[string]*ostreeRunner{
		"http":           {remoteURL: "http://updates.example.test/repo"},
		"gpg disabled":   {remoteURL: "https://updates.example.test/repo", gpgVerify: "false"},
		"permissive TLS": {remoteURL: "https://updates.example.test/repo", gpgVerify: "true", tlsPermissive: "true"},
		"http content":   {remoteURL: "https://updates.example.test/repo", gpgVerify: "true", contentURL: "http://content.example.test/repo"},
	} {
		t.Run(name, func(t *testing.T) {
			adapter := &nativeAdapter{name: "ostree", runner: runner}
			if err := adapter.Check(context.Background(), request, ""); err == nil {
				t.Fatal("unsafe OSTree remote accepted")
			}
		})
	}
}

func TestOSTreeWrongBootedCommitTriggersRollback(t *testing.T) {
	root := t.TempDir()
	oldCommit := strings.Repeat("1", 64)
	target := strings.Repeat("2", 64)
	runner := &ostreeRunner{target: target, current: strings.Repeat("3", 64)}
	cfg := Config{Enabled: true, Product: "board-a", StateDir: filepath.Join(root, "state"), StagingDir: filepath.Join(root, "stage"), TrustedKeysDir: filepath.Join(root, "keys"), PluginDir: filepath.Join(root, "plugins"), HealthTimeout: time.Second, AutoReboot: true}
	engine, err := NewEngine(cfg, runner)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{UpdateID: "ostree-wrong-boot", Adapter: "ostree", Version: "2.0.0", OSTreeCommit: target}
	tx := transaction{Request: request, Phase: "reboot_requested", PreviousVersion: oldCommit}
	result, err := engine.confirmAndCommit(context.Background(), &nativeAdapter{name: "ostree", runner: runner}, &tx, func(string) {})
	if !errors.Is(err, ErrRebootPending) || result.State != "rebooting" {
		t.Fatalf("wrong booted commit did not trigger rollback reboot: %#v, %v", result, err)
	}
	if runner.staged != oldCommit {
		t.Fatalf("rollback staged %q instead of previous commit %q", runner.staged, oldCommit)
	}
}

func httpHandler(content []byte) *testHandler { return &testHandler{content: content} }

type testHandler struct{ content []byte }

func (h *testHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(h.content) }
