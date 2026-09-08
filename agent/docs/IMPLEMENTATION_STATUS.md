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
* Durable OTA transaction engine with bounded resumable HTTPS downloads,
  SHA-256 and Ed25519 release verification, product/architecture/version
  compatibility, reboot recovery, health confirmation, commit, and rollback.
* Built-in Mender, RAUC, OSTree, SWUpdate, and Flatpak command adapters plus a
  root-owned executable plugin contract for product-specific update systems.
  OSTree supports signed offline static deltas and exact-commit pulls from
  preconfigured HTTPS/GPG-verified remotes, followed by exact boot confirmation.
* Gateway downstream model, simulator connector, and read-only Modbus TCP
  connector.
* CLI commands for status, identity, enrollment, config validation, telemetry,
  queue, jobs, connectors, diagnostics, and version.

## Deferred

* Generated protobuf Go bindings and production gRPC transport.
* SQLite backend and versioned SQL migrations.
* TPM 2.0 and secure-element concrete integrations.
* Full Docker Engine HTTP API implementation beyond policy-safe command hooks.
* Hardware qualification of each native OTA adapter remains a product-image
  responsibility because partition layouts, bootloaders, and health criteria
  differ between devices.
* Native remote shell in `edge-agent`, intentionally excluded; the platform
  delegates optional remote access to a separate ShellHub agent.
* SELinux/AppArmor profiles beyond guidance.
