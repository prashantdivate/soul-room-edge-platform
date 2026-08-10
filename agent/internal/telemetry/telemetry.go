package telemetry

import (
	"bufio"
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/soul-room/edge-agent/internal/protocol"
)

type Collector interface {
	Name() string
	Collect(context.Context) ([]protocol.Metric, error)
}

type BasicCollector struct{}

func (BasicCollector) Name() string { return "linux-system" }

func (BasicCollector) Collect(ctx context.Context) ([]protocol.Metric, error) {
	now := time.Now().UnixNano()
	metrics := []protocol.Metric{
		{Name: "agent.goroutines", Value: float64(runtime.NumGoroutine()), Unit: "count", TimeUnixNano: now},
		{Name: "system.uptime", Value: readUptime(), Unit: "seconds", TimeUnixNano: now},
	}
	if load := readLoadAvg(); load >= 0 {
		metrics = append(metrics, protocol.Metric{Name: "system.load1", Value: load, Unit: "load", TimeUnixNano: now})
	}
	if used, total := readMemory(); total > 0 {
		metrics = append(metrics,
			protocol.Metric{Name: "system.memory.utilization", Value: used * 100 / total, Unit: "percent", TimeUnixNano: now},
			protocol.Metric{Name: "system.memory.used", Value: used, Unit: "bytes", TimeUnixNano: now},
			protocol.Metric{Name: "system.memory.total", Value: total, Unit: "bytes", TimeUnixNano: now},
		)
	}
	if total, free := fsUsage("/"); total > 0 {
		metrics = append(metrics,
			protocol.Metric{Name: "filesystem.utilization", Value: float64(total-free) * 100 / float64(total), Unit: "percent", Labels: map[string]string{"mount": "/"}, TimeUnixNano: now},
			protocol.Metric{Name: "filesystem.total", Value: float64(total), Unit: "bytes", Labels: map[string]string{"mount": "/"}, TimeUnixNano: now},
			protocol.Metric{Name: "filesystem.free", Value: float64(free), Unit: "bytes", Labels: map[string]string{"mount": "/"}, TimeUnixNano: now},
		)
	}
	if rx, tx := readNetworkTotals(); rx >= 0 {
		metrics = append(metrics,
			protocol.Metric{Name: "network.receive_bytes_total", Value: rx, Unit: "bytes", TimeUnixNano: now},
			protocol.Metric{Name: "network.transmit_bytes_total", Value: tx, Unit: "bytes", TimeUnixNano: now},
		)
	}
	if temperature := readTemperature(); temperature >= 0 {
		metrics = append(metrics, protocol.Metric{Name: "system.temperature", Value: temperature, Unit: "celsius", TimeUnixNano: now})
	}
	if utilization := readCPUUtilization(ctx); utilization >= 0 {
		metrics = append(metrics, protocol.Metric{Name: "system.cpu.utilization", Value: utilization, Unit: "percent", TimeUnixNano: time.Now().UnixNano()})
	}
	return metrics, nil
}

func CollectAll(ctx context.Context, collectors []Collector) protocol.TelemetryBatch {
	var out []protocol.Metric
	for _, collector := range collectors {
		select {
		case <-ctx.Done():
			return protocol.TelemetryBatch{Metrics: out}
		default:
		}
		metrics, err := collector.Collect(ctx)
		if err != nil {
			out = append(out, protocol.Metric{Name: "collector.error", Value: 1, Unit: "count", Labels: map[string]string{"collector": collector.Name()}, TimeUnixNano: time.Now().UnixNano()})
			continue
		}
		out = append(out, metrics...)
	}
	return protocol.TelemetryBatch{Metrics: out}
}

func readCPUUtilization(ctx context.Context) float64 {
	totalA, idleA, ok := readCPUStat()
	if !ok {
		return -1
	}
	timer := time.NewTimer(200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return -1
	case <-timer.C:
	}
	totalB, idleB, ok := readCPUStat()
	if !ok || totalB <= totalA {
		return -1
	}
	deltaTotal := float64(totalB - totalA)
	return (deltaTotal - float64(idleB-idleA)) * 100 / deltaTotal
}

func readCPUStat() (uint64, uint64, bool) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	fields := strings.Fields(strings.SplitN(string(b), "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	var total uint64
	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		values = append(values, value)
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return total, idle, true
}

func readMemory() (float64, float64) {
	values := map[string]float64{}
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, _ := strconv.ParseFloat(fields[1], 64)
		values[strings.TrimSuffix(fields[0], ":")] = value * 1024
	}
	total := values["MemTotal"]
	available := values["MemAvailable"]
	return total - available, total
}

func readUptime() float64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return value
}

func readLoadAvg() float64 {
	f, err := os.Open("/proc/loadavg")
	if err != nil {
		return -1
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return -1
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) == 0 {
		return -1
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return -1
	}
	return value
}

func readNetworkTotals() (float64, float64) {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return -1, -1
	}
	var rx, tx float64
	for _, line := range strings.Split(string(b), "\n") {
		name, values, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) == "lo" {
			continue
		}
		fields := strings.Fields(values)
		if len(fields) < 9 {
			continue
		}
		received, _ := strconv.ParseFloat(fields[0], 64)
		transmitted, _ := strconv.ParseFloat(fields[8], 64)
		rx += received
		tx += transmitted
	}
	return rx, tx
}

func readTemperature() float64 {
	for _, path := range []string{"/sys/class/thermal/thermal_zone0/temp", "/sys/class/hwmon/hwmon0/temp1_input"} {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
		if err == nil {
			if value > 1000 {
				value /= 1000
			}
			return value
		}
	}
	return -1
}

func fsUsage(path string) (uint64, uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0
	}
	return stat.Blocks * uint64(stat.Bsize), stat.Bavail * uint64(stat.Bsize)
}
