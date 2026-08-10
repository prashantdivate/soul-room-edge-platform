package heartbeat

import (
	"time"

	"github.com/soul-room/edge-agent/internal/protocol"
)

var started = time.Now()

func Collect(version string, location *protocol.Location) protocol.Heartbeat {
	return protocol.Heartbeat{
		AgentVersion:  version,
		Health:        "healthy",
		UptimeSeconds: int64(time.Since(started).Seconds()),
		Location:      location,
	}
}
