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
