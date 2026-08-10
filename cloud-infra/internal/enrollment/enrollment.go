package enrollment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/soul-room/cloud-infra/internal/certificates"
	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/protocol"
	"github.com/soul-room/cloud-infra/internal/storage"
	"github.com/soul-room/cloud-infra/internal/tenancy"
)

type Service struct {
	Store          *storage.Store
	CA             *certificates.Authority
	CertificateTTL time.Duration
	ServerEndpoint string
}

func NewToken(store *storage.Store, tc tenancy.Context, profileID string, ttl time.Duration) (model.EnrollmentToken, error) {
	plain, err := randomToken(32)
	if err != nil {
		return model.EnrollmentToken{}, err
	}
	token := model.EnrollmentToken{
		ProfileID:  profileID,
		TokenHash:  HashToken(plain),
		PlainToken: plain,
		MaxDevices: 1,
		ExpiresAt:  time.Now().Add(ttl),
	}
	return store.CreateEnrollmentToken(tc, token)
}

func (s Service) Enroll(ctx context.Context, req protocol.EnrollmentRequest) (protocol.EnrollmentResponse, error) {
	token := req.Token
	if token == "" {
		token = req.EnrollmentToken
	}
	if token == "" {
		return protocol.EnrollmentResponse{}, errors.New("enrollment token is required")
	}
	enr, err := s.Store.ConsumeEnrollmentToken(HashToken(token), req.Fingerprint.Serial, req.Fingerprint.Product)
	if err != nil {
		return protocol.EnrollmentResponse{}, err
	}
	device := model.Device{
		TenantID:         enr.TenantID,
		ProfileID:        enr.ProfileID,
		DisplayName:      first(req.RequestedName, req.Fingerprint.Hostname, "edge-device"),
		EnrollmentStatus: "enrolled",
		Architecture:     req.Fingerprint.Architecture,
		OS:               req.Fingerprint.OS,
		Kernel:           req.Fingerprint.Kernel,
		Serial:           req.Fingerprint.Serial,
		HardwareModel:    req.Fingerprint.Product,
		Capabilities:     []string{"telemetry", "inventory", "jobs"},
		Presence:         "recently_seen",
		UpdateState:      "idle",
		HealthState:      "unknown",
		LastSeenAt:       time.Now(),
	}
	device, err = s.Store.CreateDevice(device)
	if err != nil {
		return protocol.EnrollmentResponse{}, err
	}
	certPEM, serial, expires, err := s.CA.IssueDeviceCertificate(req.PublicKeyPEM, device.ID, s.CertificateTTL)
	if err != nil {
		return protocol.EnrollmentResponse{}, err
	}
	device.CertificateSerial = serial
	device.CertificateSubject = device.ID
	device.CertificateExpiresAt = expires
	if err := s.Store.UpdateDevice(device); err != nil {
		return protocol.EnrollmentResponse{}, err
	}
	if len(req.Inventory) > 0 {
		_ = s.Store.SaveInventory(tenancy.Context{TenantID: enr.TenantID}, device.ID, req.Inventory)
	}
	return protocol.EnrollmentResponse{
		TenantID:             enr.TenantID,
		DeviceID:             device.ID,
		ClientCertificatePEM: string(certPEM),
		CACertificatePEM:     string(s.CA.CertPEM),
		ServerEndpoint:       s.ServerEndpoint,
	}, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
