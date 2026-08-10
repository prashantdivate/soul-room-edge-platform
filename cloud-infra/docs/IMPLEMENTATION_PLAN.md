# Implementation Plan

## Assumptions

* MVP is a modular monolith with isolated command entry points: `control-api`,
  `device-gateway`, `worker`, `adminctl`, and `dev-seed`.
* PostgreSQL is the production database target. The code also includes a durable
  local JSON repository for development and tests where a database is not
  available.
* Device communication starts with HTTPS JSON envelopes compatible with the
  agent MVP and versioned protobuf contracts under `api/proto`. gRPC generated
  bindings are a later compatibility artifact.
* Object storage is S3-compatible. Local/on-premises default is MinIO.
* Remote shell and arbitrary port forwarding are intentionally out of scope.

## Phase 1: Foundation

* Repository structure, docs, protocol, OpenAPI, migrations, CI.
* Configuration, structured logs, health endpoints.
* Tenancy model, auth, sessions, RBAC, audit.

## Phase 2: Device Foundation

* Enrollment tokens, local CA, device certificates, device registry.
* Device gateway endpoints for enrollment, hello, heartbeat, inventory,
  telemetry, jobs, results, and presence.
* Device protocol compatibility tests.

## Phase 3: Monitoring

* Telemetry ingestion with tenant/device validation, payload limits, cardinality
  controls, deduplication, retention fields, and query APIs.
* Inventory snapshots, fleet overview, device detail APIs and UI pages.

## Phase 4: Operations

* Typed jobs with idempotency, expiry, offline delivery, cancellation,
  concurrency controls, and audit history.
* Profiles, safe file deployment job type, and deployment planning.

## Phase 5: Applications

* Artifact records, S3 object references, immutable versions, digests, signatures
  and quotas.
* Restricted Compose application model and staged deployment state.

## Phase 6: Gateways and Controllers

* Gateway and downstream-device registry, connector health, controller
  telemetry, downstream jobs, and parent-child UI navigation.

## Phase 7: OTA

* OTA campaign metadata, signatures, compatibility rules, deployment rings,
  simulator state machine, rollback reporting.

## Phase 8: Production Packaging

* Docker Compose, Helm, Terraform, backup/restore, upgrade/rollback, on-premises
  docs, security hardening, and offline installation guidance.
