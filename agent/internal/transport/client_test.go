package transport

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/soul-room/edge-agent/internal/config"
	"github.com/soul-room/edge-agent/internal/identity"
)

func TestNewRejectsInvalidEnrolledCA(t *testing.T) {
	_, err := New(config.Default(), identity.Identity{CAPEM: "not a certificate"})
	if err == nil {
		t.Fatal("expected invalid enrolled CA to be rejected")
	}
}

func TestNewUsesEnrolledCABeforeConfiguredBootstrapCA(t *testing.T) {
	cfg := config.Default()
	cfg.Server.CAFile = t.TempDir() + "/missing-bootstrap-ca.pem"
	_, err := New(cfg, identity.Identity{CAPEM: testCAPEM(t)})
	if err != nil {
		t.Fatalf("expected enrolled CA to take precedence: %v", err)
	}
}

func testCAPEM(t *testing.T) string {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}
