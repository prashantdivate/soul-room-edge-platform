package connectors

import (
	"context"
	"time"

	"github.com/unified-fleet/edge-agent/pkg/connectorsdk"
)

type SimulatorConnector struct {
	GatewayID string
}

func (s SimulatorConnector) Name() string { return "simulator" }

func (s SimulatorConnector) Discover(context.Context) ([]connectorsdk.DownstreamDevice, error) {
	return []connectorsdk.DownstreamDevice{
		{ID: s.GatewayID + "-controller-a", ParentGatewayID: s.GatewayID, ConnectorType: s.Name(), ProtocolAddress: "sim://controller-a", Model: "sim-controller", FirmwareVersion: "1.0.0", LastSeen: time.Now(), Health: "healthy", Capabilities: []string{"telemetry"}, CommandAllowlist: []string{"firmware.simulate"}},
		{ID: s.GatewayID + "-meter-c", ParentGatewayID: s.GatewayID, ConnectorType: s.Name(), ProtocolAddress: "sim://meter-c", Model: "sim-meter", FirmwareVersion: "1.0.0", LastSeen: time.Now(), Health: "healthy", Capabilities: []string{"telemetry"}, CommandAllowlist: nil},
	}, nil
}

func (s SimulatorConnector) Collect(ctx context.Context, d connectorsdk.DownstreamDevice) ([]connectorsdk.Metric, error) {
	return []connectorsdk.Metric{{DeviceID: d.ID, Name: "sim.health", Value: 1, Unit: "state", Time: time.Now()}}, nil
}

func (s SimulatorConnector) AllowedCommands(d connectorsdk.DownstreamDevice) []string {
	return append([]string(nil), d.CommandAllowlist...)
}

func (s SimulatorConnector) Invoke(ctx context.Context, d connectorsdk.DownstreamDevice, c connectorsdk.Command) error {
	for _, allowed := range d.CommandAllowlist {
		if c.Name == allowed {
			return nil
		}
	}
	return ErrCommandNotAllowed
}

func (s SimulatorConnector) Health(context.Context) error { return nil }
