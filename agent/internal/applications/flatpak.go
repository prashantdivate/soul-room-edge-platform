package applications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

var (
	flatpakRefPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{2,199}$`)
	flatpakRemotePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	flatpakCommitPattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type FlatpakUpdate struct {
	CampaignID    string `json:"campaign_id"`
	Ref           string `json:"ref"`
	Remote        string `json:"remote"`
	Commit        string `json:"commit,omitempty"`
	RepositoryURL string `json:"repository_url,omitempty"`
}

func ParseFlatpakUpdate(payload []byte) (FlatpakUpdate, error) {
	var update FlatpakUpdate
	if err := json.Unmarshal(payload, &update); err != nil {
		return update, errors.New("invalid Flatpak update payload")
	}
	if update.CampaignID == "" {
		return update, errors.New("Flatpak update is missing its campaign ID")
	}
	if !flatpakRefPattern.MatchString(update.Ref) {
		return update, errors.New("invalid Flatpak reference")
	}
	if !flatpakRemotePattern.MatchString(update.Remote) {
		return update, errors.New("invalid Flatpak remote")
	}
	if update.Commit != "" && !flatpakCommitPattern.MatchString(update.Commit) {
		return update, errors.New("invalid Flatpak OSTree commit")
	}
	if update.RepositoryURL != "" {
		parsed, err := url.Parse(update.RepositoryURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || !strings.HasSuffix(strings.ToLower(parsed.Path), ".flatpakrepo") {
			return update, errors.New("Flatpak repository URL must use HTTPS and point to a .flatpakrepo descriptor")
		}
	}
	return update, nil
}

func RunFlatpakUpdate(ctx context.Context, update FlatpakUpdate) (string, error) {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return "", errors.New("flatpak is not installed on this device")
	}
	var remoteOutput string
	if update.RepositoryURL != "" {
		output, err := exec.CommandContext(ctx, "flatpak", "remote-add", "--system", "--if-not-exists", update.Remote, update.RepositoryURL).CombinedOutput()
		remoteOutput = strings.TrimSpace(string(output))
		if err != nil {
			return remoteOutput, fmt.Errorf("Flatpak repository setup failed: %w", err)
		}
	}
	infoArgs := []string{"remote-info", "--system", "--show-commit"}
	if update.Commit != "" {
		infoArgs = append(infoArgs, "--commit="+update.Commit)
	}
	infoArgs = append(infoArgs, update.Remote, update.Ref)
	availableCommit, err := exec.CommandContext(ctx, "flatpak", infoArgs...).CombinedOutput()
	if err != nil {
		return string(availableCommit), fmt.Errorf("Flatpak remote validation failed: %w", err)
	}
	updateArgs := []string{"update", "--system", "--noninteractive", "--assumeyes"}
	if update.Commit != "" {
		updateArgs = append(updateArgs, "--commit="+update.Commit)
	}
	updateArgs = append(updateArgs, update.Ref)
	output, err := exec.CommandContext(ctx, "flatpak", updateArgs...).CombinedOutput()
	combined := ""
	if remoteOutput != "" {
		combined = "Repository: " + remoteOutput + "\n"
	}
	combined += "Resolved commit: " + strings.TrimSpace(string(availableCommit)) + "\n" + string(output)
	if err != nil {
		return combined, fmt.Errorf("Flatpak update failed: %w", err)
	}
	return combined, nil
}
