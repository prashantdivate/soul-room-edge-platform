package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestBasicCollectorReadsLinuxMetrics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	metrics, err := (BasicCollector{}).Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]float64, len(metrics))
	for _, metric := range metrics {
		values[metric.Name] = metric.Value
	}

	for _, name := range []string{
		"system.uptime",
		"system.load1",
		"system.memory.utilization",
		"system.memory.total",
		"filesystem.utilization",
		"filesystem.total",
		"network.receive_bytes_total",
		"system.cpu.utilization",
	} {
		if _, ok := values[name]; !ok {
			t.Errorf("expected Linux metric %q", name)
		}
	}
	for _, name := range []string{"system.cpu.utilization", "system.memory.utilization", "filesystem.utilization"} {
		if value := values[name]; value < 0 || value > 100 {
			t.Errorf("%s = %v, want 0..100", name, value)
		}
	}
	if values["system.memory.total"] <= 0 || values["filesystem.total"] <= 0 {
		t.Error("expected non-zero memory and filesystem capacity")
	}
}
