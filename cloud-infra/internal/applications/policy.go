package applications

import (
	"encoding/json"
	"errors"
	"strings"
)

type ComposeService struct {
	Image      string   `json:"image"`
	Managed    bool     `json:"managed"`
	Privileged bool     `json:"privileged"`
	Network    string   `json:"network_mode"`
	PID        string   `json:"pid"`
	IPC        string   `json:"ipc"`
	Volumes    []string `json:"volumes"`
	Devices    []string `json:"devices"`
	CapAdd     []string `json:"cap_add"`
}

type ComposeManifest struct {
	Services map[string]ComposeService `json:"services"`
}

func ValidateCompose(raw json.RawMessage, allowExceptions bool) error {
	var manifest ComposeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if len(manifest.Services) == 0 {
		return errors.New("compose manifest must include services")
	}
	for name, svc := range manifest.Services {
		if name == "" || svc.Image == "" {
			return errors.New("service name and image are required")
		}
		if !svc.Managed {
			return errors.New("all services must be platform-managed")
		}
		if allowExceptions {
			continue
		}
		if svc.Privileged || svc.Network == "host" || svc.PID == "host" || svc.IPC == "host" {
			return errors.New("privileged mode and host namespaces are rejected")
		}
		if len(svc.Devices) > 0 || len(svc.CapAdd) > 0 {
			return errors.New("device passthrough and added capabilities are rejected")
		}
		for _, volume := range svc.Volumes {
			host, _, _ := strings.Cut(volume, ":")
			if strings.Contains(host, "/var/run/docker.sock") {
				return errors.New("docker socket mounts are rejected")
			}
			if strings.HasPrefix(host, "/") && !strings.HasPrefix(host, "/var/lib/soul-room/managed/") {
				return errors.New("unrestricted host mounts are rejected")
			}
		}
	}
	return nil
}
