package containers

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Policy struct {
	AllowPrivileged       bool
	AllowHostNetwork      bool
	AllowedHostPathMounts []string
}

type ComposeApp struct {
	Services map[string]Service `json:"services"`
}

type Service struct {
	Image      string   `json:"image"`
	Managed    bool     `json:"managed"`
	Privileged bool     `json:"privileged"`
	Network    string   `json:"network_mode"`
	Volumes    []string `json:"volumes"`
}

func ValidateCompose(raw []byte, policy Policy) error {
	var app ComposeApp
	if err := json.Unmarshal(raw, &app); err != nil {
		return err
	}
	if len(app.Services) == 0 {
		return fmt.Errorf("compose app must include services")
	}
	for name, svc := range app.Services {
		if !svc.Managed {
			return fmt.Errorf("service %s is not marked platform-managed", name)
		}
		if svc.Image == "" {
			return fmt.Errorf("service %s missing image", name)
		}
		if svc.Privileged && !policy.AllowPrivileged {
			return fmt.Errorf("service %s requests privileged mode", name)
		}
		if (svc.Network == "host" || svc.Network == "container:host") && !policy.AllowHostNetwork {
			return fmt.Errorf("service %s requests host networking", name)
		}
		for _, v := range svc.Volumes {
			host, _, _ := strings.Cut(v, ":")
			if strings.Contains(host, "/var/run/docker.sock") {
				return fmt.Errorf("service %s mounts Docker socket", name)
			}
			if !hostAllowed(host, policy.AllowedHostPathMounts) {
				return fmt.Errorf("service %s mounts disallowed host path %s", name, host)
			}
		}
	}
	return nil
}

func hostAllowed(path string, allowed []string) bool {
	if path == "" || !strings.HasPrefix(path, "/") {
		return true
	}
	for _, p := range allowed {
		if path == p || strings.HasPrefix(path, strings.TrimRight(p, "/")+"/") {
			return true
		}
	}
	return false
}

type Provider interface {
	RuntimeInfo() (RuntimeInfo, error)
	ListManaged() ([]Container, error)
	ApplyCompose([]byte) error
}

type RuntimeInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Container struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	ImageDigest string `json:"image_digest"`
	State       string `json:"state"`
	Health      string `json:"health"`
	Managed     bool   `json:"managed"`
}
