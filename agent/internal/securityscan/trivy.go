package securityscan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type AdvisoryPackage struct {
	Name             string   `json:"name"`
	InstalledVersion string   `json:"installed_version"`
	FixedVersion     string   `json:"fixed_version,omitempty"`
	HighestSeverity  string   `json:"highest_severity"`
	Critical         int      `json:"critical"`
	High             int      `json:"high"`
	Medium           int      `json:"medium"`
	Low              int      `json:"low"`
	AdvisoryIDs      []string `json:"advisory_ids,omitempty"`
}

type Report struct {
	Scanner   string            `json:"scanner"`
	ScannedAt time.Time         `json:"scanned_at"`
	Total     int               `json:"total"`
	Counts    map[string]int    `json:"counts"`
	Packages  []AdvisoryPackage `json:"packages"`
}

type trivyReport struct {
	Results []struct {
		Vulnerabilities []struct {
			ID               string `json:"VulnerabilityID"`
			Package          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Severity         string `json:"Severity"`
		} `json:"Vulnerabilities"`
	} `json:"Results"`
}

func Scan(ctx context.Context) (Report, error) {
	if _, err := exec.LookPath("trivy"); err != nil {
		return Report{}, errors.New("Trivy is not installed on this device")
	}
	output, err := exec.CommandContext(ctx, "trivy", "rootfs", "--pkg-types", "os", "--scanners", "vuln", "--format", "json", "--quiet", "--disable-telemetry", "/").Output()
	if err != nil {
		return Report{}, fmt.Errorf("Trivy scan failed: %w", err)
	}
	return Parse(output)
}

func Parse(output []byte) (Report, error) {
	var source trivyReport
	if err := json.Unmarshal(output, &source); err != nil {
		return Report{}, errors.New("Trivy returned an invalid JSON report")
	}
	report := Report{Scanner: "trivy", ScannedAt: time.Now().UTC(), Counts: map[string]int{}, Packages: []AdvisoryPackage{}}
	byPackage := map[string]*AdvisoryPackage{}
	for _, result := range source.Results {
		for _, vulnerability := range result.Vulnerabilities {
			severity := strings.ToUpper(vulnerability.Severity)
			report.Total++
			report.Counts[severity]++
			key := vulnerability.Package + "\x00" + vulnerability.InstalledVersion
			pkg := byPackage[key]
			if pkg == nil {
				pkg = &AdvisoryPackage{Name: vulnerability.Package, InstalledVersion: vulnerability.InstalledVersion, FixedVersion: vulnerability.FixedVersion, HighestSeverity: severity}
				byPackage[key] = pkg
			}
			if pkg.FixedVersion == "" {
				pkg.FixedVersion = vulnerability.FixedVersion
			}
			if severityRank(severity) > severityRank(pkg.HighestSeverity) {
				pkg.HighestSeverity = severity
			}
			switch severity {
			case "CRITICAL":
				pkg.Critical++
			case "HIGH":
				pkg.High++
			case "MEDIUM":
				pkg.Medium++
			case "LOW":
				pkg.Low++
			}
			if len(pkg.AdvisoryIDs) < 8 && vulnerability.ID != "" {
				pkg.AdvisoryIDs = append(pkg.AdvisoryIDs, vulnerability.ID)
			}
		}
	}
	for _, pkg := range byPackage {
		report.Packages = append(report.Packages, *pkg)
	}
	sort.Slice(report.Packages, func(i, j int) bool {
		left, right := report.Packages[i], report.Packages[j]
		if severityRank(left.HighestSeverity) != severityRank(right.HighestSeverity) {
			return severityRank(left.HighestSeverity) > severityRank(right.HighestSeverity)
		}
		return left.Name < right.Name
	})
	if len(report.Packages) > 500 {
		report.Packages = report.Packages[:500]
	}
	return report, nil
}

func severityRank(value string) int {
	switch value {
	case "CRITICAL":
		return 5
	case "HIGH":
		return 4
	case "MEDIUM":
		return 3
	case "LOW":
		return 2
	default:
		return 1
	}
}
