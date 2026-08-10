package jobs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/soul-room/cloud-infra/internal/model"
)

var AllowedTypes = map[string]bool{
	"reboot":                         true,
	"restart_service":                true,
	"collect_diagnostics":            true,
	"collect_logs":                   true,
	"collect_inventory":              true,
	"scan_vulnerabilities":           true,
	"deploy_file":                    true,
	"apply_configuration":            true,
	"container_action":               true,
	"compose_application_deployment": true,
	"ota_stage":                      true,
	"ota_activate":                   true,
	"flatpak_update":                 true,
	"downstream_connector_command":   true,
}

func New(deviceID, jobType string, payload json.RawMessage, createdBy string, ttl time.Duration) (model.Job, error) {
	if !AllowedTypes[jobType] {
		return model.Job{}, errors.New("unsupported job type")
	}
	if ttl <= 0 {
		return model.Job{}, errors.New("job ttl must be positive")
	}
	sum := sha256.Sum256(payload)
	return model.Job{
		DeviceID:       deviceID,
		Type:           jobType,
		Payload:        payload,
		PayloadDigest:  hex.EncodeToString(sum[:]),
		CreatedBy:      createdBy,
		ExpiresAt:      time.Now().Add(ttl),
		TimeoutSeconds: 600,
		State:          "queued",
		ApprovalStatus: "approved",
	}, nil
}

func ValidateResult(job model.Job, result json.RawMessage) error {
	if job.ID == "" || len(result) > 128*1024 {
		return errors.New("invalid job result")
	}
	return nil
}
