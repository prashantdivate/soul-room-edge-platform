import { describe, expect, it } from "vitest";
import type { FleetData } from "./api";
import { navigationFor, resolveShellHubURL, telemetryChart, trivySetupCommands } from "./App";
import { packageVisualKind } from "./PackageBrandIcon";

function pagesFor(role: string, permissions: string[]) {
  const data = { membership: { role }, roles: { [role]: permissions } } as FleetData;
  return navigationFor(data).flatMap((group) => group.pages.map((page) => page.id));
}

describe("permission-aware navigation", () => {
  it("keeps fleet analysis visible while hiding unavailable administration", () => {
    const pages = pagesFor("read_only_viewer", ["device.read", "telemetry.read"]);
    expect(pages).toContain("devices");
    expect(pages).toContain("health");
    expect(pages).not.toContain("remote");
    expect(pages).not.toContain("users");
    expect(pages).not.toContain("audit");
    expect(pages).not.toContain("onprem");
  });
});

describe("ShellHub browser URL", () => {
  it("always maps bundled ShellHub to the interface serving Soul Room", () => {
    expect(resolveShellHubURL("http://old-interface.local:8088", "http://active-interface.local:3080/remote", true)).toBe("http://active-interface.local:8088/");
  });

  it("preserves the configured host for an external ShellHub service", () => {
    expect(resolveShellHubURL("https://shell.example.com", "https://fleet.example.com/remote", false)).toBe("https://shell.example.com/");
  });
});

describe("device presentation", () => {
  it("preserves fractional telemetry instead of flattening quiet devices", () => {
    const data = { metrics: [
      { device_id: "dev-1", name: "system.cpu.utilization", value: 1.25, unit: "percent", device_time: "2026-09-16T06:40:12Z", received_at: "2026-09-16T06:40:12Z" },
      { device_id: "dev-1", name: "system.cpu.utilization", value: 2.47, unit: "percent", device_time: "2026-09-16T06:40:42Z", received_at: "2026-09-16T06:40:42Z" },
    ] } as FleetData;
    expect(telemetryChart(data)[0].cpu).toBeCloseTo(1.86);
  });

  it("uses only upstream artwork and never presents categories as logos", () => {
    expect(packageVisualKind("apache2")).toBe("brand");
    expect(packageVisualKind("adwaita-icon-theme")).toBe("brand");
    expect(packageVisualKind("alsa-utils")).toBe("system");
    expect(packageVisualKind("unknown-system-package")).toBe("system");
  });
});

describe("advisory scan guidance", () => {
  it("keeps device install, scan verification, and agent restart commands visible", () => {
    expect(trivySetupCommands.debian).toContain("sudo apt-get install -y trivy");
    expect(trivySetupCommands.verify).toContain("trivy rootfs --pkg-types os --scanners vuln");
    expect(trivySetupCommands.restartAgent).toContain("edge-agentctl -config /etc/edge-agent/config.yaml status");
  });
});
