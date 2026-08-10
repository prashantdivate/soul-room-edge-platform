package security

import "errors"

type Quotas struct {
	DevicesPerTenant      int
	EnrollmentTokens      int
	APIRequestsPerMinute  int
	TelemetryBytesPerDay  int64
	TelemetryCardinality  int
	ArtifactStorageBytes  int64
	ArtifactSizeBytes     int64
	ConcurrentJobs        int
	ConcurrentDeployments int
	NotificationRate      int
	DiagnosticBundleBytes int64
}

func CheckLimit(used, limit int) error {
	if limit > 0 && used >= limit {
		return ErrQuotaExceeded
	}
	return nil
}

var ErrQuotaExceeded = errors.New("quota exceeded")
