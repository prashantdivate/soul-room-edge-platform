# Implementation Status

## Implemented

* Repository layout, Makefile, CI, systemd, Debian metadata, Yocto recipe.
* Config defaults and validation.
* File-backed device identity with Ed25519 keys and restrictive permissions.
* Development enrollment against `cmd/device-simulator`.
* HTTPS transport with TLS validation, message envelopes, sequence numbers,
  timestamps, and exponential backoff with jitter.
* Heartbeat, inventory, and telemetry collection.
* Bounded durable queue with priority-aware drops and age pruning.
* Typed job framework with validation, expiry, idempotency, result persistence,
  bounded output, timeouts, and handlers.
* Safe file deployment with destination allowlist, checksum verification,
  staging, backup, atomic rename, rollback, and audit records.
* Docker provider abstraction and policy validation.
* OTA adapter interface and simulator adapter.
* Gateway downstream model, simulator connector, and read-only Modbus TCP
  connector.
* CLI commands for status, identity, enrollment, config validation, telemetry,
  queue, jobs, connectors, diagnostics, and version.

## Deferred

* Generated protobuf Go bindings and production gRPC transport.
* SQLite backend and versioned SQL migrations.
* TPM 2.0 and secure-element concrete integrations.
* Full Docker Engine HTTP API implementation beyond policy-safe command hooks.
* Production OSTree, Mender, RAUC, SWUpdate, and MCU bootloader adapters.
* Production remote access tunnel, intentionally excluded from MVP.
* SELinux/AppArmor profiles beyond guidance.
