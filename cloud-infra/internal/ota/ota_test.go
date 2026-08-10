package ota

import "testing"

func TestValidateMetadata(t *testing.T) {
	valid := Campaign{Name: "release", ArtifactURL: "https://example.test/release.mender", Version: "2", Digest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Signature: "key-1", Adapter: "mender", TargetIDs: []string{"device-1"}, CanaryPercent: 10}
	if err := ValidateMetadata(valid); err != nil {
		t.Fatalf("valid campaign rejected: %v", err)
	}
	invalid := valid
	invalid.Adapter = "shell"
	if err := ValidateMetadata(invalid); err == nil {
		t.Fatal("unsupported adapter accepted")
	}
}

func TestValidateFlatpakCampaign(t *testing.T) {
	valid := Campaign{Name: "application update", Version: "1.4.0", Adapter: "flatpak", FlatpakRef: "org.example.Axon", FlatpakRemote: "factory", FlatpakCommit: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", FlatpakRepositoryURL: "https://updates.example.test/factory.flatpakrepo", TargetIDs: []string{"device-1"}, CanaryPercent: 25}
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
