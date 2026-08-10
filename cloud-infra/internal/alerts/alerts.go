package alerts

import "time"

type Rule struct {
	Type      string        `json:"type"`
	Severity  string        `json:"severity"`
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
	Cooldown  time.Duration `json:"cooldown"`
}

func RecoveryState(active bool) string {
	if active {
		return "firing"
	}
	return "resolved"
}
