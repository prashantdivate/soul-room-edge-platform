package location

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/soul-room/edge-agent/internal/protocol"
)

type Config struct {
	Source      string
	Latitude    float64
	Longitude   float64
	Label       string
	GPSDAddress string
	IPURL       string
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
	case "ip":
		return fromIP(ctx, cfg.IPURL, cfg.Label)
	}
	return nil
}

func fromIP(ctx context.Context, endpoint, configuredLabel string) *protocol.Location {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil
	}
	var result struct {
		Success   *bool   `json:"success"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		City      string  `json:"city"`
		Region    string  `json:"region"`
		Country   string  `json:"country"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 64*1024)).Decode(&result) != nil || (result.Success != nil && !*result.Success) || !validCoordinates(result.Latitude, result.Longitude) {
		return nil
	}
	label := strings.TrimSpace(configuredLabel)
	if label == "" {
		parts := make([]string, 0, 3)
		for _, part := range []string{result.City, result.Region, result.Country} {
			if part = strings.TrimSpace(part); part != "" {
				parts = append(parts, part)
			}
		}
		label = strings.Join(parts, ", ")
		if label != "" {
			label += " (network approximate)"
		}
	}
	return &protocol.Location{Latitude: result.Latitude, Longitude: result.Longitude, Source: "ip", Label: label}
}

func validCoordinates(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180 && (latitude != 0 || longitude != 0)
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
