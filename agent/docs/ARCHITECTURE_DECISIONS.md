# Architecture Decisions

## ADR-001: Linux Agent Written in Go

Go is used for the Linux agent, CLI, simulator, connectors, and tests because it
cross-compiles cleanly for `linux/amd64` and `linux/arm64`, has strong TLS and
crypto libraries, and produces simple static-ish deployment artifacts.

## ADR-002: Outbound HTTPS First, gRPC Later

The first runtime transport is versioned HTTPS with JSON envelopes. The protocol
contract is defined in protobuf under `api/proto`, but generated bindings are
deferred until the cloud control-plane API stabilizes. This keeps the MVP
interoperable without weakening the protocol boundaries.

## ADR-003: Local Key Generation

Each device creates its own Ed25519 private key during enrollment. The private
key is stored locally with `0600` permissions and is never transmitted. The
identity package exposes a signer interface so TPM 2.0 or secure-element-backed
keys can replace file-backed keys.

## ADR-004: No Remote Shell in MVP

The MVP intentionally excludes unrestricted remote shell and tunnel execution.
Future remote access must use short-lived credentials, scoped authorization,
session expiry, no PTY on tunnel accounts, no forwarding, rate limits, approval
workflows where required, and complete audit logs.

## ADR-005: Typed Jobs Only

Jobs use explicit types and schemas. Generic shell execution is disabled by
default and not registered in the production handler set. Job IDs, tenant IDs,
device IDs, expiry, attempts, scopes, payload digests, and state are validated
before execution.

## ADR-006: Conservative Connector Model

Downstream connectors default to monitoring only. Modbus TCP supports
allowlisted register reads and normalized metrics; arbitrary writes are not
implemented. Safety-critical commands require connector-specific capability
declarations.

## ADR-007: Durable Queue Abstraction

The storage package exposes queue semantics compatible with a future SQLite
backend. The current MVP backend is file based to avoid unvendored native
dependencies in the bootstrap repository. Queue bounds, priorities, age
expiration, and corruption handling are implemented and tested.
