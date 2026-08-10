package location

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/soul-room/edge-agent/internal/protocol"
)

type Config struct {
	Source      string
	Latitude    float64
	Longitude   float64
	Label       string
	GPSDAddress string
}

func Collect(ctx context.Context, cfg Config) *protocol.Location {
	switch cfg.Source {
	case "static":
		return &protocol.Location{Latitude: cfg.Latitude, Longitude: cfg.Longitude, Source: "static", Label: cfg.Label}
	case "gpsd":
		if value := fromGPSD(ctx, cfg.GPSDAddress); value != nil {
			value.Label = cfg.Label
			return value
		}
	}
	return nil
}

func fromGPSD(ctx context.Context, address string) *protocol.Location {
	dialer := net.Dialer{Timeout: 2 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write([]byte("?WATCH={\"enable\":true,\"json\":true};\n")); err != nil {
		return nil
	}
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var report struct {
			Class string  `json:"class"`
			Mode  int     `json:"mode"`
			Lat   float64 `json:"lat"`
			Lon   float64 `json:"lon"`
			EPH   float64 `json:"eph"`
		}
		if json.Unmarshal(scanner.Bytes(), &report) == nil && report.Class == "TPV" && report.Mode >= 2 {
			return &protocol.Location{Latitude: report.Lat, Longitude: report.Lon, AccuracyM: report.EPH, Source: "gpsd"}
		}
	}
	return nil
}
