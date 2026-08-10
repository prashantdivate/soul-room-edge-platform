package deployments

type RolloutPolicy struct {
	CanaryPercent     int    `json:"canary_percent"`
	MaxConcurrent     int    `json:"max_concurrent"`
	FailureThreshold  int    `json:"failure_threshold"`
	SuccessThreshold  int    `json:"success_threshold"`
	MaintenanceWindow string `json:"maintenance_window,omitempty"`
}

func ShouldPause(failures, total int, policy RolloutPolicy) bool {
	if total == 0 || policy.FailureThreshold == 0 {
		return false
	}
	return failures*100/total >= policy.FailureThreshold
}
