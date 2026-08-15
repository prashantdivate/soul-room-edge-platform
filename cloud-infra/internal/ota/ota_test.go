package ota

import (
	"encoding/base64"
	"testing"
)

func testSignature() string { return base64.StdEncoding.EncodeToString(make([]byte, 64)) }

func TestValidateMetadata(t *testing.T) {
	valid := Campaign{Name: "release", ArtifactURL: "https://example.test/release.mender", Version: "2", Architecture: "arm64", Digest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Signature: testSignature(), SigningKeyID: "release-2026", Adapter: "mender", TargetIDs: []string{"device-1"}, CanaryPercent: 10}
	if err := ValidateMetadata(valid); err != nil {
		t.Fatalf("valid campaign rejected: %v", err)
	}
	invalid := valid
	invalid.Adapter = "shell;command"
	if err := ValidateMetadata(invalid); err == nil {
		t.Fatal("unsupported adapter accepted")
	}
}

func TestValidateARMArchitectures(t *testing.T) {
	base := Campaign{Name: "release", ArtifactURL: "https://example.test/release.mender", Version: "2", Digest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Signature: testSignature(), SigningKeyID: "release-2026", Adapter: "mender", TargetIDs: []string{"device-1"}, CanaryPercent: 10}
	for _, architecture := range []string{"arm64", "aarch64", "armv7", "armv7l"} {
		campaign := base
		campaign.Architecture = architecture
		if err := ValidateMetadata(campaign); err != nil {
			t.Fatalf("%s rejected: %v", architecture, err)
		}
	}
}

func TestValidateFlatpakCampaign(t *testing.T) {
	valid := Campaign{Name: "application update", Version: "1.4.0", Adapter: "flatpak", FlatpakRef: "org.soulroom.Console", FlatpakRemote: "factory", FlatpakCommit: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", FlatpakRepositoryURL: "https://updates.example.test/factory.flatpakrepo", TargetIDs: []string{"device-1"}, CanaryPercent: 25}
	if err := ValidateMetadata(valid); err != nil {
		t.Fatalf("valid Flatpak campaign rejected: %v", err)
	}
	invalid := valid
	invalid.FlatpakRef = "org.example.App; reboot"
	if err := ValidateMetadata(invalid); err == nil {
		t.Fatal("unsafe Flatpak reference accepted")
	}
	invalid = valid
	invalid.FlatpakRepositoryURL = "http://updates.example.test/factory.flatpakrepo"
	if err := ValidateMetadata(invalid); err == nil {
		t.Fatal("insecure Flatpak repository URL accepted")
	}
}
