package enrollment

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/unified-fleet/cloud-infra/internal/certificates"
	"github.com/unified-fleet/cloud-infra/internal/protocol"
	"github.com/unified-fleet/cloud-infra/internal/storage"
	"github.com/unified-fleet/cloud-infra/internal/tenancy"
)

func TestEnrollmentConsumesOneTimeToken(t *testing.T) {
	store, _ := storage.Open("")
	org, _ := store.CreateOrganization("Tenant", "tenant")
	tc := tenancy.Context{TenantID: org.ID, ActorID: "admin"}
	token, err := NewToken(store, tc, "", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := certificates.LoadOrCreateDevCA(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	der, _ := x509.MarshalPKIXPublicKey(pub)
	req := protocol.EnrollmentRequest{
		Token:        token.PlainToken,
		PublicKeyPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
		Fingerprint:  protocol.DeviceFingerprint{Hostname: "dev1", Architecture: "amd64", OS: "linux"},
	}
	svc := Service{Store: store, CA: ca, CertificateTTL: time.Hour, ServerEndpoint: "https://device.local"}
	res, err := svc.Enroll(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.DeviceID == "" || res.ClientCertificatePEM == "" {
		t.Fatalf("bad enrollment response: %+v", res)
	}
	if _, err := svc.Enroll(context.Background(), req); err == nil {
		t.Fatal("one-time token was reused")
	}
}
