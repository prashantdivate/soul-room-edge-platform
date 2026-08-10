# Implementation Status

## Implemented

* Modular Go control-plane foundation with API, gateway, worker, admin, seed
  command entry points.
* Tenant-aware in-process repository and PostgreSQL migration schema.
* Local password authentication with salted PBKDF2-HMAC-SHA256 fallback,
  server-side sessions, API-token model, lockout fields, and audit hooks.
* Centralized RBAC permissions and role bindings.
* Enrollment tokens, development CA, unique device IDs and certificates.
* Device registry, gateway/downstream representation, heartbeat, presence,
  telemetry ingestion, inventory snapshots.
* Typed job creation, expiry, idempotency fields, offline queue status, results.
* Artifacts, applications, deployments, OTA campaigns, alerts, notifications,
  quotas, and audit records in the data model.
* REST/OpenAPI sketch, protobuf contract, Docker Compose, Helm, Terraform,
  backup/restore scripts, and a responsive operations console with distinct
  workflows for every primary navigation area.
* Compose service image can start `control-api`, `device-gateway`, or `worker`
  through per-service commands.

## Simulated or Interface-Only

* S3/MinIO object storage access is represented by artifact metadata and
  deployment manifests; production SDK integration is deferred.
* PostgreSQL migrations are complete for the MVP schema, while local tests use
  the in-process repository.
* OIDC, SAML, external PKI, malware scanning, and notification providers are
  interfaces/docs rather than wired production integrations.
* Remote access is intentionally documentation/interface only.

## Deferred

* Generated protobuf clients.
* Full database driver wiring and migration runner.
* Production Argon2id hasher via `x/crypto` once dependencies are vendored.
* Production end-to-end browser coverage. The current web console uses the real
  development APIs and chart integration and is verified against the Compose
  environment.
