package artifacts

import "errors"

type UploadPolicy struct {
	MaxBytes int64
}

func ValidateUpload(size int64, digest, contentType string, policy UploadPolicy) error {
	if size <= 0 || digest == "" || contentType == "" {
		return errors.New("artifact size, digest, and content type are required")
	}
	if policy.MaxBytes > 0 && size > policy.MaxBytes {
		return errors.New("artifact exceeds tenant quota")
	}
	return nil
}
