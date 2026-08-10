ALTER TABLE enrollment_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_certificates ENABLE ROW LEVEL SECURITY;
ALTER TABLE downstream_devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE telemetry_series ENABLE ROW LEVEL SECURITY;
ALTER TABLE telemetry_samples ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_snapshots ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE artifacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE deployment_targets ENABLE ROW LEVEL SECURITY;
ALTER TABLE ota_campaigns ENABLE ROW LEVEL SECURITY;
ALTER TABLE alerts ENABLE ROW LEVEL SECURITY;
ALTER TABLE alert_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;

-- Applications set app.tenant_id per transaction. RLS is defense in depth; the
-- Go repositories still require explicit tenant context.
CREATE POLICY tenant_isolation_devices ON devices
  USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_jobs ON jobs
  USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_telemetry_samples ON telemetry_samples
  USING (tenant_id::text = current_setting('app.tenant_id', true));
CREATE POLICY tenant_isolation_audit ON audit_events
  USING (tenant_id::text = current_setting('app.tenant_id', true));
