package transport

type DeviceAuthMode string

const (
	DeviceAuthEnrollmentToken DeviceAuthMode = "enrollment_token"
	DeviceAuthMTLS            DeviceAuthMode = "mtls"
)

type Compatibility struct {
	SupportedVersions []string `json:"supported_versions"`
	SelectedVersion   string   `json:"selected_version"`
}
