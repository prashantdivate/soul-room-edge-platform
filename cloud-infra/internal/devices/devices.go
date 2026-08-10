package devices

import "github.com/unified-fleet/cloud-infra/internal/model"

func IsGateway(device model.Device) bool {
	for _, capability := range device.Capabilities {
		if capability == "gateway" {
			return true
		}
	}
	return false
}
