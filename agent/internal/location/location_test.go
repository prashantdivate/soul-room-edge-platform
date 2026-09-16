package location

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCollectStatic(t *testing.T) {
	got := Collect(context.Background(), Config{Source: "static", Latitude: 18.5204, Longitude: 73.8567, Label: "lab"})
	if got == nil || got.Latitude != 18.5204 || got.Longitude != 73.8567 || got.Source != "static" || got.Label != "lab" {
		t.Fatalf("unexpected static location: %+v", got)
	}
}

func TestCollectDisabled(t *testing.T) {
	if got := Collect(context.Background(), Config{Source: "disabled"}); got != nil {
		t.Fatalf("disabled location should not report coordinates: %+v", got)
	}
}

func TestCollectIPLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"latitude":12.9719,"longitude":77.5937,"city":"Bengaluru","region":"Karnataka","country":"India"}`))
	}))
	defer server.Close()

	got := Collect(context.Background(), Config{Source: "ip", IPURL: server.URL})
	if got == nil || got.Latitude != 12.9719 || got.Longitude != 77.5937 || got.Source != "ip" || got.Label != "Bengaluru, Karnataka, India (network approximate)" {
		t.Fatalf("unexpected IP location: %+v", got)
	}
}
