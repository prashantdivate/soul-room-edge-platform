# Implementation Plan

## Assumptions

* The initial control plane for local development is a mock HTTPS server shipped
  as `cmd/device-simulator`.
* The first transport is HTTPS with JSON payloads and a versioned protobuf
  contract in `api/proto/edge/v1/edge.proto`. Generated protobuf bindings are
  deferred until the control-plane repository is aligned.
* The default embedded store is a crash-safe bounded file store with the same
  queue semantics expected from the future SQLite adapter. A SQLite-backed
  implementation remains the production target.
* Container operations are policy-validated through a provider interface. The
  default provider shells out to Docker only when an administrator has Docker
  installed and enabled.
* OTA uses a shared transaction engine with built-in OSTree, Mender, RAUC,
  SWUpdate, and Flatpak adapters plus a local plugin contract.

## Phase 1

* Create repository foundation and command layout.
* Implement versioned configuration with safe defaults and validation.
* Generate local Ed25519 device keys with restrictive permissions.
* Implement development enrollment using a local CA simulator.
* Implement HTTPS transport with certificate validation, client identity loading,
  message IDs, sequence counters, timestamps, and reconnect backoff.
* Collect basic hardware, OS, network, storage, uptime, and agent telemetry.
* Persist outbound data in a bounded durable queue.
* Add unit tests and local simulator tests.

## Phase 2

* Implement typed job framework with validation, expiry, idempotency, timeout,
  cancellation hooks, bounded output, structured results, and persisted state.
* Implement restart-agent, reboot-device simulation, diagnostics, safe file
  deployment, service restart policy, inventory collection, and configuration
  application hooks.

## Phase 3

* Implement Docker provider abstraction and policy validator.
* Support list, inspect, pull, start, stop, restart, remove, and restricted
  Compose validation for managed containers only.
* Report runtime, container, image digest, and health state.

## Phase 4

* Implement gateway downstream-device model.
* Implement simulator connector.
* Implement Modbus TCP read-only connector with allowlisted register reads,
  timeouts, rate limits, health, tests, and no arbitrary writes.

## Phase 5

* Implement signed OTA downloads, adapter discovery, persistent reboot recovery,
  native install/commit/rollback operations, credential rotation interfaces,
  migration hooks, hardening docs, and packaging.

## Completion Criteria Tracking

See `docs/IMPLEMENTATION_STATUS.md`.
