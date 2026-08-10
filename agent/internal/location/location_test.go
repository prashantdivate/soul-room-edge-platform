package location

import (
	"context"
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
