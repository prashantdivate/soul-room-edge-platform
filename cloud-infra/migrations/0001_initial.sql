CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE organizations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  slug text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  email citext NOT NULL UNIQUE,
  password_hash text NOT NULL,
  activated_at timestamptz,
  mfa_enabled boolean NOT NULL DEFAULT false,
  locked_until timestamptz,
  failed_login_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE roles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  permissions text[] NOT NULL,
  UNIQUE (tenant_id, name)
);

CREATE TABLE memberships (
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id uuid NOT NULL REFERENCES roles(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, user_id)
);

CREATE TABLE sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE api_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  token_hash text NOT NULL UNIQUE,
  scopes text[] NOT NULL,
  expires_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE enrollment_tokens (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  profile_id uuid,
  token_hash text NOT NULL UNIQUE,
  expected_serial text,
  expected_product text,
  max_devices integer NOT NULL DEFAULT 1,
  used_count integer NOT NULL DEFAULT 0,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE device_profiles (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  telemetry_policy jsonb NOT NULL DEFAULT '{}',
  allowed_job_types text[] NOT NULL DEFAULT '{}',
  update_channel text,
  deployment_ring text,
  maintenance_window jsonb NOT NULL DEFAULT '{}',
  connector_policy jsonb NOT NULL DEFAULT '{}',
  configuration_version text NOT NULL DEFAULT '1',
  UNIQUE (tenant_id, name)
);

CREATE TABLE devices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  profile_id uuid REFERENCES device_profiles(id),
  parent_gateway_id uuid REFERENCES devices(id),
  display_name text NOT NULL,
  enrollment_status text NOT NULL,
  certificate_subject text,
  certificate_serial text,
  certificate_expires_at timestamptz,
  last_seen_at timestamptz,
  presence text NOT NULL DEFAULT 'offline',
  agent_version text,
  architecture text,
  os text,
  kernel text,
  hardware_model text,
  serial text,
  capabilities text[] NOT NULL DEFAULT '{}',
  tags text[] NOT NULL DEFAULT '{}',
  location jsonb NOT NULL DEFAULT '{}',
  software_state jsonb NOT NULL DEFAULT '{}',
  update_state text NOT NULL DEFAULT 'idle',
  health_state text NOT NULL DEFAULT 'unknown',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX devices_tenant_presence_idx ON devices (tenant_id, presence);
CREATE UNIQUE INDEX devices_tenant_certificate_idx ON devices (tenant_id, certificate_serial) WHERE certificate_serial IS NOT NULL;

CREATE TABLE device_certificates (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  serial text NOT NULL,
  subject text NOT NULL,
  not_after timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, serial)
);

CREATE TABLE downstream_devices (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  parent_gateway_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  connector_type text NOT NULL,
  protocol_address text NOT NULL,
  model text,
  serial text,
  firmware_version text,
  last_seen_at timestamptz,
  health text NOT NULL DEFAULT 'unknown',
  capabilities text[] NOT NULL DEFAULT '{}',
  approved_commands text[] NOT NULL DEFAULT '{}',
  UNIQUE (tenant_id, parent_gateway_id, connector_type, protocol_address)
);

CREATE TABLE telemetry_series (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  metric_name text NOT NULL,
  unit text,
  labels jsonb NOT NULL DEFAULT '{}',
  UNIQUE (tenant_id, device_id, metric_name, labels)
);

CREATE TABLE telemetry_samples (
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  series_id uuid NOT NULL REFERENCES telemetry_series(id) ON DELETE CASCADE,
  device_time timestamptz NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  value double precision NOT NULL,
  message_id text NOT NULL,
  PRIMARY KEY (tenant_id, series_id, device_time, message_id)
);

CREATE TABLE inventory_snapshots (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  schema_version text NOT NULL,
  payload jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  deployment_id uuid,
  type text NOT NULL,
  payload jsonb NOT NULL,
  payload_digest text NOT NULL,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  timeout_seconds integer NOT NULL,
  approval_status text NOT NULL DEFAULT 'approved',
  attempt integer NOT NULL DEFAULT 0,
  execution_window jsonb NOT NULL DEFAULT '{}',
  state text NOT NULL,
  result jsonb NOT NULL DEFAULT '{}'
);

CREATE INDEX jobs_tenant_device_state_idx ON jobs (tenant_id, device_id, state);

CREATE TABLE job_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  job_id uuid NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  state text NOT NULL,
  message text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE artifacts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  version text NOT NULL,
  object_key text NOT NULL,
  digest text NOT NULL,
  signature text,
  content_type text NOT NULL,
  size_bytes bigint NOT NULL,
  architecture text,
  product text,
  hardware_compatibility jsonb NOT NULL DEFAULT '{}',
  uploaded_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, name, version)
);

CREATE TABLE applications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  version text NOT NULL,
  compose_manifest jsonb NOT NULL,
  required_architecture text,
  image_digests text[] NOT NULL DEFAULT '{}',
  compatibility jsonb NOT NULL DEFAULT '{}',
  UNIQUE (tenant_id, name, version)
);

CREATE TABLE deployments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  target_selector jsonb NOT NULL,
  rollout_policy jsonb NOT NULL,
  state text NOT NULL,
  created_by uuid REFERENCES users(id),
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE deployment_targets (
  deployment_id uuid NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  device_id uuid NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  state text NOT NULL,
  last_error text,
  PRIMARY KEY (deployment_id, device_id)
);

CREATE TABLE ota_campaigns (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  artifact_id uuid REFERENCES artifacts(id),
	name text NOT NULL,
	adapter text NOT NULL,
	version text NOT NULL,
	product text,
	architecture text,
	artifact_url text,
	artifact_size bigint,
	digest text,
	signature text,
	signing_key_id text,
	compatible_from jsonb NOT NULL DEFAULT '[]'::jsonb,
	target_device_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
	canary_percent integer NOT NULL CHECK (canary_percent BETWEEN 1 AND 100),
  metadata jsonb NOT NULL,
  state text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE alerts (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name text NOT NULL,
  type text NOT NULL,
  severity text NOT NULL,
  rule jsonb NOT NULL,
  enabled boolean NOT NULL DEFAULT true
);

CREATE TABLE alert_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  alert_id uuid REFERENCES alerts(id),
  device_id uuid REFERENCES devices(id),
  state text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  type text NOT NULL,
  target text NOT NULL,
  secret_ref text,
  rate_limit_per_minute integer NOT NULL DEFAULT 60,
  dead_letter_count integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid REFERENCES organizations(id) ON DELETE SET NULL,
  actor_id uuid,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text,
  result text NOT NULL,
  source_ip inet,
  user_agent text,
  correlation_id text,
  summary jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_tenant_time_idx ON audit_events (tenant_id, created_at DESC);
