package telemetry

import (
	"testing"

	"github.com/soul-room/cloud-infra/internal/protocol"
)

func TestNormalizeRejectsHighCardinality(t *testing.T) {
	labels := map[string]string{}
	for i := 0; i < 13; i++ {
		labels[string(rune('a'+i))] = "x"
	}
	_, err := Normalize("dev1", "msg1", protocol.TelemetryBatch{Metrics: []protocol.Metric{{Name: "cpu", Labels: labels}}})
	if err == nil {
		t.Fatal("expected cardinality rejection")
	}
}
