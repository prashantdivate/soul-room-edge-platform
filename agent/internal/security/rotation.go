package security

import (
	"context"
	"time"
)

type RotationRequest struct {
	TenantID    string    `json:"tenant_id"`
	DeviceID    string    `json:"device_id"`
	RequestedAt time.Time `json:"requested_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Reason      string    `json:"reason"`
}

type RotationResult struct {
	DeviceID       string    `json:"device_id"`
	RotatedAt      time.Time `json:"rotated_at"`
	NewKeyID       string    `json:"new_key_id"`
	CertificatePEM string    `json:"certificate_pem"`
}

type KeyRotator interface {
	PrepareRotation(context.Context, RotationRequest) error
	CommitRotation(context.Context, RotationRequest) (RotationResult, error)
	RollbackRotation(context.Context, RotationRequest) error
}

func ValidateRotationRequest(now time.Time, req RotationRequest) error {
	if req.TenantID == "" || req.DeviceID == "" {
		return ErrInvalidRotation
	}
	if req.ExpiresAt.IsZero() || now.After(req.ExpiresAt) {
		return ErrExpiredRotation
	}
	return nil
}

type rotationError string

func (e rotationError) Error() string { return string(e) }

const (
	ErrInvalidRotation rotationError = "invalid credential rotation request"
	ErrExpiredRotation rotationError = "expired credential rotation request"
)
