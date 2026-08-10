package securityscan

import "testing"

func TestParseAggregatesPackageAdvisories(t *testing.T) {
	report, err := Parse([]byte(`{"Results":[{"Vulnerabilities":[{"VulnerabilityID":"CVE-1","PkgName":"openssl","InstalledVersion":"1.0","FixedVersion":"1.1","Severity":"HIGH"},{"VulnerabilityID":"CVE-2","PkgName":"openssl","InstalledVersion":"1.0","FixedVersion":"1.1","Severity":"CRITICAL"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || len(report.Packages) != 1 || report.Packages[0].Critical != 1 || report.Packages[0].High != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
