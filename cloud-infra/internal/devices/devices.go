package devices

import "github.com/soul-room/cloud-infra/internal/model"

func IsGateway(device model.Device) bool {
	for _, capability := range device.Capabilities {
		if capability == "gateway" {
			return true
		}
	}
	return false
}
