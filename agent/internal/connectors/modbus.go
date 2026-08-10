package connectors

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/unified-fleet/edge-agent/pkg/connectorsdk"
)

var ErrCommandNotAllowed = errors.New("connector command is not allowlisted")

type ModbusRegister struct {
	Name    string
	Address uint16
	Unit    string
	Scale   float64
}

type ModbusEndpoint struct {
	ID        string
	Address   string
	UnitID    byte
	Model     string
	Firmware  string
	Registers []ModbusRegister
}

type ModbusTCPConnector struct {
	GatewayID string
	Timeout   time.Duration
	MinPeriod time.Duration
	Endpoints []ModbusEndpoint
	mu        sync.Mutex
	last      time.Time
}

func (m *ModbusTCPConnector) Name() string { return "modbus-tcp" }

func (m *ModbusTCPConnector) Discover(context.Context) ([]connectorsdk.DownstreamDevice, error) {
	out := make([]connectorsdk.DownstreamDevice, 0, len(m.Endpoints))
	for _, ep := range m.Endpoints {
		out = append(out, connectorsdk.DownstreamDevice{
			ID: ep.ID, ParentGatewayID: m.GatewayID, ConnectorType: m.Name(),
			ProtocolAddress: ep.Address, Model: ep.Model, FirmwareVersion: ep.Firmware,
			LastSeen: time.Now(), Health: "unknown", Capabilities: []string{"read_registers"},
		})
	}
	return out, nil
}

func (m *ModbusTCPConnector) Collect(ctx context.Context, d connectorsdk.DownstreamDevice) ([]connectorsdk.Metric, error) {
	ep, ok := m.endpoint(d.ID)
	if !ok {
		return nil, fmt.Errorf("unknown downstream device %s", d.ID)
	}
	if len(ep.Registers) == 0 {
		return nil, nil
	}
	if m.Timeout == 0 {
		m.Timeout = 2 * time.Second
	}
	if err := m.rateLimit(ctx); err != nil {
		return nil, err
	}
	out := make([]connectorsdk.Metric, 0, len(ep.Registers))
	for _, reg := range ep.Registers {
		val, err := readHoldingRegister(ctx, ep.Address, ep.UnitID, reg.Address, m.Timeout)
		if err != nil {
			return nil, err
		}
		scale := reg.Scale
		if scale == 0 {
			scale = 1
		}
		out = append(out, connectorsdk.Metric{DeviceID: d.ID, Name: reg.Name, Value: float64(val) * scale, Unit: reg.Unit, Time: time.Now()})
	}
	return out, nil
}

func (m *ModbusTCPConnector) AllowedCommands(connectorsdk.DownstreamDevice) []string { return nil }

func (m *ModbusTCPConnector) Invoke(context.Context, connectorsdk.DownstreamDevice, connectorsdk.Command) error {
	return ErrCommandNotAllowed
}

func (m *ModbusTCPConnector) Health(context.Context) error { return nil }

func (m *ModbusTCPConnector) endpoint(id string) (ModbusEndpoint, bool) {
	for _, ep := range m.Endpoints {
		if ep.ID == id {
			return ep, true
		}
	}
	return ModbusEndpoint{}, false
}

func (m *ModbusTCPConnector) rateLimit(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.MinPeriod == 0 {
		m.MinPeriod = 100 * time.Millisecond
	}
	wait := m.MinPeriod - time.Since(m.last)
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	m.last = time.Now()
	return nil
}

func readHoldingRegister(ctx context.Context, addr string, unit byte, register uint16, timeout time.Duration) (uint16, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	req := []byte{0, 1, 0, 0, 0, 6, unit, 3, byte(register >> 8), byte(register), 0, 1}
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}
	res := make([]byte, 11)
	if _, err := io.ReadFull(conn, res); err != nil {
		return 0, err
	}
	if len(res) < 11 || res[7] != 3 || res[8] != 2 {
		return 0, fmt.Errorf("invalid modbus response")
	}
	return binary.BigEndian.Uint16(res[9:11]), nil
}
