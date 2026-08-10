package gateway

import (
	"context"

	"github.com/soul-room/edge-agent/pkg/connectorsdk"
)

type Gateway struct {
	ID         string
	Connectors []connectorsdk.Connector
}

func (g Gateway) Discover(ctx context.Context) ([]connectorsdk.DownstreamDevice, error) {
	var out []connectorsdk.DownstreamDevice
	for _, c := range g.Connectors {
		devices, err := c.Discover(ctx)
		if err != nil {
			return out, err
		}
		out = append(out, devices...)
	}
	return out, nil
}

func (g Gateway) Collect(ctx context.Context) ([]connectorsdk.Metric, error) {
	var out []connectorsdk.Metric
	for _, c := range g.Connectors {
		devices, err := c.Discover(ctx)
		if err != nil {
			return out, err
		}
		for _, d := range devices {
			metrics, err := c.Collect(ctx, d)
			if err != nil {
				continue
			}
			out = append(out, metrics...)
		}
	}
	return out, nil
}
