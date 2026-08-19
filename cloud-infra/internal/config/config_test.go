package config

import "testing"

func TestOperationalDefaultsCanComeFromEnvironment(t *testing.T) {
	t.Setenv("SOULROOM_COMPANY_DOMAIN", "fleet.example.com")
	t.Setenv("SOULROOM_DEVICE_OFFLINE_MINUTES", "12")
	t.Setenv("SOULROOM_DEFAULT_TELEMETRY_WINDOW", "24h")
	t.Setenv("SOULROOM_DEFAULT_OTA_PILOT_PERCENT", "20")
	t.Setenv("SOULROOM_DEFAULT_ENROLLMENT_TTL_HOURS", "48")
	t.Setenv("SOULROOM_DEFAULT_JOB_TTL_MINUTES", "90")
	got := Load()
	if got.CompanyDomain != "fleet.example.com" || got.DeviceOfflineMinutes != 12 || got.DefaultTelemetryWindow != "24h" || got.DefaultOTAPilotPercent != 20 || got.DefaultEnrollmentTTLHours != 48 || got.DefaultJobTTLMinutes != 90 {
		t.Fatalf("environment defaults were not loaded: %+v", got)
	}
}

func TestInvalidOperationalDefaultsFallBack(t *testing.T) {
	t.Setenv("SOULROOM_DEVICE_OFFLINE_MINUTES", "0")
	t.Setenv("SOULROOM_DEFAULT_TELEMETRY_WINDOW", "7d")
	t.Setenv("SOULROOM_DEFAULT_OTA_PILOT_PERCENT", "101")
	t.Setenv("SOULROOM_DEFAULT_ENROLLMENT_TTL_HOURS", "0")
	t.Setenv("SOULROOM_DEFAULT_JOB_TTL_MINUTES", "2")
	got := Load()
	if got.DeviceOfflineMinutes != 5 || got.DefaultTelemetryWindow != "1h" || got.DefaultOTAPilotPercent != 10 || got.DefaultEnrollmentTTLHours != 24 || got.DefaultJobTTLMinutes != 60 {
		t.Fatalf("invalid values bypassed fallback limits: %+v", got)
	}
}
