package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soul-room/cloud-infra/internal/config"
	"github.com/soul-room/cloud-infra/internal/model"
	"github.com/soul-room/cloud-infra/internal/storage"
)

func TestDeviceStatusVerifiesRegistration(t *testing.T) {
	store, err := storage.Open("")
	if err != nil {
		t.Fatal(err)
	}
	device, err := store.CreateDevice(model.Device{TenantID: "tenant-1", DisplayName: "device-1"})
	if err != nil {
		t.Fatal(err)
	}
	app := New(config.Config{}, store, nil)

	request := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	request.Header.Set("X-Tenant-ID", device.TenantID)
	request.Header.Set("X-Device-ID", device.ID)
	response := httptest.NewRecorder()
	app.DeviceHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected registered device status 200, got %d: %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	request.Header.Set("X-Tenant-ID", device.TenantID)
	request.Header.Set("X-Device-ID", "missing-device")
	response = httptest.NewRecorder()
	app.DeviceHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected unknown device status 401, got %d", response.Code)
	}
}
