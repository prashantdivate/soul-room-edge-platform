package settings

import (
	"errors"
	"net/url"
	"strings"

	"github.com/soul-room/cloud-infra/internal/model"
)

func Validate(value model.PlatformSettings) error {
	value.OrganizationName = strings.TrimSpace(value.OrganizationName)
	if len(value.OrganizationName) < 2 || len(value.OrganizationName) > 80 {
		return errors.New("organization name must contain 2 to 80 characters")
	}
	if value.CompanyDomain != "" {
		domain := strings.ToLower(strings.TrimSpace(value.CompanyDomain))
		parsed, err := url.Parse("https://" + domain)
		if err != nil || parsed.Hostname() != domain || !strings.Contains(domain, ".") {
			return errors.New("company domain must be a hostname such as example.com")
		}
	}
	if value.ShellHubURL != "" {
		portal, err := url.Parse(strings.TrimSpace(value.ShellHubURL))
		if err != nil || (portal.Scheme != "http" && portal.Scheme != "https") || portal.Hostname() == "" || portal.User != nil {
			return errors.New("ShellHub portal URL must be an absolute HTTP or HTTPS URL")
		}
	}
	if value.ShellHubSSHPort < 1 || value.ShellHubSSHPort > 65535 {
		return errors.New("ShellHub SSH port must be between 1 and 65535")
	}
	if value.DeviceOfflineMinutes < 2 || value.DeviceOfflineMinutes > 1440 {
		return errors.New("offline window must be between 2 and 1440 minutes")
	}
	if value.DefaultTelemetryWindow != "15m" && value.DefaultTelemetryWindow != "1h" && value.DefaultTelemetryWindow != "24h" {
		return errors.New("telemetry window must be 15m, 1h, or 24h")
	}
	if value.DefaultOTAPilotPercent < 1 || value.DefaultOTAPilotPercent > 100 {
		return errors.New("pilot percentage must be between 1 and 100")
	}
	if value.DefaultEnrollmentTTLHours < 1 || value.DefaultEnrollmentTTLHours > 720 {
		return errors.New("enrollment token lifetime must be between 1 and 720 hours")
	}
	if value.DefaultJobTTLMinutes < 5 || value.DefaultJobTTLMinutes > 10080 {
		return errors.New("job expiry must be between 5 and 10080 minutes")
	}
	return nil
}
