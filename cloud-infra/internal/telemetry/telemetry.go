package telemetry

import (
	"errors"
	"time"

	"github.com/unified-fleet/cloud-infra/internal/model"
	"github.com/unified-fleet/cloud-infra/internal/protocol"
)

func Normalize(deviceID, messageID string, batch protocol.TelemetryBatch) ([]model.Metric, error) {
	if len(batch.Metrics) > 5000 {
		return nil, errors.New("telemetry batch too large")
	}
	out := make([]model.Metric, 0, len(batch.Metrics))
	for _, m := range batch.Metrics {
		if m.Name == "" || len(m.Labels) > 12 {
			return nil, errors.New("invalid metric")
		}
		deviceTime := time.Now()
		if m.TimeUnixNano != 0 {
			deviceTime = time.Unix(0, m.TimeUnixNano)
		}
		out = append(out, model.Metric{
			DeviceID:   deviceID,
			Name:       m.Name,
			Value:      m.Value,
			Unit:       m.Unit,
			Labels:     m.Labels,
			DeviceTime: deviceTime,
			ReceivedAt: time.Now(),
			MessageID:  messageID,
		})
	}
	return out, nil
}
