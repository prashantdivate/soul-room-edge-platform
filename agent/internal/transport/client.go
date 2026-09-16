package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/soul-room/edge-agent/internal/config"
	"github.com/soul-room/edge-agent/internal/identity"
	"github.com/soul-room/edge-agent/internal/protocol"
)

type Client struct {
	endpoint string
	http     *http.Client
	identity identity.Identity
	sequence atomic.Uint64
}

func New(cfg config.Config, id identity.Identity) (*Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	caPEM := []byte(id.CAPEM)
	if len(caPEM) == 0 && cfg.Server.CAFile != "" {
		caPEM, err = os.ReadFile(cfg.Server.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read server CA %s: %w", cfg.Server.CAFile, err)
		}
	}
	if len(caPEM) > 0 && !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("server CA is not a valid PEM certificate")
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
	if id.CertificatePEM != "" {
		keyPEM, err := os.ReadFile(cfg.Identity.StateDir + "/device_key.pem")
		if err != nil {
			return nil, err
		}
		cert, err := tls.X509KeyPair([]byte(id.CertificatePEM), keyPEM)
		if err != nil {
			return nil, err
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}
	return &Client{
		endpoint: cfg.Server.Endpoint,
		identity: id,
		http: &http.Client{
			Timeout:   cfg.Server.ConnectTimeout,
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}, nil
}

func (c *Client) Status(ctx context.Context) error {
	var response struct {
		Status string `json:"status"`
	}
	if err := c.GetDeviceJSON(ctx, "/v1/status", &response); err != nil {
		return err
	}
	if response.Status != "accepted" {
		return fmt.Errorf("unexpected device status %q", response.Status)
	}
	return nil
}

func (c *Client) PostEnvelope(ctx context.Context, path, schema string, payload any) ([]byte, error) {
	env, err := protocol.NewEnvelope(c.identity.TenantID, c.identity.DeviceID, schema, c.sequence.Add(1), payload)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(env)
	if err != nil {
		return nil, err
	}
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err == nil && res.StatusCode >= 200 && res.StatusCode < 300 {
			defer res.Body.Close()
			return io.ReadAll(io.LimitReader(res.Body, 1<<20))
		}
		if res != nil {
			last = fmt.Errorf("server returned %s", res.Status)
			_ = res.Body.Close()
		} else {
			last = err
		}
		wait := time.Duration(1<<attempt)*time.Second + time.Duration(rand.Int63n(int64(250*time.Millisecond)))
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, last
}

func (c *Client) GetDeviceJSON(ctx context.Context, path string, out any) error {
	return c.deviceJSON(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) PostDeviceJSON(ctx context.Context, path string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.deviceJSON(ctx, http.MethodPost, path, body, out)
}

func (c *Client) deviceJSON(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("X-Tenant-ID", c.identity.TenantID)
	req.Header.Set("X-Device-ID", c.identity.DeviceID)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("server returned %s: %s", res.Status, string(responseBody))
	}
	if out == nil || len(responseBody) == 0 {
		return nil
	}
	return json.Unmarshal(responseBody, out)
}
