export type Membership = { tenant_id: string; user_id: string; role: string };

export type Device = {
  id: string;
  tenant_id: string;
  profile_id?: string;
  parent_gateway_id?: string;
  display_name: string;
  enrollment_status: string;
  certificate_expires_at?: string;
  last_seen_at?: string;
  presence: string;
  agent_version?: string;
  architecture?: string;
  os?: string;
  kernel?: string;
  hardware_model?: string;
  serial?: string;
  capabilities?: string[];
  tags?: string[];
  update_state?: string;
  health_state?: string;
  location?: DeviceLocation;
  remote_access_id?: string;
};

export type DeviceLocation = {
  latitude: number;
  longitude: number;
  accuracy_m?: number;
  source: string;
  label?: string;
  updated_at?: string;
};

export type Downstream = {
  id: string;
  parent_gateway_id: string;
  connector_type: string;
  protocol_address: string;
  model?: string;
  serial?: string;
  firmware_version?: string;
  last_seen_at?: string;
  health: string;
  capabilities?: string[];
};

export type Metric = {
  device_id: string;
  name: string;
  value: number;
  unit: string;
  device_time: string;
  received_at: string;
};

export type Job = {
  id: string;
  device_id: string;
  type: string;
  state: string;
  approval_status: string;
  created_at: string;
  expires_at: string;
  result?: unknown;
};

export type Artifact = {
  id: string;
  name: string;
  version: string;
  digest: string;
  signature?: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
};

export type AuditEvent = {
  id: string;
  actor_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  result: string;
  created_at: string;
};

export type EnrollmentToken = {
  id: string;
  profile_id?: string;
  token?: string;
  used_count: number;
  max_devices: number;
  expires_at: string;
  created_at: string;
};

export type User = { id: string; email: string; role: string; mfa_enabled: boolean; activated_at: string; created_at: string };
export type Organization = { id: string; name: string; slug: string };
export type InstalledPackage = { name: string; version: string; architecture?: string; source: string };
export type InventoryFacts = Record<string, unknown> & { installed_packages?: InstalledPackage[] };
export type Inventory = Record<string, InventoryFacts>;
export type PlatformInfo = { device_gateway_endpoint: string; remote_access_provider: string; remote_access_url: string; remote_access_configured: boolean; remote_access_managed: boolean; remote_access_ssh_port: number };

export type OTACampaign = {
  id: string;
  name: string;
  artifact_id?: string;
  artifact_url: string;
  version: string;
  product: string;
  architecture: string;
  digest: string;
  signature: string;
  signing_key_id?: string;
  artifact_size?: number;
  compatible_from?: string[];
  adapter: string;
  flatpak_ref?: string;
  flatpak_remote?: string;
  flatpak_commit?: string;
  flatpak_repository_url?: string;
  target_device_ids: string[];
  canary_percent: number;
  state: string;
  created_at: string;
};

export type FleetData = {
  membership: Membership;
  organization: Organization;
  devices: Device[];
  downstream: Downstream[];
  metrics: Metric[];
  jobs: Job[];
  artifacts: Artifact[];
  audit: AuditEvent[];
  enrollmentTokens: EnrollmentToken[];
  users: User[];
  inventory: Inventory;
  roles: Record<string, string[]>;
  applications: Record<string, unknown>[];
  deployments: Record<string, unknown>[];
  otaCampaigns: OTACampaign[];
  platform: PlatformInfo;
};

class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...(options.headers || {}) },
    ...options,
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(response.status, body?.error?.message || `Request failed (${response.status})`);
  }
  return response.json() as Promise<T>;
}

export async function login(email: string, password: string): Promise<Membership> {
  const response = await request<{ memberships: Membership[] }>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  if (!response.memberships[0]) throw new Error("This account does not belong to an organization.");
  return response.memberships[0];
}

export async function logout(): Promise<void> {
  await request<{ status: string }>("/api/v1/auth/logout", { method: "POST" });
}

export async function loadFleet(membership: Membership): Promise<FleetData> {
  const headers = { "X-Tenant-ID": membership.tenant_id };
  const get = <T,>(path: string) => request<T>(path, { headers });
  const [orgs, devices, downstream, jobs, artifacts, audit, tokens, users, inventory, roles, applications, deployments, ota, platform] = await Promise.all([
    get<{ organizations: Organization[] }>("/api/v1/organizations"),
    get<{ devices: Device[] }>("/api/v1/devices"),
    get<{ downstream: Downstream[] }>("/api/v1/downstream"),
    get<{ jobs: Job[] }>("/api/v1/jobs"),
    get<{ artifacts: Artifact[] }>("/api/v1/artifacts"),
    get<{ audit: AuditEvent[] }>("/api/v1/audit"),
    get<{ enrollment_tokens: EnrollmentToken[] }>("/api/v1/enrollment-tokens"),
    get<{ users: User[] }>("/api/v1/users"),
    get<{ inventory: Inventory }>("/api/v1/inventory"),
    get<Record<string, string[]>>("/api/v1/roles"),
    get<{ applications: Record<string, unknown>[] }>("/api/v1/applications"),
    get<{ deployments: Record<string, unknown>[] }>("/api/v1/deployments"),
    get<{ ota_campaigns: OTACampaign[] }>("/api/v1/ota"),
    get<PlatformInfo>("/api/v1/platform-info"),
  ]);
  const deviceList = devices.devices || [];
  const metrics = (
    await Promise.all(deviceList.map((device) => get<{ metrics: Metric[] | null }>(`/api/v1/telemetry?device_id=${encodeURIComponent(device.id)}`)))
  ).flatMap((response) => response.metrics || []);
  return {
    membership,
    organization: (orgs.organizations || []).find((org) => org.id === membership.tenant_id) || (orgs.organizations || [])[0],
    devices: deviceList,
    downstream: downstream.downstream || [],
    metrics,
    jobs: jobs.jobs || [],
    artifacts: artifacts.artifacts || [],
    audit: audit.audit || [],
    enrollmentTokens: tokens.enrollment_tokens || [],
    users: users.users || [],
    inventory: inventory.inventory || {},
    roles: roles || {},
    applications: applications.applications || [],
    deployments: deployments.deployments || [],
    otaCampaigns: ota.ota_campaigns || [],
    platform,
  };
}

export async function createJob(membership: Membership, deviceId: string, type: string): Promise<Job> {
  return request<Job>("/api/v1/jobs", {
    method: "POST",
    headers: { "X-Tenant-ID": membership.tenant_id, "Idempotency-Key": crypto.randomUUID() },
    body: JSON.stringify({ device_id: deviceId, type, payload: { requested_from: "web-console" }, ttl: "1h" }),
  });
}

export async function createEnrollmentToken(membership: Membership, profileId: string, ttl: string): Promise<EnrollmentToken> {
  return request<EnrollmentToken>("/api/v1/enrollment-tokens", {
    method: "POST",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify({ profile_id: profileId, ttl }),
  });
}

export async function updateDevice(membership: Membership, deviceId: string, patch: { location?: DeviceLocation; remote_access_id?: string }): Promise<Device> {
  return request<Device>("/api/v1/devices", {
    method: "PATCH",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify({ device_id: deviceId, ...patch }),
  });
}

export async function deleteDevice(membership: Membership, deviceId: string): Promise<void> {
	await request<{ status: string }>(`/api/v1/devices?device_id=${encodeURIComponent(deviceId)}`, {
		method: "DELETE",
		headers: { "X-Tenant-ID": membership.tenant_id },
	});
}

export async function deleteJob(membership: Membership, jobId: string): Promise<void> {
	await request<{ status: string }>(`/api/v1/jobs?job_id=${encodeURIComponent(jobId)}`, {
		method: "DELETE",
		headers: { "X-Tenant-ID": membership.tenant_id },
	});
}

export async function createRemoteAccess(membership: Membership, deviceId: string, user: string): Promise<{ provider: string; launch_url: string; ssh_command: string }> {
  return request("/api/v1/remote-access", {
    method: "POST",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify({ device_id: deviceId, user }),
  });
}

export async function createOTACampaign(membership: Membership, campaign: Omit<OTACampaign, "id" | "state" | "created_at">): Promise<OTACampaign> {
  return request<OTACampaign>("/api/v1/ota", {
    method: "POST",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify(campaign),
  });
}

export async function promoteOTACampaign(membership: Membership, campaignId: string): Promise<OTACampaign> {
  return request<OTACampaign>("/api/v1/ota", {
    method: "PATCH",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify({ campaign_id: campaignId, action: "promote" }),
  });
}

export async function createTeamUser(membership: Membership, email: string, password: string, role: string): Promise<User> {
  return request<User>("/api/v1/users", {
    method: "POST",
    headers: { "X-Tenant-ID": membership.tenant_id },
    body: JSON.stringify({ email, password, role }),
  });
}

export function exportJson(filename: string, value: unknown) {
  const blob = new Blob([JSON.stringify(value, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export async function exportPdf(filename: string, element: HTMLElement) {
	const [{ default: html2canvas }, { jsPDF }] = await Promise.all([import("html2canvas"), import("jspdf")]);
	element.classList.add("pdfExporting");
	try {
		const canvas = await html2canvas(element, { scale: 2, backgroundColor: "#ffffff", useCORS: true, logging: false });
		const orientation = canvas.width > canvas.height ? "landscape" : "portrait";
		const pdf = new jsPDF({ orientation, unit: "mm", format: "a4", compress: true });
		const margin = 8;
		const pageWidth = pdf.internal.pageSize.getWidth() - margin * 2;
		const pageHeight = pdf.internal.pageSize.getHeight() - margin * 2;
		const imageHeight = canvas.height * pageWidth / canvas.width;
		const image = canvas.toDataURL("image/jpeg", 0.92);
		let offset = 0;
		do {
			if (offset > 0) pdf.addPage();
			pdf.addImage(image, "JPEG", margin, margin - offset, pageWidth, imageHeight, undefined, "FAST");
			offset += pageHeight;
		} while (offset < imageHeight);
		pdf.save(filename);
	} finally {
		element.classList.remove("pdfExporting");
	}
}

export { ApiError };
