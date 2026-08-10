package connectorsdk

import (
	"context"
	"time"
)

type DownstreamDevice struct {
	ID              string            `json:"id"`
	ParentGatewayID string            `json:"parent_gateway_id"`
	ConnectorType   string            `json:"connector_type"`
	ProtocolAddress string            `json:"protocol_address"`
	Model           string            `json:"model"`
	FirmwareVersion string            `json:"firmware_version"`
	LastSeen        time.Time         `json:"last_seen"`
	Health          string            `json:"health"`
	Capabilities    []string          `json:"capabilities"`
	CommandAllowlist []string          `json:"command_allowlist"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type Metric struct {
	DeviceID string            `json:"device_id"`
	Name     string            `json:"name"`
	Value    float64           `json:"value"`
	Unit     string            `json:"unit"`
	Labels   map[string]string `json:"labels,omitempty"`
	Time     time.Time         `json:"time"`
}

type Command struct {
	Name    string         `json:"name"`
	Payload map[string]any `json:"payload"`
}

type Connector interface {
	Name() string
	Discover(context.Context) ([]DownstreamDevice, error)
	Collect(context.Context, DownstreamDevice) ([]Metric, error)
	AllowedCommands(DownstreamDevice) []string
	Invoke(context.Context, DownstreamDevice, Command) error
	Health(context.Context) error
}
