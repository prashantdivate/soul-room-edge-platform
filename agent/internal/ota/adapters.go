package ota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/soul-room/edge-agent/internal/applications"
)

func NewAdapter(name, pluginDir string, runner Runner) (Adapter, error) {
	if adapterNamePattern.MatchString(name) {
		path := filepath.Join(pluginDir, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() && info.Mode().Perm()&0111 != 0 {
			return &pluginAdapter{name: name, path: path}, nil
		}
	}
	switch name {
	case "flatpak", "mender", "ostree", "rauc", "swupdate":
		return &nativeAdapter{name: name, runner: runner}, nil
	default:
		if !adapterNamePattern.MatchString(name) {
			return nil, errors.New("invalid OTA adapter name")
		}
		path := filepath.Join(pluginDir, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Mode().Perm()&0111 == 0 {
			return nil, fmt.Errorf("OTA adapter %q is not installed", name)
		}
		return &pluginAdapter{name: name, path: path}, nil
	}
}

type nativeAdapter struct {
	name   string
	runner Runner
}

func (a *nativeAdapter) Name() string           { return a.name }
func (a *nativeAdapter) ArtifactRequired() bool { return a.name != "flatpak" }
func (a *nativeAdapter) RequiresReboot() bool   { return a.name != "flatpak" }

func (a *nativeAdapter) Available() bool {
	command := a.name
	if a.name == "mender" {
		command = "mender-update"
	}
	if _, err := a.runner.LookPath(command); err == nil {
		return true
	}
	if a.name == "mender" {
		_, err := a.runner.LookPath("mender")
		return err == nil
	}
	return false
}

func (a *nativeAdapter) Check(ctx context.Context, request Request, artifact string) error {
	if !a.Available() {
		return fmt.Errorf("%s is not installed on this device", a.name)
	}
	var err error
	switch a.name {
	case "flatpak":
		_, err = applications.ParseFlatpakUpdate(flatpakPayload(request))
	case "rauc":
		_, err = a.runner.Run(ctx, "rauc", "info", artifact)
	case "ostree":
		_, err = a.runner.Run(ctx, "ostree", "static-delta", "verify", artifact)
	case "swupdate":
		_, err = a.runner.Run(ctx, "swupdate", "--check", "-i", artifact)
	case "mender":
		if _, lookupErr := a.runner.LookPath("mender-artifact"); lookupErr == nil {
			_, err = a.runner.Run(ctx, "mender-artifact", "read", artifact)
		}
	}
	return err
}

func (a *nativeAdapter) Install(ctx context.Context, request Request, artifact string) error {
	switch a.name {
	case "flatpak":
		update, err := applications.ParseFlatpakUpdate(flatpakPayload(request))
		if err != nil {
			return err
		}
		if update.RepositoryURL != "" {
			if _, err := a.runner.Run(ctx, "flatpak", "remote-add", "--system", "--if-not-exists", update.Remote, update.RepositoryURL); err != nil {
				return err
			}
		}
		infoArgs := []string{"remote-info", "--system", "--show-commit"}
		if update.Commit != "" {
			infoArgs = append(infoArgs, "--commit="+update.Commit)
		}
		infoArgs = append(infoArgs, update.Remote, update.Ref)
		if _, err := a.runner.Run(ctx, "flatpak", infoArgs...); err != nil {
			return err
		}
		updateArgs := []string{"update", "--system", "--noninteractive", "--assumeyes"}
		if update.Commit != "" {
			updateArgs = append(updateArgs, "--commit="+update.Commit)
		}
		updateArgs = append(updateArgs, update.Ref)
		_, err = a.runner.Run(ctx, "flatpak", updateArgs...)
		return err
	case "mender":
		command := "mender-update"
		if _, err := a.runner.LookPath(command); err != nil {
			command = "mender"
		}
		_, err := a.runner.Run(ctx, command, "install", artifact)
		return err
	case "rauc":
		_, err := a.runner.Run(ctx, "rauc", "install", artifact)
		return err
	case "ostree":
		_, err := a.runner.Run(ctx, "ostree", "static-delta", "apply-offline", artifact)
		return err
	case "swupdate":
		_, err := a.runner.Run(ctx, "swupdate", "-i", artifact)
		return err
	default:
		return errors.New("unsupported native OTA adapter")
	}
}

func (a *nativeAdapter) Activate(ctx context.Context, request Request) error {
	if a.name != "ostree" {
		return nil
	}
	args := []string{"admin", "deploy"}
	if request.Product != "" {
		args = append(args, "--os="+request.Product)
	}
	args = append(args, request.Version)
	_, err := a.runner.Run(ctx, "ostree", args...)
	return err
}

func (a *nativeAdapter) Commit(ctx context.Context, request Request) error {
	switch a.name {
	case "mender":
		command := "mender-update"
		if _, err := a.runner.LookPath(command); err != nil {
			command = "mender"
		}
		_, err := a.runner.Run(ctx, command, "commit")
		return err
	case "rauc":
		_, err := a.runner.Run(ctx, "rauc", "status", "mark-good")
		return err
	default:
		return nil
	}
}

func (a *nativeAdapter) Rollback(ctx context.Context, request Request, previous string) error {
	switch a.name {
	case "flatpak":
		if previous == "" {
			return errors.New("previous Flatpak commit is unknown")
		}
		_, err := a.runner.Run(ctx, "flatpak", "update", "--system", "--noninteractive", "--assumeyes", "--commit="+previous, request.FlatpakRef)
		return err
	case "mender":
		command := "mender-update"
		if _, err := a.runner.LookPath(command); err != nil {
			command = "mender"
		}
		_, err := a.runner.Run(ctx, command, "rollback")
		return err
	case "rauc":
		_, err := a.runner.Run(ctx, "rauc", "status", "mark-bad")
		return err
	case "ostree":
		if !regexp.MustCompile(`^[a-fA-F0-9]{64}$`).MatchString(previous) {
			return errors.New("previous OSTree deployment checksum is unavailable")
		}
		args := []string{"admin", "deploy"}
		if request.Product != "" {
			args = append(args, "--os="+request.Product)
		}
		args = append(args, previous)
		_, err := a.runner.Run(ctx, "ostree", args...)
		return err
	case "swupdate":
		return errors.New("SWUpdate rollback requires a device-specific plugin adapter")
	default:
		return errors.New("rollback is not implemented")
	}
}

func (a *nativeAdapter) CurrentVersion(ctx context.Context, request Request) (string, error) {
	switch a.name {
	case "flatpak":
		return a.runner.Run(ctx, "flatpak", "info", "--system", "--show-commit", request.FlatpakRef)
	case "mender":
		command := "mender-update"
		if _, err := a.runner.LookPath(command); err != nil {
			command = "mender"
		}
		output, err := a.runner.Run(ctx, command, "show-provides")
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(output, "\n") {
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), "rootfs-image.version="); ok {
				return strings.TrimSpace(value), nil
			}
		}
		return strings.TrimSpace(output), nil
	case "rauc":
		return a.runner.Run(ctx, "rauc", "status", "--detailed")
	case "ostree":
		output, err := a.runner.Run(ctx, "ostree", "admin", "status")
		if err != nil {
			return "", err
		}
		checksumPattern := regexp.MustCompile(`[a-fA-F0-9]{64}`)
		for _, line := range strings.Split(output, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "*") {
				if checksum := checksumPattern.FindString(line); checksum != "" {
					return checksum, nil
				}
			}
		}
		return checksumPattern.FindString(output), nil
	case "swupdate":
		return "", nil
	default:
		return "", errors.New("unsupported native OTA adapter")
	}
}

func flatpakPayload(request Request) []byte {
	b, _ := json.Marshal(applications.FlatpakUpdate{CampaignID: request.UpdateID, Ref: request.FlatpakRef, Remote: request.FlatpakRemote, Commit: request.FlatpakCommit, RepositoryURL: request.RepositoryURL})
	return b
}

type pluginAdapter struct{ name, path string }

func (a *pluginAdapter) Name() string           { return a.name }
func (a *pluginAdapter) ArtifactRequired() bool { return true }
func (a *pluginAdapter) Available() bool        { return true }
func (a *pluginAdapter) RequiresReboot() bool {
	out, err := exec.Command(a.path, "requires-reboot").Output()
	return err != nil || strings.TrimSpace(string(out)) != "false"
}
func (a *pluginAdapter) invoke(ctx context.Context, action string, request Request, artifact, previous string) (string, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, a.path, action)
	cmd.Stdin = strings.NewReader(string(payload))
	cmd.Env = append(os.Environ(), "SOULROOM_OTA_ARTIFACT="+artifact, "SOULROOM_OTA_PREVIOUS_VERSION="+previous)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), fmt.Errorf("adapter %s %s failed: %w", a.name, action, err)
	}
	return strings.TrimSpace(string(output)), nil
}
func (a *pluginAdapter) Check(ctx context.Context, r Request, p string) error {
	_, err := a.invoke(ctx, "check", r, p, "")
	return err
}
func (a *pluginAdapter) Install(ctx context.Context, r Request, p string) error {
	_, err := a.invoke(ctx, "install", r, p, "")
	return err
}
func (a *pluginAdapter) Activate(ctx context.Context, r Request) error {
	_, err := a.invoke(ctx, "activate", r, "", "")
	return err
}
func (a *pluginAdapter) Commit(ctx context.Context, r Request) error {
	_, err := a.invoke(ctx, "commit", r, "", "")
	return err
}
func (a *pluginAdapter) Rollback(ctx context.Context, r Request, previous string) error {
	_, err := a.invoke(ctx, "rollback", r, "", previous)
	return err
}
func (a *pluginAdapter) CurrentVersion(ctx context.Context, r Request) (string, error) {
	return a.invoke(ctx, "version", r, "", "")
}
