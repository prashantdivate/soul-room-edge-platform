package identity

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Identity struct {
	TenantID       string `json:"tenant_id"`
	DeviceID       string `json:"device_id"`
	CertificatePEM string `json:"certificate_pem"`
	CAPEM          string `json:"ca_pem"`
}

type Signer interface {
	crypto.Signer
	PublicKeyPEM() ([]byte, error)
	Fingerprint() (string, error)
}

type FileSigner struct {
	private ed25519.PrivateKey
}

func LoadOrCreateFileSigner(stateDir string) (*FileSigner, error) {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(stateDir, "device_key.pem")
	if b, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(b)
		if block == nil {
			return nil, fmt.Errorf("invalid private key pem")
		}
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		priv, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not ed25519")
		}
		return &FileSigner{private: priv}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, err
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, block, 0600); err != nil {
		return nil, err
	}
	return &FileSigner{private: priv}, nil
}

func (s *FileSigner) Public() crypto.PublicKey { return s.private.Public() }

func (s *FileSigner) Sign(randReader io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.private.Sign(randReader, digest, opts)
}

func (s *FileSigner) PublicKeyPEM() ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(s.private.Public())
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func (s *FileSigner) Fingerprint() (string, error) {
	pub, err := s.PublicKeyPEM()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:]), nil
}

func SaveIdentity(stateDir string, id Identity) error {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stateDir, "device_id"), []byte(id.DeviceID+"\n"), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(stateDir, "tenant_id"), []byte(id.TenantID+"\n"), 0600); err != nil {
		return err
	}
	if id.CertificatePEM != "" {
		if err := os.WriteFile(filepath.Join(stateDir, "device.crt"), []byte(id.CertificatePEM), 0600); err != nil {
			return err
		}
	}
	if id.CAPEM != "" {
		if err := os.WriteFile(filepath.Join(stateDir, "ca.pem"), []byte(id.CAPEM), 0644); err != nil {
			return err
		}
	}
	return nil
}

func LoadIdentity(stateDir string) (Identity, error) {
	read := func(name string) string {
		b, _ := os.ReadFile(filepath.Join(stateDir, name))
		return string(bytesTrimSpace(b))
	}
	id := Identity{
		TenantID:       read("tenant_id"),
		DeviceID:       read("device_id"),
		CertificatePEM: read("device.crt"),
		CAPEM:          read("ca.pem"),
	}
	if id.DeviceID == "" || id.TenantID == "" {
		return id, fmt.Errorf("identity is not enrolled")
	}
	return id, nil
}

func bytesTrimSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ' || b[len(b)-1] == '\t') {
		b = b[:len(b)-1]
	}
	for len(b) > 0 && (b[0] == '\n' || b[0] == '\r' || b[0] == ' ' || b[0] == '\t') {
		b = b[1:]
	}
	return b
}
