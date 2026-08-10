# Architecture Decisions

## ADR-001: Modular Monolith First

The MVP starts as a modular Go monolith with separate process entry points. This
avoids premature distributed-system complexity while keeping extraction
boundaries explicit.

## ADR-002: PostgreSQL Primary Store

PostgreSQL is the production system of record. TimescaleDB can be enabled for
telemetry but is optional. Redis is not required for correctness.

## ADR-003: Tenant Context Required

Repository methods require tenant context for tenant-owned records. Optional
PostgreSQL RLS is defense in depth, not the only isolation mechanism.

## ADR-004: Device Protocol Is Stable and External

Device protocol contracts live under `api/proto` and are independent of
internal database models. Runtime JSON envelopes remain compatible with the
agent MVP until generated protobuf clients are wired.

## ADR-005: No Remote Access in MVP

Remote shell and arbitrary forwarding are not implemented. A future interface is
documented and remains isolated from production routes.
