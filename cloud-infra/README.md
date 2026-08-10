# Axon Control Plane

Server-side fleet-management platform for Linux edge agents, gateways, and
downstream embedded controllers. It supports SaaS and customer-managed
on-premises deployments from the same modular-monolith codebase.

## MVP Scope

Implemented in this repository:

* tenant-aware API server
* local password authentication and server-side sessions
* centralized RBAC checks
* enrollment-token creation and consumption
* development CA and unique per-device certificate issuance
* device registry, presence, heartbeat, inventory, telemetry ingestion
* actual GPSD, fixed-site, and operator-managed device locations with fleet map
* typed jobs and offline-device queue semantics
* tenant-scoped team accounts with built-in roles and password hashing
* audited ShellHub remote-access integration
* audit log
* artifacts, applications, deployments, OS release plans, canary Flatpak updates, alerts and quotas
* protocol contracts, OpenAPI sketch, PostgreSQL migrations
* Docker Compose, Helm skeleton, Terraform skeleton, backup/restore scripts
* responsive React/TypeScript operations console with global fleet search
* device package inventory with optional Trivy advisory results
* self-hosted Flatpak repository descriptors with canary-first rollout

The implementation uses repository interfaces so the runtime can use the local
durable store for development while production deployments use PostgreSQL and
S3-compatible object storage.

## Local Workflow

```text
docker compose up --build -d
```

Then open `http://localhost:3080`. The local Compose stack starts and seeds every
required service automatically; no operating-system-specific scripts are
required.

See `docs/RUNNING_LOCALLY.md` for device connectivity and local credentials.
See `docs/SHELLHUB_INTEGRATION.md` before enabling remote SSH.
