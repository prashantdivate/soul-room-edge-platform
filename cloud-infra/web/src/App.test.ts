import { describe, expect, it } from "vitest";
import type { FleetData } from "./api";
import { navigationFor, resolveShellHubURL } from "./App";

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
