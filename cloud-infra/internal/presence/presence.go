package presence

import "time"

func State(lastSeen time.Time, disabled, decommissioned bool, now time.Time) string {
	switch {
	case decommissioned:
		return "decommissioned"
	case disabled:
		return "disabled"
	case lastSeen.IsZero():
		return "offline"
	case now.Sub(lastSeen) <= 2*time.Minute:
		return "connected"
	case now.Sub(lastSeen) <= 15*time.Minute:
		return "recently_seen"
	case now.Sub(lastSeen) <= 24*time.Hour:
		return "offline"
	default:
		return "stale"
	}
}
