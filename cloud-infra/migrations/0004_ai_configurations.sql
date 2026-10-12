CREATE TABLE ai_configurations (
  tenant_id uuid PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
  provider text NOT NULL CHECK (provider IN ('openai', 'anthropic', 'compatible')),
  base_url text NOT NULL,
  model text NOT NULL,
  api_key_ciphertext text,
  key_hint text,
  enabled boolean NOT NULL DEFAULT true,
  updated_by uuid REFERENCES users(id),
  updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE ai_configurations ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_ai_configurations ON ai_configurations
  USING (tenant_id::text = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id::text = current_setting('app.tenant_id', true));
