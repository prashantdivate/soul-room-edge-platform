package enrollment

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/unified-fleet/edge-agent/internal/config"
	"github.com/unified-fleet/edge-agent/internal/identity"
	"github.com/unified-fleet/edge-agent/internal/protocol"
)

type Client struct {
	Config config.Config
	HTTP   *http.Client
}

func NewClient(cfg config.Config) (*Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	if cfg.Server.CAFile != "" {
		ca, err := os.ReadFile(cfg.Server.CAFile)
		if err != nil {
			return nil, err
		}
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to load CA file %s", cfg.Server.CAFile)
		}
	}
	return &Client{
		Config: cfg,
		HTTP: &http.Client{
			Timeout: cfg.Server.ConnectTimeout,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    pool,
			}},
		},
	}, nil
}

func (c *Client) Enroll(ctx context.Context, token, requestedName string) (identity.Identity, error) {
	signer, err := identity.LoadOrCreateFileSigner(c.Config.Identity.StateDir)
	if err != nil {
		return identity.Identity{}, err
	}
	pub, err := signer.PublicKeyPEM()
	if err != nil {
		return identity.Identity{}, err
	}
	req := protocol.EnrollmentRequest{
		Token:         token,
		PublicKeyPEM:  string(pub),
		RequestedName: requestedName,
		Fingerprint:   Fingerprint(),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return identity.Identity{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Config.Server.Endpoint+"/v1/enroll", bytes.NewReader(body))
	if err != nil {
		return identity.Identity{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return identity.Identity{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return identity.Identity{}, fmt.Errorf("enrollment failed: %s", res.Status)
	}
	var out protocol.EnrollmentResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return identity.Identity{}, err
	}
	if out.DeviceID == "" || out.TenantID == "" {
		return identity.Identity{}, fmt.Errorf("enrollment response missing identity")
	}
	id := identity.Identity{
		TenantID:       out.TenantID,
		DeviceID:       out.DeviceID,
		CertificatePEM: out.ClientCertificatePEM,
		CAPEM:          out.CACertificatePEM,
	}
	return id, identity.SaveIdentity(c.Config.Identity.StateDir, id)
}

func Fingerprint() protocol.DeviceFingerprint {
	host, _ := os.Hostname()
	machine := readFirst("/etc/machine-id", "/var/lib/dbus/machine-id")
	sum := sha256.Sum256([]byte(machine))
	return protocol.DeviceFingerprint{
		Hostname:      host,
		MachineIDHash: hex.EncodeToString(sum[:]),
		Serial:        deviceSerial(),
		Product:       deviceProduct(),
		Architecture:  runtime.GOARCH,
		OS:            runtime.GOOS,
		Kernel:        readFirst("/proc/sys/kernel/osrelease"),
	}
}

func deviceProduct() string {
	for _, path := range []string{"/proc/device-tree/model", "/sys/firmware/devicetree/base/model", "/sys/devices/virtual/dmi/id/product_name"} {
		value := readFirst(path)
		if value != "unknown" && value != "" {
			return value
		}
	}
	return "unknown"
}

func deviceSerial() string {
	for _, path := range []string{"/sys/firmware/devicetree/base/serial-number", "/sys/devices/virtual/dmi/id/product_serial"} {
		value := readFirst(path)
		if value != "unknown" && value != "" {
			return value
		}
	}
	file, err := os.Open("/proc/cpuinfo")
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			key, value, ok := strings.Cut(scanner.Text(), ":")
			if ok && strings.EqualFold(strings.TrimSpace(key), "serial") {
				return strings.TrimSpace(value)
			}
		}
	}
	return "unknown"
}

func readFirst(paths ...string) string {
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err == nil {
			return string(trim(b))
		}
	}
	return "unknown"
}

func trim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	return b
}

func ContextWithDefaultTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, 30*time.Second)
}
