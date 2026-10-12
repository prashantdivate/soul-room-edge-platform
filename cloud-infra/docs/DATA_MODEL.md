# Data Model

The canonical PostgreSQL schema is in `migrations/0001_initial.sql`.

Tenant-owned tables include `tenant_id` and indexes for common access patterns.
Flexible JSONB appears only for versioned payloads such as inventory snapshots,
job payloads, application manifests, compatibility metadata, and audit change
summaries.

Core entities:

* organizations
* users, memberships, roles, permissions
* API tokens and sessions
* enrollment tokens and device certificates
* devices, device profiles, device tags, gateways, downstream devices
* telemetry series and samples
* inventory snapshots
* jobs, job attempts, job events
* applications, application versions, artifacts
* deployments and deployment targets
* OTA campaigns
* alerts, alert events, notifications
* audit events
* encrypted AI-provider configurations

The lightweight local store also persists one AI-provider configuration per
tenant. It contains provider metadata, model name, an AES-256-GCM ciphertext,
and a non-secret key hint. Plaintext provider keys and AI conversation text are
not data-model records. The encryption key is stored separately as
`/data/ai-secrets.key` and must be backed up with the application-data volume.
