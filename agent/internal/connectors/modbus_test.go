package connectors

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/unified-fleet/edge-agent/pkg/connectorsdk"
)

func TestModbusReadOnlyConnector(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 12)
		_, _ = conn.Read(buf)
		_, _ = conn.Write([]byte{0, 1, 0, 0, 0, 5, 1, 3, 2, 0, 42})
	}()
	c := &ModbusTCPConnector{GatewayID: "gw1", Timeout: time.Second, Endpoints: []ModbusEndpoint{{ID: "meter1", Address: ln.Addr().String(), UnitID: 1, Registers: []ModbusRegister{{Name: "energy", Address: 1, Unit: "kwh"}}}}}
	devs, err := c.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	metrics, err := c.Collect(context.Background(), devs[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(metrics) != 1 || metrics[0].Value != 42 {
		t.Fatalf("unexpected metrics %+v", metrics)
	}
	if err := c.Invoke(context.Background(), devs[0], structCommand("write")); err == nil {
		t.Fatal("writes must be rejected")
	}
}

func structCommand(name string) connectorsdk.Command {
	return connectorsdk.Command{Name: name}
}
