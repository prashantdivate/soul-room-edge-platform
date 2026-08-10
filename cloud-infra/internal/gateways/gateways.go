package gateways

import "github.com/soul-room/cloud-infra/internal/model"

func Children(parent model.Device, downstream []model.DownstreamDevice) []model.DownstreamDevice {
	out := []model.DownstreamDevice{}
	for _, d := range downstream {
		if d.ParentGatewayID == parent.ID && d.TenantID == parent.TenantID {
			out = append(out, d)
		}
	}
	return out
}
