package ota

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var signingKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

func verifyArtifact(path string, request Request, trustedKeysDir string) error {
	if !digestMatches(path, request.Digest) {
		return errors.New("artifact SHA-256 digest mismatch")
	}
	return verifyReleaseSignature(request, trustedKeysDir)
}

func verifyReleaseSignature(request Request, trustedKeysDir string) error {
	if !signingKeyIDPattern.MatchString(request.SigningKeyID) {
		return errors.New("invalid signing key ID")
	}
	keyPEM, err := os.ReadFile(filepath.Join(trustedKeysDir, request.SigningKeyID+".pem"))
	if err != nil {
		return fmt.Errorf("trusted release key %q is unavailable: %w", request.SigningKeyID, err)
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return errors.New("trusted release key is not PEM encoded")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse trusted release key: %w", err)
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return errors.New("trusted release key must be Ed25519")
	}
	signature, err := base64.StdEncoding.DecodeString(request.Signature)
	if err != nil {
		signature, err = base64.RawStdEncoding.DecodeString(request.Signature)
	}
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("detached signature must be a base64 Ed25519 signature")
	}
	if !ed25519.Verify(publicKey, []byte(strings.ToLower(request.Digest)), signature) {
		return errors.New("release signature is invalid")
	}
	return nil
}
