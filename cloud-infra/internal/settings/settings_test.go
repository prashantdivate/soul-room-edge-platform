package settings

import (
	"testing"

	"github.com/soul-room/cloud-infra/internal/model"
)

func validSettings() model.PlatformSettings {
	return model.PlatformSettings{
		OrganizationName:          "Acme Devices",
		CompanyDomain:             "devices.example.com",
		ShellHubURL:               "https://shell.example.com",
		ShellHubSSHPort:           22,
		DeviceOfflineMinutes:      5,
		DefaultTelemetryWindow:    "1h",
		DefaultOTAPilotPercent:    10,
		DefaultEnrollmentTTLHours: 24,
		DefaultJobTTLMinutes:      60,
	}
}

func TestValidateAcceptsOperationalSettings(t *testing.T) {
	if err := Validate(validSettings()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnsafeRanges(t *testing.T) {
	tests := []struct {
		name   string
		change func(*model.PlatformSettings)
	}{
		{"offline window", func(value *model.PlatformSettings) { value.DeviceOfflineMinutes = 1 }},
		{"telemetry window", func(value *model.PlatformSettings) { value.DefaultTelemetryWindow = "7d" }},
		{"pilot percentage", func(value *model.PlatformSettings) { value.DefaultOTAPilotPercent = 0 }},
		{"token lifetime", func(value *model.PlatformSettings) { value.DefaultEnrollmentTTLHours = 0 }},
		{"job expiry", func(value *model.PlatformSettings) { value.DefaultJobTTLMinutes = 2 }},
		{"company domain", func(value *model.PlatformSettings) { value.CompanyDomain = "https://example.com/path" }},
		{"ShellHub URL", func(value *model.PlatformSettings) { value.ShellHubURL = "javascript:alert(1)" }},
		{"ShellHub SSH port", func(value *model.PlatformSettings) { value.ShellHubSSHPort = 70000 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validSettings()
			test.change(&value)
			if err := Validate(value); err == nil {
				t.Fatal("invalid settings were accepted")
			}
		})
	}
}
