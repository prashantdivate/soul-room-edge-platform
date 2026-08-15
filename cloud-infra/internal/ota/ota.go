package ota

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	flatpakRefPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{2,199}$`)
	flatpakRemotePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	flatpakCommitPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type Campaign struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	Name                 string    `json:"name"`
	ArtifactID           string    `json:"artifact_id"`
	ArtifactURL          string    `json:"artifact_url"`
	Version              string    `json:"version"`
	Product              string    `json:"product"`
	Architecture         string    `json:"architecture"`
	Digest               string    `json:"digest"`
	Signature            string    `json:"signature"`
	SigningKeyID         string    `json:"signing_key_id,omitempty"`
	ArtifactSize         int64     `json:"artifact_size,omitempty"`
	CompatibleFrom       []string  `json:"compatible_from,omitempty"`
	Adapter              string    `json:"adapter"`
	FlatpakRef           string    `json:"flatpak_ref,omitempty"`
	FlatpakRemote        string    `json:"flatpak_remote,omitempty"`
	FlatpakCommit        string    `json:"flatpak_commit,omitempty"`
	FlatpakRepositoryURL string    `json:"flatpak_repository_url,omitempty"`
	TargetIDs            []string  `json:"target_device_ids"`
	CanaryPercent        int       `json:"canary_percent"`
	CreatedBy            string    `json:"created_by,omitempty"`
	State                string    `json:"state"`
	CreatedAt            time.Time `json:"created_at"`
}

func ValidateMetadata(c Campaign) error {
	if c.Name == "" || c.Version == "" {
		return errors.New("update metadata requires name and version")
	}
	if c.Adapter == "flatpak" {
		if !flatpakRefPattern.MatchString(c.FlatpakRef) {
			return errors.New("Flatpak reference is invalid")
		}
		if !flatpakRemotePattern.MatchString(c.FlatpakRemote) {
			return errors.New("Flatpak remote is invalid")
		}
		if c.FlatpakCommit != "" && !flatpakCommitPattern.MatchString(c.FlatpakCommit) {
			return errors.New("Flatpak commit must be a 64-character OSTree commit")
		}
		if c.FlatpakRepositoryURL != "" {
			parsed, err := url.Parse(c.FlatpakRepositoryURL)
			if err != nil || parsed.Scheme != "https" || parsed.Host == "" || !strings.HasSuffix(strings.ToLower(parsed.Path), ".flatpakrepo") {
				return errors.New("Flatpak repository URL must use HTTPS and point to a .flatpakrepo descriptor")
			}
		}
		return validateRollout(c)
	}
	if c.ArtifactURL == "" || c.Digest == "" || c.Signature == "" || c.SigningKeyID == "" {
		return errors.New("OTA metadata requires an artifact URL, version, digest, detached signature, and signing key ID")
	}
	if !supportedArchitecture(c.Architecture) {
		return errors.New("OTA architecture must be arm64/aarch64, armv7, or amd64")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}$`).MatchString(c.Adapter) {
		return errors.New("update adapter name is invalid")
	}
	parsed, err := url.Parse(c.ArtifactURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("OTA artifact URL must use HTTPS")
	}
	digest, err := hex.DecodeString(c.Digest)
	if err != nil || len(digest) != 32 {
		return errors.New("OTA digest must be a SHA-256 value")
	}
	if c.ArtifactSize < 0 {
		return errors.New("OTA artifact size cannot be negative")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`).MatchString(c.SigningKeyID) {
		return errors.New("OTA signing key ID is invalid")
	}
	signature, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		signature, err = base64.RawStdEncoding.DecodeString(c.Signature)
	}
	if err != nil || len(signature) != 64 {
		return errors.New("OTA signature must be a base64 Ed25519 signature")
	}
	return validateRollout(c)
}

func supportedArchitecture(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "arm64", "aarch64", "armv7", "armv7l", "armhf", "amd64", "x86_64":
		return true
	default:
		return false
	}
}

func validateRollout(c Campaign) error {
	if c.CanaryPercent < 1 || c.CanaryPercent > 100 {
		return errors.New("OTA canary percentage must be between 1 and 100")
	}
	if len(c.TargetIDs) == 0 {
		return errors.New("OTA campaign requires at least one target device")
	}
	return nil
}
